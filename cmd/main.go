package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/github/actions-example-go/internal/hydro"
	"github.com/github/actions-example-go/internal/redis"
	"github.com/github/actions-example-go/internal/server"
	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/github-telemetry-go/telemetry"
	"github.com/github/github-telemetry-go/trace"
	"go.opentelemetry.io/otel/sdk/resource"

	internalMetrics "github.com/github/actions-example-go/internal/metrics"
	internalOtel "github.com/github/actions-example-go/internal/otel"
	otelSdk "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	data, err := os.ReadFile("./settings.json")

	if err != nil {
		panic(err)
	}

	var settings map[string]interface{}

	err = json.Unmarshal(data, &settings)
	if err != nil {
		panic(err)
	}

	// METRICS
	internalMetrics.Init(
		internalMetrics.Settings{
			Environment: settings["environment"].(string),
			Service:     settings["service"].(string),
			Version:     "0.0.1",
		})

	defer internalMetrics.Close()

	// LOGGING & TRACING
	telemetryProvider, err := telemetry.NewFromEnv(
		telemetry.WithTracerOptions(
			trace.WithTracerProviderOptions(
				otelSdk.WithSpanProcessor(internalOtel.Processor{}),
				otelSdk.WithResource(resource.Default()),
			),
		),
	)
	if err != nil {
		fmt.Printf("Failed configuring telemetry: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if err := telemetryProvider.Shutdown(context.Background()); err != nil {
			fmt.Printf("failed to shutdown telemetry: %v", err)
			os.Exit(1)
		}
	}()

	logger := telemetryProvider.Logger.Named("main")
	log.SetDefault(logger)
	log.Info("hello world")

	// REDIS
	redis.Init(telemetryProvider)

	// HYDRO
	err = hydro.Init(telemetryProvider, internalMetrics.Githubstatsd_client, settings["environment"].(string))
	if err != nil {
		log.Error("failed to initialize hydro", kvp.Any("error", err))
	}

	// Close the publisher on shutdown to flush any pending events.
	defer hydro.Publisher.Close()

	go hydro.Consume()

	// To ensure publishing is working correctly, run a goroutine to publish a test message every hour.
	go func() {
		for {
			hydro.Publish(context.Background(), "AUTO PUBLISH TEST: "+time.Now().String())
			time.Sleep(time.Hour * 1)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		server.Shutdown()
		defer os.Exit(0)
	}()

	log.Info("Starting server")
	err = server.StartServer(8080, telemetryProvider)

	if err != nil {
		panic(err)
	}
}
