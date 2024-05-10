package otel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Handler struct {
	handler http.Handler
}
type AdditionalHeaders struct{}
type Processor struct{}

func NewHandler(handler http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	h := Handler{
		handler: otelhttp.NewHandler(handler, operation, opts...),
	}
	return &h
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestId := r.Header[http.CanonicalHeaderKey("X-Request-Id")]
	ctx := r.Context()
	if len(requestId) > 0 {
		ctx = context.WithValue(r.Context(), AdditionalHeaders{}, requestId[0])
	}

	h.handler.ServeHTTP(w, r.WithContext(ctx))
}

func (p Processor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
	header, ok := parent.Value(AdditionalHeaders{}).(string)
	if ok {
		s.SetAttributes(attribute.KeyValue{
			Key:   "guid:x-request-id",
			Value: attribute.StringValue(header),
		})
	}
}

func (p Processor) OnEnd(s trace.ReadOnlySpan)           {}
func (p Processor) Shutdown(ctx context.Context) error   { return nil }
func (p Processor) ForceFlush(ctx context.Context) error { return nil }
