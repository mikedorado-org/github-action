package rest

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/github/actions-example-go/internal/hydro"
	"github.com/github/actions-example-go/internal/metrics"
	"github.com/github/actions-example-go/internal/redis"

	"github.com/github/github-telemetry-go/kvp"
	"github.com/github/github-telemetry-go/log"
	"github.com/github/github-telemetry-go/telemetry"

	"github.com/justinas/alice"
)

func NewHTTPHandler(telemetryProvider *telemetry.Provider) http.Handler {
	mw := alice.New(logMiddleware, metrics.StatsMiddleware())

	base := mw.Then(tracedHttpHandler(notFound, "notFound"))
	ready := mw.Then(tracedHttpHandler(getReady, "ready"))
	health := mw.Then(tracedHttpHandler(getHealth, "health"))
	hello := mw.Then(tracedHttpHandler(getHello, "hello"))
	redis := mw.Then(tracedHttpHandler(postRedis, "redis"))
	hydro := mw.Then(tracedHttpHandler(postHydro, "hydro"))

	mux := http.NewServeMux()
	mux.Handle("/", base)
	mux.Handle("/ready", ready)
	mux.Handle("/health", health)
	mux.Handle("/hello/", hello)
	mux.Handle("/redis/", redis)
	mux.Handle("/hydro/", hydro)

	return mux
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	log.Info("Getting health status...")
	log.Debug("Debugging health status.", kvp.Int("http.request_content_length", int(r.ContentLength)))

	fs := syscall.Statfs_t{}
	err := syscall.Statfs("/", &fs)
	if err != nil {
		respond(r, w, http.StatusInternalServerError)
	}
	diskTotal := float64(fs.Blocks*uint64(fs.Bsize)) / float64(1024*1024*1024)
	diskFree := float64(fs.Bfree*uint64(fs.Bsize)) / float64(1024*1024*1024)
	diskUsed := diskTotal - diskFree

	_, err = redis.Ping()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"status": "DOWN",
			"details": map[string]any{
				"redis": map[string]any{
					"status": "DOWN",
					"details": map[string]any{
						"error": err.Error(),
						"url":   os.Getenv("REDIS_URL"),
					},
				},
				"diskSpace": map[string]any{
					"total":     fmt.Sprintf("%.2f GB", diskTotal),
					"used":      fmt.Sprintf("%.2f GB", diskUsed),
					"available": fmt.Sprintf("%.2f GB", diskFree),
				},
			},
		})
		respond(r, w, http.StatusInternalServerError)
	} else {
		json.NewEncoder(w).Encode(map[string]any{
			"status": "UP",
			"details": map[string]any{
				"redis": map[string]any{
					"status": "UP",
					"details": map[string]any{
						"url": os.Getenv("REDIS_URL"),
					},
				},
				"diskSpace": map[string]any{
					"total":     fmt.Sprintf("%.2f GB", diskTotal),
					"used":      fmt.Sprintf("%.2f GB", diskUsed),
					"available": fmt.Sprintf("%.2f GB", diskFree),
				},
			},
		})
		respond(r, w, http.StatusOK)
	}
}

func getReady(w http.ResponseWriter, r *http.Request) {
	metrics.Counter("ReadinessProbe.Calls", 1)

	json.NewEncoder(w).Encode(map[string]any{
		"ready": true,
	})
	respond(r, w, http.StatusOK)
}

func getHello(w http.ResponseWriter, r *http.Request) {
	vars := strings.Split(r.URL.Path, "/")
	vars = remove(vars, "")

	response := fmt.Sprintf("Hello, %v", vars[1])
	if os.Getenv("KUBE_SITE") != "" {
		response = fmt.Sprintf("%v from %v", response, os.Getenv("KUBE_SITE"))
	}
	response = fmt.Sprintf("%v! 🚀", response)
	respondWithString(r, w, http.StatusOK, response)
}

func postRedis(w http.ResponseWriter, r *http.Request) {
	vars := strings.Split(r.URL.Path, "/")
	vars = remove(vars, "")

	if len(vars) == 2 {
		response := redis.Subscribe(r.Context(), vars[1])
		respondWithString(r, w, http.StatusOK, response)
	} else if len(vars) == 3 {
		response := redis.Publish(r.Context(), vars[1], vars[2])
		respondWithString(r, w, http.StatusOK, response)
	}
}

func postHydro(w http.ResponseWriter, r *http.Request) {
	vars := strings.Split(r.URL.Path, "/")
	vars = remove(vars, "")
	hydro.Publish(r.Context(), vars[1])
	respond(r, w, http.StatusOK)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	respondWithString(r, w, http.StatusNotFound, "Not found")
}

// Logs the request and sends an HTTP response
func respondWithString(r *http.Request, w http.ResponseWriter, status int, response string) {
	w.Write([]byte(html.EscapeString(response)))
	respond(r, w, status)
}

func respond(r *http.Request, w http.ResponseWriter, status int) {
	log.Info("Request received",
		kvp.String("http.client_ip", r.RemoteAddr),
		kvp.String("http.url", r.URL.String()),
		kvp.String("http.user_agent", r.UserAgent()),
		kvp.String("http.method", r.Method),
		kvp.Int("http.status", status),
		kvp.String("net.sock.peer.addr", r.RemoteAddr))
	w.WriteHeader(status)
}

func remove(s []string, r string) []string {
	var res []string
	for _, str := range s {
		if str != "" {
			res = append(res, str)
		}
	}
	return res
}
