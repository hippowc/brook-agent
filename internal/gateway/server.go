package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/hippowc/brook/internal/launcher"
)

// Run 启动 HTTP 服务并在 ctx 取消时优雅关闭。要求 root.Gateway.Enabled 且已校验。
func Run(ctx context.Context, rt *launcher.Runtime, store SessionStore) error {
	spec := &rt.Root.Gateway
	if !spec.Enabled {
		return errors.New("gateway: gateway.enabled is false")
	}
	addr := strings.TrimSpace(spec.Listen)
	if addr == "" {
		addr = ":8787"
	}

	var mu sync.Mutex
	h := &chatHandler{rt: rt, store: store, spec: spec, mu: &mu}

	maxBody := int64(spec.MaxRequestBodyBytes)
	if maxBody <= 0 {
		maxBody = 1 << 20
	}

	chatChain := authMiddleware(&spec.Auth, maxBody, h)
	if spec.RateLimit != nil && spec.RateLimit.Enabled {
		lim := newIPLimiter(spec.RateLimit.RequestsPerMinute, spec.RateLimit.Burst)
		chatChain = limitMiddleware(lim, chatChain)
	}
	chatChain = recoverMiddleware(chatChain)

	mux := http.NewServeMux()
	mux.Handle("POST /v1/chat", chatChain)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler)

	rh := spec.ReadHeaderTimeoutSeconds
	if rh <= 0 {
		rh = 10
	}
	rtSec := spec.ReadTimeoutSeconds
	if rtSec <= 0 {
		rtSec = 120
	}
	qsec := spec.QueryTimeoutSeconds
	if qsec <= 0 {
		qsec = 600
	}
	wtSec := spec.WriteTimeoutSeconds
	if wtSec <= 0 {
		wtSec = qsec + 120
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(rh) * time.Second,
		ReadTimeout:       time.Duration(rtSec) * time.Second,
		WriteTimeout:      time.Duration(wtSec) * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("brook-gateway listening", "addr", addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shut := spec.ShutdownTimeoutSeconds
		if shut <= 0 {
			shut = 30
		}
		sctx, cancel := context.WithTimeout(context.Background(), time.Duration(shut)*time.Second)
		defer cancel()
		if err := srv.Shutdown(sctx); err != nil {
			slog.Error("gateway shutdown", "err", err)
		}
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"ready":true}`))
}

func limitMiddleware(l *ipLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			writeJSON(w, http.StatusTooManyRequests, jsonError{Error: "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("gateway panic", "recover", rec, "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(jsonError{Error: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
