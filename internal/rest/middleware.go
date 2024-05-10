package rest

import (
	"fmt"
	"net/http"

	internalOtel "github.com/github/actions-example-go/internal/otel"
	"github.com/github/github-telemetry-go/log"
)

func logMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Info(fmt.Sprintf("Request to %v", r.URL.Path))
		next.ServeHTTP(w, r)
		log.Info(fmt.Sprintf("End of request to %v", r.URL.Path))
	}

	return http.HandlerFunc(fn)
}

func tracedHttpHandler(httpHandlerFunc func(http.ResponseWriter, *http.Request), operation string) http.Handler {
	return internalOtel.NewHandler(http.HandlerFunc(httpHandlerFunc), operation)
}
