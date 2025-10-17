package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics: Simple counters for requests
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"path", "method", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
}

// Health response
type Health struct {
	Status string `json:"status"`
}

// Request logger with correlation ID
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		r.Header.Set("X-Request-ID", id) // Propagate

		// Log with ID (redact secrets later)
		log.Printf("[%s] %s %s from %s", id, r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		// Track status (simple: assume 200 unless error)
		status := "200"
		latency := time.Since(start)
		requestsTotal.WithLabelValues(r.URL.Path, r.Method, status).Inc()
		log.Printf("[%s] Completed %s %s in %v", id, r.Method, r.URL.Path, latency)
	})
}

// Health handler
func healthz(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	log.Printf("[%s] Health check", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Health{Status: "healthy"})
}

// Metrics handler
func metrics(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	log.Printf("[%s] Metrics request", id)
	promhttp.Handler().ServeHTTP(w, r)
}

// Root handler (simple echo for testing)
func root(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	fmt.Fprintf(w, "Podinfo from scratch! Req ID: %s", id)
}

func main() {
	// Env vars (e.g., for secret later)
	port := os.Getenv("PORT")
	if port == "" {
		port = "9898"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", root)

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, loggingMiddleware(mux)))
}