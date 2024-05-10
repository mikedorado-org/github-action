package metrics

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	stats "github.com/github/go-stats"
	statsHttp "github.com/github/go-stats/http"
	runtimeMetrics "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var Githubstatsd_client *stats.Statsd
var githubstatd_tags stats.Tags
var initialized bool

type Settings struct {
	Environment string
	Service     string
	Version     string
}

func Init(settings Settings) {
	// Setup automatic runtime metrics
	runtimeMetrics.Start(
		runtimeMetrics.WithEnv(settings.Environment),
		runtimeMetrics.WithService(settings.Service),
		runtimeMetrics.WithServiceVersion(settings.Version),
		runtimeMetrics.WithTraceEnabled(false), // Tracing is not supported through Datadog at this time.
		runtimeMetrics.WithRuntimeMetrics(),
	)

	Githubstatsd_client = NewGitHubStats(settings)
	githubstatd_tags = stats.Tags{
		"environment": settings.Environment,
		"service":     settings.Service,
		"version":     settings.Version,
	}

	Githubstatsd_client.Run() // Runs the client in a non-blocking manner
	initialized = true
}

func Close() {
	// When the Datadog metrics are stopped, it will flush everything it has to the Datadog Agent before quitting.
	runtimeMetrics.Stop()

	Githubstatsd_client.Stop() // Ensure the client will stop running before the function returns
}

func NewGitHubStats(settings Settings) *stats.Statsd {
	statsdHost := os.Getenv("DD_AGENT_HOST")
	if statsdHost == "" {
		statsdHost = "localhost"
	}

	statsdPort := os.Getenv("DD_DOGSTATSD_PORT")
	if statsdPort == "" {
		statsdPort = "8125"
	}

	sink := stats.UDPSink(fmt.Sprintf("%s:%v", statsdHost, statsdPort))
	client := stats.NewClient(sink, time.Second, "")

	return client
}

func StatsMiddleware() func(http.Handler) http.Handler {
	return statsHttp.WithResponseMetrics(Githubstatsd_client, statsHttp.Options{})
}

func Counter(name string, rate float64) {
	if !initialized {
		// this no-op is for runnings tests without emitting metrics
		log.Error("Metrics not initialized", kvp.String("package", "metrics_publisher"))
		return
	}
	Githubstatsd_client.Counter(name, githubstatd_tags, int64(rate))
}
