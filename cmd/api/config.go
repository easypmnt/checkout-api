package main

import (
	"time"

	"github.com/dmitrymomot/go-env"
	_ "github.com/joho/godotenv/autoload" // Load .env file automatically
)

var (
	// Application
	appName  = env.GetString("APP_NAME", "api")
	appDebug = env.GetBool("APP_DEBUG", false)

	// Product
	productName    = env.GetString("PRODUCT_NAME", "Checkout API")                                                // To show on client side
	productIconURI = env.GetString("PRODUCT_ICON", "https://avatars.githubusercontent.com/u/125194068?s=200&v=4") // absolute URI to product icon

	// HTTP Router
	httpPort                  = env.GetInt("HTTP_PORT", 8080)
	httpRequestTimeout        = env.GetDuration("HTTP_REQUEST_TIMEOUT", time.Second*10)
	httpServerShutdownTimeout = env.GetDuration("HTTP_SERVER_SHUTDOWN_TIMEOUT", time.Second*5)
	httpLimitRequestBodySize  = env.GetInt[int64]("HTTP_LIMIT_REQUEST_BODY_SIZE", 1<<20) // 1 MB
	httpRateLimit             = env.GetInt("HTTP_RATE_LIMIT", 100)
	httpRateLimitDuration     = env.GetDuration("HTTP_RATE_LIMIT_DURATION", time.Minute)

	// Cors
	corsAllowedOrigins     = env.GetStrings("CORS_ALLOWED_ORIGINS", ",", []string{"*"})
	corsAllowedMethods     = env.GetStrings("CORS_ALLOWED_METHODS", ",", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"})
	corsAllowedHeaders     = env.GetStrings("CORS_ALLOWED_HEADERS", ",", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "X-Request-Id", "Origin", "User-Agent", "Accept-Encoding", "Accept-Language", "Cache-Control", "Connection", "DNT", "Host", "Pragma", "Referer"})
	corsAllowedCredentials = env.GetBool("CORS_ALLOWED_CREDENTIALS", true)
	corsMaxAge             = env.GetInt("CORS_MAX_AGE", 300)

	// Build tag is set up while deployment
	buildTag        = "undefined"
	buildTagRuntime = env.GetString("COMMIT_HASH", buildTag)

	// DB
	dbConnString   = env.MustString("DATABASE_URL")
	dbMaxOpenConns = env.GetInt("DATABASE_MAX_OPEN_CONNS", 20)
	dbMaxIdleConns = env.GetInt("DATABASE_IDLE_CONNS", 2)

	// Redis
	redisConnAddr     = env.MustString("REDIS_CONN_ADDR")
	redisNetwork      = env.GetString("REDIS_NETWORK", "tcp")
	redisUsername     = env.GetString("REDIS_USERNAME", "")
	redisPassword     = env.GetString("REDIS_PASSWORD", "")
	redisDB           = env.GetInt("REDIS_DB", 0)
	redisDialTimeout  = env.GetDuration("REDIS_DIAL_TIMEOUT", 5*time.Second)
	redisReadTimeout  = env.GetDuration("REDIS_READ_TIMEOUT", 3*time.Second)
	redisWriteTimeout = env.GetDuration("REDIS_WRITE_TIMEOUT", 3*time.Second)
	redisPoolSize     = env.GetInt("REDIS_POOL_SIZE", 10)

	// Auth
	oauthSigningKey = env.MustString("OAUTH_SIGNING_KEY")
	accessTokenTTL  = env.GetDuration("ACCESS_TOKEN_TTL", time.Minute*5)
	refreshTokenTTL = env.GetDuration("REFRESH_TOKEN_TTL", time.Hour)
	clientID        = env.MustString("CLIENT_ID")
	clientSecret    = env.MustString("CLIENT_SECRET")

	// Worker
	workerConcurrency = env.GetInt("WORKER_CONCURRENCY", 10)
	queueName         = env.GetString("QUEUE_NAME", "default")

	// Webhook
	webhookSignatureSecret = env.MustBytes("WEBHOOK_SIGNATURE_SECRET")
	webhookURI             = env.MustString("WEBHOOK_URI")

	// Solana
	solanaRPCEndpoint = env.GetString("SOLANA_RPC_ENDPOINT", "https://api.devnet.solana.com")
	solanaWSSEndpoint = env.GetString("SOLANA_WSS_ENDPOINT", "wss://api.devnet.solana.com")
	solanaPayBaseURI  = env.GetString("SOLANA_PAY_BASE_URI", "https://checkout-api.easypmnt.com/payment/checkout/")

	// Merchant
	merchantWalletAddress      = env.MustString("MERCHANT_WALLET_ADDRESS")
	merchantApplyBonus         = env.GetBool("MERCHANT_APPLY_BONUS", true)
	merchantMaxBonusPercentage = env.GetInt[int16]("MERCHANT_MAX_BONUS_PERCENTAGE", 5000)
	bonusMintAddress           = env.MustString("BONUS_MINT_ADDRESS")
	bonusMintAuthority         = env.MustString("BONUS_MINT_AUTHORITY")
	bonusRate                  = env.GetInt[int64]("BONUS_RATE", 100)
)
