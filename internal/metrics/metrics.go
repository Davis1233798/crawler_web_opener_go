package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	TasksCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "crawler_tasks_completed_total",
		Help: "The total number of successfully completed tasks",
	})

	TasksFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "crawler_tasks_failed_total",
		Help: "The total number of failed tasks",
	})

	ActiveThreads = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "crawler_active_threads",
		Help: "The number of currently active browser threads",
	})

	QueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "crawler_queue_size",
		Help: "Current number of tasks in the queue",
	})

	SessionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "crawler_session_duration_seconds",
		Help:    "Duration of browser sessions",
		Buckets: prometheus.LinearBuckets(10, 10, 10), // 10s to 100s
	})
)

func StartMetricsServer(port int) {
	http.Handle("/metrics", promhttp.Handler())
	addr := ":" + strconv.Itoa(port)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			// log.Fatal(err) // Don't crash main app if metrics fail
		}
	}()
}
