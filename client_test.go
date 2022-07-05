package mget

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func startMockMetricsEndpoints(t *testing.T, metrics string) (*http.Server, int) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)
	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		fmt.Fprint(w, metrics)
	})

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			require.NoError(t, err)
		}
	}()

	time.Sleep(1 * time.Second)
	return &server, port
}

func TestClient(t *testing.T) {
	dummyMetrics := `# HELP go_gc_duration_seconds A summary of the pause duration.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 3.275e-05
go_gc_duration_seconds{quantile="0.5"} 0.000126252
go_gc_duration_seconds{quantile="1"} 0.096255362
go_gc_duration_seconds_count 7971
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 86
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.17.2"} 1
empty{} 1
server_up{short="service_a",boundary="backend"} 1
server_up{short="service_b",boundary="frontend"} 1
server_up{short="service_c",boundary="backend"} 1
server_up{short="service_d",boundary="frontend"} 1
server_up{short="service_e",boundary="backend"} 1`

	server, port := startMockMetricsEndpoints(t, dummyMetrics)
	endpoint := fmt.Sprintf("http://localhost:%d/metrics", port)
	defer server.Close()

	t.Run("Simple", func(t *testing.T) {
		result := NewClient().
			Endpoint(endpoint).Do(context.Background())
		require.NoError(t, result.Err)
		require.Equal(t, 200, result.ResponseStatus)

		require.Equal(t, []Metric{
			{
				Name:    "go_gc_duration_seconds",
				Help:    "A summary of the pause duration.",
				Type:    "SUMMARY",
				TypeRaw: "summary",
				Values: []LabelValuePair{
					{Label: map[string]string{"quantile": "0"}, Value: 3.275e-05},
					{Label: map[string]string{"quantile": "0.5"}, Value: 0.000126252},
					{Label: map[string]string{"quantile": "1"}, Value: 0.096255362},
				},
			},
			{
				Name:    "go_gc_duration_seconds_count",
				Help:    "",
				Type:    "UNKNOWN",
				TypeRaw: "",
				Values: []LabelValuePair{
					{Label: map[string]string{}, Value: 7971},
				},
			},
			{
				Name:    "go_goroutines",
				Help:    "Number of goroutines that currently exist.",
				Type:    "GAUGE",
				TypeRaw: "gauge",
				Values: []LabelValuePair{
					{Label: map[string]string{}, Value: 86},
				},
			},
			{
				Name:    "go_info",
				Help:    "Information about the Go environment.",
				Type:    "GAUGE",
				TypeRaw: "gauge",
				Values: []LabelValuePair{
					{Label: map[string]string{"version": "go1.17.2"}, Value: 1},
				},
			},
			{
				Name:    "empty",
				Help:    "",
				Type:    "UNKNOWN",
				TypeRaw: "",
				Values: []LabelValuePair{
					{Label: map[string]string{}, Value: 1},
				},
			},
			{
				Name:    "server_up",
				Help:    "",
				Type:    "UNKNOWN",
				TypeRaw: "",
				Values: []LabelValuePair{
					{Label: map[string]string{"boundary": "backend", "short": "service_a"}, Value: 1},
					{Label: map[string]string{"boundary": "frontend", "short": "service_b"}, Value: 1},
					{Label: map[string]string{"boundary": "backend", "short": "service_c"}, Value: 1},
					{Label: map[string]string{"boundary": "frontend", "short": "service_d"}, Value: 1},
					{Label: map[string]string{"boundary": "backend", "short": "service_e"}, Value: 1}},
			},
		}, result.Metric)
	})

	t.Run("FilterByName", func(t *testing.T) {
		result := NewClient().
			Endpoint(endpoint).
			FilterByName("go_gc_duration_seconds", "go_info").
			Do(context.Background())
		require.NoError(t, result.Err)
		require.Equal(t, 200, result.ResponseStatus)

		require.Equal(t, []Metric{
			{
				Name:    "go_gc_duration_seconds",
				Help:    "A summary of the pause duration.",
				Type:    SummaryType,
				TypeRaw: "summary",
				Values: []LabelValuePair{
					{Label: map[string]string{"quantile": "0"}, Value: 3.275e-05},
					{Label: map[string]string{"quantile": "0.5"}, Value: 0.000126252},
					{Label: map[string]string{"quantile": "1"}, Value: 0.096255362},
				},
			},
			{
				Name:    "go_info",
				Help:    "Information about the Go environment.",
				Type:    GaugeType,
				TypeRaw: "gauge",
				Values: []LabelValuePair{
					{Label: map[string]string{"version": "go1.17.2"}, Value: 1},
				},
			},
		}, result.Metric)
	})

	t.Run("FilterByType", func(t *testing.T) {
		result := NewClient().
			Endpoint(endpoint).
			FilterByType(SummaryType).
			Do(context.Background())
		require.NoError(t, result.Err)
		require.Equal(t, 200, result.ResponseStatus)

		require.Equal(t, []Metric{
			{
				Name:    "go_gc_duration_seconds",
				Help:    "A summary of the pause duration.",
				Type:    SummaryType,
				TypeRaw: "summary",
				Values: []LabelValuePair{
					{Label: map[string]string{"quantile": "0"}, Value: 3.275e-05},
					{Label: map[string]string{"quantile": "0.5"}, Value: 0.000126252},
					{Label: map[string]string{"quantile": "1"}, Value: 0.096255362},
				},
			},
		}, result.Metric)
	})

	t.Run("FilterByLabels", func(t *testing.T) {
		result := NewClient().
			Endpoint(endpoint).
			FilterByLabel(map[string][]string{
				"short":    {"service_a", "service_b", "service_c"},
				"boundary": {"backend"},
			}).
			Do(context.Background())
		require.NoError(t, result.Err)
		require.Equal(t, 200, result.ResponseStatus)

		require.Equal(t, []Metric{
			{
				Name:    "server_up",
				Help:    "",
				Type:    UnknownType,
				TypeRaw: "",
				Values: []LabelValuePair{
					{Label: map[string]string{"boundary": "backend", "short": "service_a"}, Value: 1},
					{Label: map[string]string{"boundary": "backend", "short": "service_c"}, Value: 1},
				},
			},
		}, result.Metric)
	})
}
