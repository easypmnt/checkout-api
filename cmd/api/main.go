package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/easypmnt/checkout-api/auth"
	"github.com/easypmnt/checkout-api/events"
	"github.com/easypmnt/checkout-api/internal/kitlog"
	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/payments"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/easypmnt/checkout-api/server"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/easypmnt/checkout-api/webhook"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/oauth"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	_ "github.com/lib/pq" // init pg driver
)

func main() {
	// Init logger
	logrus.SetReportCaller(false)
	logger := logrus.WithFields(logrus.Fields{
		"app":       appName,
		"build_tag": buildTagRuntime,
	})
	if appDebug {
		logger.Logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.Logger.SetLevel(logrus.InfoLevel)
	}

	defer func() { logger.Info("server successfully shutdown") }()

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
	repo, err := repository.Prepare(ctx, db)
	if err != nil {
		logger.WithError(err).Fatal("failed to init repository")
	}

	// Init event emitter
	eventEmitter := events.NewEmitter(logger)

	// Redis connect options for asynq client
	redisConnOpt, err := asynq.ParseRedisURI(redisConnString)
	if err != nil {
		logger.WithError(err).Fatal("failed to parse redis connection string")
	}

	// Init asynq client
	asynqClient := asynq.NewClient(redisConnOpt)
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
	paymentEnqueuer := payments.NewEnqueuer(asynqClient)

	// Setup event listener
	// wsConn := openWebsocketConnection(ctx, solanaWSSEndpoint, logger, eg)
	// websocketrpcClient := websocketrpc.NewClient(wsConn,
	// 	websocketrpc.WithEventsEmitter(eventEmitter),
	// )

	var paymentService payments.PaymentService
	// Payment service
	paymentService = payments.NewService(
		repo, solClient, jupiterClient,
		payments.Config{
			ApplyBonus:           merchantApplyBonus,
			BonusMintAddress:     bonusMintAddress,
			BonusAuthAccount:     bonusMintAuthority,
			MaxApplyBonusAmount:  uint64(maxApplyBonusAmount),
			MaxApplyBonusPercent: uint16(merchantMaxBonusPercentage),
			AccrueBonus:          bonusRate > 0,
			AccrueBonusRate:      uint64(bonusRate),
			DestinationMint:      merchantDefaultMint,
			DestinationWallet:    merchantWalletAddress,
			PaymentTTL:           paymentTTL,
			SolPayBaseURL:        solanaPayBaseURI,
		},
	)
	// Events decorator
	paymentService = payments.NewServiceEvents(paymentService, eventEmitter.Emit)
	// Logging decorator
	paymentService = payments.NewServiceLogger(paymentService, logger)

	// Init sse service
	// sseService := sse.NewService(sse.NewMemStorage())

	// Event listener
	eventEmitter.On(events.TransactionUpdated, payments.UpdateTransactionStatusListener(paymentService))
	eventEmitter.On(events.TransactionCreated, payments.TransactionCreatedListener(paymentService, paymentEnqueuer))
	eventEmitter.On(
		events.TransactionReferenceNotification,
		payments.ReferenceAccountNotificationListener(paymentService, paymentEnqueuer),
	)
	eventEmitter.ListenEvents(
		webhook.TranslateEventsToWebhookEvents(webhookEnqueuer),
		events.AllEvents...,
	)
	// eventEmitter.ListenEvents(
	// 	sse.TranslateEventsToSSEChannel(sseService),
	// 	events.AllEvents...,
	// )

	// Event broadcaster
	eventBroadcaster := events.NewEventBroadcaster(eventEmitter, logger)

	// Mount HTTP endpoints
	{
		// oauth service
		r.With(middleware.Timeout(httpRequestTimeout)).
			Mount("/oauth", auth.MakeHTTPHandler(
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
		r.With(middleware.Timeout(httpRequestTimeout)).
			Mount("/payment", server.MakeHTTPHandler(
				server.MakeEndpoints(
					paymentService,
					jupiterClient,
					server.Config{
						AppName:    productName,
						AppIconURI: productIconURI,
					},
				),
				kitlog.NewLogger(logger),
				oauthMdw,
			))

		// sse service
		r.With(middleware.Timeout(time.Hour)).
			Mount("/ws", events.MakeHTTPHandler(eventBroadcaster))
	}

	// Run HTTP server
	eg.Go(runServer(ctx, httpPort, r, logger))

	// Run asynq worker
	eg.Go(runQueueServer(
		redisConnOpt,
		logger,
		payments.NewWorker(paymentService, solClient, paymentEnqueuer),
		webhook.NewWorker(webhook.NewService(
			webhook.WithSignatureSecret(webhookSignatureSecret),
			webhook.WithWebhookURI(webhookURI),
		)),
	))

	// Run asynq scheduler
	eg.Go(runScheduler(
		redisConnOpt,
		logger,
		payments.NewScheduler(),
	))

	// Run event broadcaster
	eg.Go(func() error {
		return eventBroadcaster.Run(ctx)
	})

	// Run event listener
	// eg.Go(func() error {
	// 	return websocketrpcClient.Run(ctx)
	// })

	// Run all goroutines
	if err := eg.Wait(); err != nil {
		logger.WithError(err).Fatal("error occurred")
	}
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
