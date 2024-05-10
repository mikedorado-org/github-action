package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/github-telemetry-go/telemetry"

	"github.com/redis/go-redis/extra/redisotel/v9"
	redis "github.com/redis/go-redis/v9"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"

	otelTrace "go.opentelemetry.io/otel/trace"
)

var redis_client *redis.ClusterClient
var tracer otelTrace.Tracer

func Init(telemetryProvider *telemetry.Provider) {
	tracer = telemetryProvider.Tracer.Provider.Tracer("redis")

	if url := os.Getenv("REDIS_URL"); url != "" {
		opt, err := redis.ParseURL(url)
		if err != nil {
			log.Fatal("Could not parse options" + err.Error())
		}

		clusterOptions := redis.ClusterOptions{
			Addrs:           []string{opt.Addr},
			DialTimeout:     opt.DialTimeout,
			MaxRetries:      opt.MaxRetries,
			MaxRetryBackoff: opt.MaxRetryBackoff,
			MinIdleConns:    opt.MinIdleConns,
			MinRetryBackoff: opt.MinRetryBackoff,
			Password:        opt.Password,
			PoolSize:        opt.PoolSize,
			PoolTimeout:     opt.PoolTimeout,
			ReadTimeout:     opt.ReadTimeout,
			TLSConfig:       opt.TLSConfig,
			NewClient: func(nodeOpt *redis.Options) *redis.Client {
				node := redis.NewClient(nodeOpt)
				return node
			},
		}

		if password := os.Getenv("REDIS_PASSWORD"); password != "" {
			clusterOptions.Password = password
		}

		redis_client = redis.NewClusterClient(&clusterOptions)
	} else {
		redis_client = redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{"redis:6379"}})
	}

	// Instrument client with distributed tracing for each node
	if err := redisotel.InstrumentTracing(redis_client); err != nil {
		log.Fatal("error starting tracing for redis")
	}
}

type PubSubMessage struct {
	// Metadata to include tracing headers
	Metadata map[string]string `json:"metadata,omitempty"`
	Data     string            `json:"request,omitempty"`
}

func ExtractSpanFromRedisPubSubMessage(msg *PubSubMessage) context.Context {
	// Extract the parent span from the message metadata
	return otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(msg.Metadata))
}

// Ping the Redis server
func Ping() (string, error) {
	return redis_client.Ping(context.Background()).Result()
}

func Subscribe(ctx context.Context, channel string) string {
	// Subscribe to the channel
	pubsub := redis_client.Subscribe(ctx, channel)

	// Listen for messages
	go func() {
		for redisMsg := range pubsub.Channel() {
			// Deserialize the message
			var msg PubSubMessage
			json.Unmarshal([]byte(redisMsg.Payload), &msg)

			// Extract the parent span from the message and start a new span
			_, span := tracer.Start(
				ExtractSpanFromRedisPubSubMessage(&msg),
				"message received",
				otelTrace.WithAttributes(attribute.String("message", msg.Data)),
				otelTrace.WithSpanKind(otelTrace.SpanKindConsumer),
			)

			// Perform some work
			log.Info("message received", kvp.String("channel", channel), kvp.String("message", msg.Data))

			span.End()
		}
	}()

	return fmt.Sprintf("Subscribed to %v", channel)
}

func Publish(ctx context.Context, channel string, message string) string {
	// Inject the current span tracing data into the message metadata
	metadata := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(metadata))

	// Serialize the message
	msg := PubSubMessage{metadata, message}
	b, _ := json.Marshal(msg)

	// Publish the message
	intCmd := redis_client.Publish(ctx, channel, b)

	return fmt.Sprintf("Message published to %v subscribers", intCmd.Val())
}
