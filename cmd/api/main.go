package main

import (
	"context"
	"database/sql"
	"log"
	"os"

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

	// Global context
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

	// Init asynq client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisConnAddr,
		PoolSize: redisPoolSize,
	})
	defer asynqClient.Close()

	// Init HTTP router
	r := initRouter(logger)

	// Set up limited oauth2 server for password, client_credentials, and refresh_token flows.
	// Does not support authorization code flow.
	// oauthServer := oauth.NewBearerServer(
	// 	oauthSigningKey,
	// 	accessTokenTTL,
	// 	iam.NewOAuthVerifier(
	// 		iamRepo,
	// 		iam.WithOauthAccessTokenTTL(accessTokenTTL),
	// 		iam.WithOauthRefreshTokenTTL(refreshTokenTTL),
	// 	),
	// 	nil,
	// )

	// OAuth2 Middleware
	oauthMdw := oauth.Authorize(oauthSigningKey, nil)

	// Mount HTTP endpoints
	{
		repo, err := repository.NewWithConnection(ctx, db)
		if err != nil {
			logger.Log("error", err, "msg", "failed to init repository")
			os.Exit(1)
		}

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
