package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/easypmnt/checkout-api/auth"
	"github.com/easypmnt/checkout-api/payment"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/easypmnt/checkout-api/server"
	"github.com/go-chi/oauth"
	kitlog "github.com/go-kit/log"
	"github.com/hibiken/asynq"

	_ "github.com/lib/pq" // init pg driver
)

func main() {
	// Setup go-kit logger
	var logger kitlog.Logger
	{
		logger = kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stdout))
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
		logger = kitlog.With(logger, "build", buildTagRuntime)
		logger = kitlog.With(logger, "app", appName)

		log.SetOutput(kitlog.NewStdlibAdapter(logger))
	}

	// Global app context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init DB connection
	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		logger.Log("error", err, "msg", "failed to open db connection")
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)

	if err := db.Ping(); err != nil {
		logger.Log("error", err, "msg", "failed to ping db")
		os.Exit(1)
	}

	// Init repository
	repo, err := repository.NewWithConnection(ctx, db)
	if err != nil {
		logger.Log("error", err, "msg", "failed to init repository")
		os.Exit(1)
	}

	// Init asynq client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisConnAddr,
		PoolSize: redisPoolSize,
	})
	defer asynqClient.Close()

	// Init HTTP router
	r := initRouter(logger)

	// OAuth2 Middleware
	oauthMdw := oauth.Authorize(oauthSigningKey, nil)

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
				payment.NewService(repo, nil, nil),
				server.Config{
					AppName:    productName,
					AppIconURI: productIconURI,
				},
			),
			logger, oauthMdw,
		))
	}

	// Run HTTP server
	runServer(httpPort, r, logger)
}
