package daemon

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"codeberg.org/snonux/goprecords/internal/goprecords"
)

// Config holds daemon HTTP server settings.
type Config struct {
	StatsDir string
	Addr     string
}

// Handler returns an HTTP handler that serves health checks and reports from statsDir.
func Handler(statsDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/report", report(statsDir))
	return mux
}

// Run starts a plain HTTP server until ctx is cancelled or ListenAndServe fails.
func Run(ctx context.Context, cfg Config) error {
	if cfg.StatsDir == "" {
		return fmt.Errorf("stats directory is required")
	}
	if cfg.Addr == "" {
		return fmt.Errorf("listen address is required")
	}
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: Handler(cfg.StatsDir),
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		err := <-errCh
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("shutdown: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		if err != nil {
			return fmt.Errorf("listen %s: %w", cfg.Addr, err)
		}
		return nil
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func report(statsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cfg, err := goprecords.ParseReportQuery(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		aggr := goprecords.NewAggregator(statsDir)
		aggregates, err := aggr.Aggregate(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var buf bytes.Buffer
		if err := goprecords.WriteReports(&buf, aggregates, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", reportContentType(cfg.OutputFormat))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func reportContentType(f goprecords.OutputFormat) string {
	switch f {
	case goprecords.FormatMarkdown:
		return "text/markdown; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}
