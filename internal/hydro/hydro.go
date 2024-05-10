package hydro

import (
	"context"
	"os"

	"github.com/Shopify/sarama"
	example_pb "github.com/github/actions-example-go/internal/hydro/schemas/github/actions/v0"
	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/github-telemetry-go/telemetry"
	"github.com/github/go-stats"
	hydro_pb "github.com/github/hydro-client-go/v6/generated/hydro/v1"
	"github.com/github/hydro-client-go/v6/pkg/hydro"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	otelTrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// Point the production KafkaSink at the truckee kafka brokers.
// Use port 9093 for SSL (required).
var brokers = []string{
	"kafka-lite:9092", // Include for local development, will only be used if kafka-lite pod is running
	"hydro-kafka-truckee-boot-1.service.iad.github.net:9093",
	"hydro-kafka-truckee-boot-2.service.iad.github.net:9093",
	"hydro-kafka-truckee-boot-3.service.iad.github.net:9093",
}

var tracer otelTrace.Tracer
var statsClient *stats.Statsd
var environment string

var Publisher *hydro.Publisher

func getKafkaOptions(clientName string) []hydro.KafkaConfigOption {
	clientID, err := os.Hostname()
	if err != nil {
		log.Fatal("failed to retrieve hostname", kvp.Any("error", err))
	}

	kafkaOptions := []hydro.KafkaConfigOption{
		hydro.WithKafkaLogger(NewHydroLogger(clientName)),
		// The client ID uniquely identifies your app in logs and metrics. <app>-<environment> is a good choice.
		hydro.WithClientID(clientID + "-" + clientName),
	}

	if statsClient != nil {
		kafkaOptions = append(kafkaOptions, hydro.WithKafkaStats(statsClient))
	}

	if environment == "dev" {
		// Use a kafka-lite compatible version for the local environment
		// The version must be locked when using hydro-client-go
		// https://thehub.github.com/epd/engineering/products-and-services/internal/hydro/resources/faq/#q-how-do-i-use-hydro-in-ghes
		kafkaOptions = append(kafkaOptions, hydro.WithKafkaVersion(sarama.V1_1_1_0.String()))
	} else {
		// Add the root CA for SSL. This cert is available on all hosts and kube containers.
		kafkaOptions = append(kafkaOptions, hydro.WithRootCA("/etc/ssl/certs/cp1-iad-production-1487801205-root.pem"))
	}

	return kafkaOptions
}

func Init(telemetryProvider *telemetry.Provider, sc *stats.Statsd, env string) error {
	tracer = telemetryProvider.Tracer.Provider.Tracer("hydro")
	statsClient = sc
	environment = env

	// Set the kafka logger to surface any problems from sarama, the underlying kafka client library.
	hydro.SetKafkaLogger(NewHydroLogger("sarama"))

	kafkaConfig, err := hydro.NewKafkaConfig(brokers,
		getKafkaOptions("publisher")...,
	)
	if err != nil {
		return err
	}

	// A sink is responsible for writing events to a destination.
	sink, err := hydro.NewKafkaSink(*kafkaConfig)
	if err != nil {
		return err
	}

	Publisher, err = hydro.NewPublisher(sink)
	if err != nil {
		return err
	}

	return nil
}

func Consume() {
	kafkaConfig, err := hydro.NewKafkaConfig(brokers,
		getKafkaOptions("consumer")...,
	)
	if err != nil {
		log.Fatal("failed to create kafka config", kvp.Any("error", err))
	}

	// Build a KafkaSource. The source will subscribe to 'github.actions.v0.ActionsExample'
	// and join the 'actions-example-go' consumer group.
	src, err := hydro.NewKafkaSource(*kafkaConfig, "actions-example-go", []string{"github.actions.v0.ActionsExample"})
	if err != nil {
		log.Fatal("failed to create kafka source", kvp.Any("error", err))
	}

	// Make sure to close the consumer before exiting.
	defer src.Close()

	// Build a cancellable context to pass into Consume. If the context is
	// cancelled, the consumer will break out of the consumer loop. The context
	// can be used to halt the consumer at shutdown and should be checked if
	// processing is slow.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a consumer loop with a handler function.
	src.Consume(ctx, func(_ context.Context, m hydro.Message) error {
		// The hydro.Message contains an encoded protobuf, so we must decode the event.
		envelope := hydro_pb.Envelope{}
		if err := proto.Unmarshal(m.Value, &envelope); err != nil {
			// Log the error so our logs inform us why the consumer processing failed.
			log.Error("failed to unmarshal envelope", kvp.Any("error", err))
			cancel()
			return err
		}

		// The hydro.Message contains an encoded protobuf, so we must decode the event.
		actionsExampleMessage := example_pb.ActionsExample{}
		err := proto.Unmarshal(envelope.Message, &actionsExampleMessage)

		// The handler return value controls offset commit behavior.
		// - If the handler returns nil, the message offset will be committed
		//   and the consumer will move on to the next message.
		// - If the handler returns an err, the offset will not be committed
		//   and the message will be retried forever (not usually desirable,
		//   should be combined with canceling the context).
		//
		// The handler should always return nil if it doesn't care about
		// processing failures. Otherwise, it should return an error to prevent
		// offset commits and cancel() the context to break out of the consumer
		// loop.
		if err != nil {
			// Log the error so our logs inform us why the consumer processing failed.
			log.Error("failed to unmarshal message", kvp.Any("error", err))
			cancel()
			return err
		}

		// Convert the headers to a string for logging
		messageHeaders := []string{}
		for k, v := range m.Headers {
			messageHeaders = append(messageHeaders, k, v)
		}

		// Extract the parent span from the message and start a new span
		_, span := tracer.Start(
			// Extract the parent span from the message metadata
			otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(m.Headers)),
			"message consumed",
			otelTrace.WithAttributes(attribute.String("envelope", envelope.String()), attribute.StringSlice("headers", messageHeaders)),
			otelTrace.WithSpanKind(otelTrace.SpanKindConsumer),
		)

		// Perform some work
		log.Info("message consumed", kvp.Strings("message headers", messageHeaders))
		log.Info("message consumed", kvp.String("message", envelope.String()))

		span.End()

		return nil
	})
}

func Publish(ctx context.Context, messageData string) {
	// Start a new span
	ctx, span := tracer.Start(
		ctx,
		"message published",
		otelTrace.WithAttributes(attribute.String("message", messageData)),
		otelTrace.WithSpanKind(otelTrace.SpanKindProducer),
	)

	// Create to protobuf message
	message := example_pb.ActionsExample{
		Message: messageData,
	}

	// Inject the current span tracing data into the metadata for the headers
	metadata := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(metadata))

	// Publish the message
	Publisher.Publish(&message, hydro.WithCustomHeaders(metadata))

	span.End()
}
