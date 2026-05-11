package metrics

import (
	"fmt"
	"net/http"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Success  *prometheus.GaugeVec
	Duration *prometheus.GaugeVec
}

// New creates and registers Prometheus metrics with the given prefix.
func New(prefix string) *Metrics {
	m := &Metrics{
		Success: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_success", prefix),
			Help: "Whether the RTSP stream probe succeeded (1) or failed (0).",
		}, []string{"camera"}),
		Duration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_duration_seconds", prefix),
			Help: "Time taken for the RTSP stream probe in seconds.",
		}, []string{"camera"}),
	}

	prometheus.MustRegister(m.Success, m.Duration)
	return m
}

func (m *Metrics) Record(camera string, success bool, durationSecs float64) {
	if success {
		m.Success.WithLabelValues(camera).Set(1)
	} else {
		m.Success.WithLabelValues(camera).Set(0)
	}
	m.Duration.WithLabelValues(camera).Set(durationSecs)
}

// ServeHTTP starts the /metrics HTTP server on the given port.
func ServeHTTP(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		slog.Info("starting metrics server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	return srv
}
