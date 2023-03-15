package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/easypmnt/checkout-api/internal/recoverer"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"
)

// Init HTTP router
func initRouter(log *logrus.Entry) *chi.Mux {
	r := chi.NewRouter()

	r.Use(
		// recoverer.WithLogger(log),
		recoverer.WithLogger(nil),
		middleware.AllowContentType(
			"application/json",
			"application/x-www-form-urlencoded",
			"text/event-stream",
		),
		middleware.CleanPath,
		middleware.StripSlashes,
		middleware.GetHead,
		middleware.NoCache,
		middleware.RealIP,
		middleware.RequestID,

		// Basic CORS
		// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
		cors.Handler(cors.Options{
			AllowedOrigins:   corsAllowedOrigins,
			AllowedMethods:   corsAllowedMethods,
			AllowedHeaders:   corsAllowedHeaders,
			AllowCredentials: corsAllowedCredentials,
			MaxAge:           corsMaxAge, // Maximum value not ignored by any of major browsers
		}),

		// Uses for testing error response with needed status code
		testingMdw,
	)

	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)

	r.Get("/", mkRootHandler(buildTagRuntime))
	r.Get("/health", healthCheckHandler)

	return r
}

// Run HTTP server
func runServer(ctx context.Context, httpPort int, router http.Handler, log *logrus.Entry) func() error {
	return func() error {
		log = log.WithField("port", httpPort)
		log.Info("Starting HTTP server")
		defer func() { log.Info("HTTP server stopped") }()

		httpServer := &http.Server{
			Handler: router,
			Addr:    fmt.Sprintf(":%d", httpPort),
		}

		go func() {
			<-ctx.Done()
			log.Info("Waiting for all connections to be closed")

			// Trigger graceful shutdown
			httpServer.Shutdown(ctx)
		}()

		// Run the server
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server shut down with an error: %w", err)
		}

		return nil
	}
}

// returns 204 HTTP status without content
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// returns 404 HTTP status with payload
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	defaultResponse(w, http.StatusNotFound, map[string]interface{}{
		"code":       http.StatusNotFound,
		"error":      fmt.Sprintf("Endpoint %s", http.StatusText(http.StatusNotFound)),
		"request_id": middleware.GetReqID(r.Context()),
	})
}

// returns 405 HTTP status with payload
func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	defaultResponse(w, http.StatusMethodNotAllowed, map[string]interface{}{
		"code":       http.StatusMethodNotAllowed,
		"error":      http.StatusText(http.StatusMethodNotAllowed),
		"request_id": middleware.GetReqID(r.Context()),
	})
}

// returns current build tag
func mkRootHandler(buildTag string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		defaultResponse(w, http.StatusOK, map[string]interface{}{
			"build_tag": buildTag,
		})
	}
}

// helper to send response as a json data
func defaultResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Testing middleware
// Helps to test any HTTP error
// Pass must_err query parameter with code you want get
// E.g.: /login?must_err=403
func testingMdw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if errCodeStr := r.URL.Query().Get("must_err"); len(errCodeStr) == 3 {
			if errCode, err := strconv.Atoi(errCodeStr); err == nil && errCode >= 400 && errCode < 600 {
				defaultResponse(w, errCode, map[string]interface{}{
					"code":       errCode,
					"error":      http.StatusText(errCode),
					"request_id": middleware.GetReqID(r.Context()),
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
