package daemon

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codeberg.org/snonux/goprecords/internal/authkeys"
	"codeberg.org/snonux/goprecords/internal/goprecords"
)

type Config struct {
	StatsDir  string
	Addr      string
	AuthDB    string
	LogOutput io.Writer
}

func routes(statsDir, authDB string, store *authkeys.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/livez", health)
	mux.HandleFunc("/readyz", readiness(statsDir, authDB))
	mux.HandleFunc("/report", report(statsDir))
	mux.Handle("/upload/", uploadHandler(statsDir, store))
	return mux
}

func Handler(statsDir string) http.Handler {
	store, err := openAuthStore(context.Background(), statsDir, "")
	if err != nil {
		panic(err)
	}
	return routes(statsDir, "", store)
}

func logWriter(cfg Config) io.Writer {
	if cfg.LogOutput != nil {
		return cfg.LogOutput
	}
	return os.Stdout
}

func newDaemonLogger(w io.Writer) (*slog.Logger, slog.Handler) {
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h), h
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.code == 0 {
		r.code = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func withAccessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(rec, r)
		code := rec.code
		if code == 0 {
			code = http.StatusOK
		}
		log.Info("http_request", "method", r.Method, "path", r.URL.Path, "status", code, "duration_ms", time.Since(start).Milliseconds())
	})
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.StatsDir == "" {
		return fmt.Errorf("stats directory is required")
	}
	if cfg.Addr == "" {
		return fmt.Errorf("listen address is required")
	}
	w := logWriter(cfg)
	log, textHandler := newDaemonLogger(w)
	store, err := openAuthStore(ctx, cfg.StatsDir, cfg.AuthDB)
	if err != nil {
		return fmt.Errorf("auth db: %w", err)
	}
	defer store.Close()
	srv := &http.Server{
		Addr:     cfg.Addr,
		Handler:  withAccessLog(log, routes(cfg.StatsDir, cfg.AuthDB, store)),
		ErrorLog: slog.NewLogLogger(textHandler, slog.LevelError),
	}
	log.Info("daemon_listen", "addr", cfg.Addr)
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

func readiness(statsDir, authDB string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := checkReadinessDirs(statsDir, authDB); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}

func checkReadinessDirs(statsDir, authDB string) error {
	absStats, err := filepath.Abs(statsDir)
	if err != nil {
		return fmt.Errorf("stats-dir: %w", err)
	}
	if err := checkDirReadableWritable(absStats); err != nil {
		return fmt.Errorf("stats-dir: %w", err)
	}
	authPath := authDB
	if authPath == "" {
		authPath = authkeys.DefaultPath(statsDir)
	}
	absAuth, err := filepath.Abs(authPath)
	if err != nil {
		return fmt.Errorf("auth-db: %w", err)
	}
	dbDir := filepath.Clean(filepath.Dir(absAuth))
	if dbDir != filepath.Clean(absStats) {
		if err := checkDirReadableWritable(dbDir); err != nil {
			return fmt.Errorf("auth-db dir: %w", err)
		}
	}
	return nil
}

func checkDirReadableWritable(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing")
		}
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("not a directory")
	}
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("not readable: %w", err)
	}
	_, err = f.Readdirnames(1)
	_ = f.Close()
	if err != nil && err != io.EOF {
		return fmt.Errorf("not readable: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".goprecords-ready-*")
	if err != nil {
		return fmt.Errorf("not writable: %w", err)
	}
	name := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("not writable: %w", err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("not writable: %w", err)
	}
	return nil
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
	case goprecords.FormatGemtext:
		return "text/gemini; charset=utf-8"
	case goprecords.FormatHTML:
		return "text/html; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}
