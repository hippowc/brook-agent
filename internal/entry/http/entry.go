package httpentry

import (
	"context"
	"encoding/json"
	"net/http"

	"brook-agent/internal/common"
	"brook-agent/internal/entry"
	"brook-agent/internal/frame"
	"brook-agent/internal/model"
)

const Name = "http"

type Entry struct {
	cfg entry.Config
}

func init() {
	entry.Register(Name, func(cfg entry.Config) entry.Entry {
		return &Entry{cfg: cfg}
	})
}

func (e *Entry) Name() string { return Name }

// Start 启动 HTTP 入口，提供 /chat POST 接口。
func (e *Entry) Start(ctx context.Context, handler frame.Handler) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req model.AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			req.SessionID = "http-session"
		}
		resp, err := handler.Handle(ctx, &req, common.NopStreamWriter{})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{
		Addr:    e.cfg.Addr,
		Handler: mux,
	}
	return server.ListenAndServe()
}
