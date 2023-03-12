package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/easypmnt/checkout-api/auth"
	"github.com/easypmnt/checkout-api/internal/kitlog"
	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/payment"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/easypmnt/checkout-api/server"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/easypmnt/checkout-api/webhook"
	"github.com/easypmnt/checkout-api/websocketrpc"
	"github.com/go-chi/oauth"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	_ "github.com/lib/pq" // init pg driver
)

func main() {
	// Init logger
	logger := logrus.WithFields(logrus.Fields{
		"app":       appName,
		"build_tag": buildTagRuntime,
	})

	// Errgroup with context
	eg, ctx := errgroup.WithContext(newCtx(logger))

	// Init DB connection
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		logger.WithError(err).Fatal("failed to init db connection")
	}
	defer db.Close()

	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)

	if err := db.Ping(); err != nil {
		logger.WithError(err).Fatal("failed to ping db")
	}

	// Init repository
	repo, err := repository.NewWithConnection(ctx, db)
	if err != nil {
		logger.WithError(err).Fatal("failed to init repository")
	}

	// Redis connect options for asynq client
	redisConnOpt := asynq.RedisClientOpt{
		Network:      redisNetwork,
		Addr:         redisConnAddr,
		Username:     redisUsername,
		Password:     redisPassword,
		DB:           redisDB,
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
		PoolSize:     redisPoolSize,
	}

	// Init asynq client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisConnAddr,
		PoolSize: redisPoolSize,
	})
	defer asynqClient.Close()

	// Init Solana client
	solClient := solana.NewClient(
		solana.WithRPCEndpoint(solanaRPCEndpoint),
	)

	// Init Jupiter client
	jupiterClient := jupiter.NewClient()

	// Init HTTP router
	r := initRouter(logger)

	// OAuth2 Middleware
	oauthMdw := oauth.Authorize(oauthSigningKey, nil)

	// webhook enqueuer
	webhookEnqueuer := webhook.NewEnqueuer(asynqClient)

	// Payment worker enqueuer
	paymentEnqueuer := payment.NewEnqueuer(asynqClient)

	// Setup event listener
	wsConn := openWebsocketConnection(ctx, solanaWSSEndpoint, logger, eg)
	eventClient := websocketrpc.NewClient(wsConn,
		websocketrpc.WithEventHandler(
			websocketrpc.EventAccountNotification,
			func(base58Addr string, _ json.RawMessage) error {
				return paymentEnqueuer.CheckPaymentByReference(ctx, base58Addr)
			},
		),
	)

	// Payment service
	paymentService := payment.NewService(
		repo, solClient, jupiterClient,
		payment.WithSolanaPayBaseURI(solanaPayBaseURI),
		payment.WithDefaultMerchantWalletAddress(merchantWalletAddress),
		payment.WithDefaultMerchantApplyBonus(merchantApplyBonus),
		payment.WithDefaultMerchantMaxBonusPerc(uint16(merchantMaxBonusPercentage)),
		payment.WithDefaultMerchantBonusMintAddr(bonusMintAddress),
		payment.WithDefaultMerchantBonusMintAuthority(bonusMintAuthority),
		payment.WithDefaultMerchantBonusRate(uint64(bonusRate)),
		payment.WithWebhookEnqueuer(webhookEnqueuer),
		payment.WithEventClient(eventClient),
	)

	// Mount HTTP endpoints
	{
		// oauth service
		r.Mount("/oauth", auth.MakeHTTPHandler(
			auth.NewOAuth2Server(
				oauthSigningKey,
				accessTokenTTL,
				auth.NewVerifier(
					repo,
					clientID,
					clientSecret,
					auth.WithAccessTokenTTL(accessTokenTTL),
					auth.WithRefreshTokenTTL(refreshTokenTTL),
				),
			),
		))

		// payment service
		r.Mount("/payment", server.MakeHTTPHandler(
			server.MakeEndpoints(
				paymentService,
				server.Config{
					AppName:    productName,
					AppIconURI: productIconURI,
				},
			),
			kitlog.NewLogger(logger), oauthMdw,
		))
	}

	// Run HTTP server
	eg.Go(runServer(ctx, httpPort, r, logger))

	// Run asynq worker
	eg.Go(runQueueServer(
		redisConnOpt,
		logger,
		payment.NewWorker(paymentService, eventClient),
		webhook.NewWorker(webhook.NewService(
			webhook.WithSignatureSecret(webhookSignatureSecret),
			webhook.WithWebhookURI(webhookURI),
		)),
	))

	// Run asynq scheduler
	eg.Go(runScheduler(
		redisConnOpt,
		logger,
		// TODO: add scheduler workers
	))

	// Run event listener
	eg.Go(func() error {
		return eventClient.Run(ctx)
	})

	// Run all goroutines
	if err := eg.Wait(); err != nil {
		logger.WithError(err).Fatal("error occurred")
	}

	time.Sleep(5 * time.Second) // wait for all goroutines to finish
	logger.Info("server successfuly shutdown")
}

// newCtx creates a new context that is cancelled when an interrupt signal is received.
func newCtx(log *logrus.Entry) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		sCh := make(chan os.Signal, 1)
		signal.Notify(sCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGPIPE)
		<-sCh

		// Shutdown signal with grace period of N seconds (default: 5 seconds)
		shutdownCtx, shutdownCtxCancel := context.WithTimeout(ctx, httpServerShutdownTimeout)
		defer shutdownCtxCancel()

		<-shutdownCtx.Done()
		if shutdownCtx.Err() == context.DeadlineExceeded {
			log.Error("shutdown timeout exceeded")
		}
	}()
	return ctx
}
