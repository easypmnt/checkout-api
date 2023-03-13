package recoverer

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// WithLogger is a custom recovery middleware that logs the error and stacktrace
func WithLogger(log *logrus.Entry) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {

					reqID := middleware.GetReqID(r.Context())

					if log != nil {
						log.WithFields(logrus.Fields{
							"request_id": reqID,
							"panic":      rvr,
							"stack":      string(debug.Stack()),
						}).Error("panic recovered")
					} else {
						middleware.PrintPrettyStack(rvr)
					}

					w.Header().Add("Content-Type", "application/json; charset=UTF-8")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":       http.StatusInternalServerError,
						"error":      http.StatusText(http.StatusInternalServerError),
						"request_id": reqID,
					})
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
