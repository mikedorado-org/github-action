package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/github/actions-example-go/internal/rest"
	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/github-telemetry-go/telemetry"
)

var httpSrv *http.Server

func StartServer(port int, telemetryProvider *telemetry.Provider) error {
	addr := fmt.Sprintf("0.0.0.0:%v", port)

	handler := rest.NewHTTPHandler(telemetryProvider)

	httpSrv = &http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * 5,
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 10,
		Handler:      handler,
	}

	log.Info("Server starting", kvp.String("address", addr))
	if err := httpSrv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func Shutdown() error {
	log.Info("Server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// anything else we need to clean up goes here

	httpSrv.Shutdown(ctx)

	return nil
}
