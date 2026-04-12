package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"

	"github.com/hippowc/brook/internal/launcher"
	"github.com/hippowc/brook/pkg/agentconfig"
)

type chatHandler struct {
	rt    *launcher.Runtime
	store SessionStore
	spec  *agentconfig.GatewaySpec
	mu    *sync.Mutex
}

type chatRequest struct {
	Text             string `json:"text"`
	UserID           string `json:"user_id"`
	ConversationID   string `json:"conversation_id"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

type jsonError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errBodyTooLarge) {
		return true
	}
	// http.MaxBytesReader / json on truncated body
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return strings.Contains(err.Error(), "request body too large")
}

func (h *chatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, jsonError{Error: "method not allowed"})
		return
	}
	maxB := int64(h.spec.MaxRequestBodyBytes)
	if maxB <= 0 {
		maxB = 1 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxB)
	defer r.Body.Close()

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if decodeBodyTooLarge(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, jsonError{Error: "request body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, jsonError{Error: "invalid json"})
		return
	}
	text := strings.TrimSpace(req.Text)
	uid := strings.TrimSpace(req.UserID)
	if uid == "" {
		writeJSON(w, http.StatusBadRequest, jsonError{Error: "user_id is required"})
		return
	}
	if text == "" {
		writeJSON(w, http.StatusBadRequest, jsonError{Error: "text is required"})
		return
	}
	cid := strings.TrimSpace(req.ConversationID)
	key := SessionKey(uid, cid)

	qsec := h.spec.QueryTimeoutSeconds
	if qsec <= 0 {
		qsec = 600
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(qsec)*time.Second)
	defer cancel()

	h.mu.Lock()
	defer h.mu.Unlock()

	sessKV, err := h.store.Load(key)
	if err != nil {
		slog.Error("gateway session load", "err", err, "key", key)
		writeJSON(w, http.StatusInternalServerError, jsonError{Error: "session load failed"})
		return
	}
	cb, snap := launcher.SessionValuesSyncHandler()
	iter := h.rt.Runner.Query(ctx, text, adk.WithSessionValues(sessKV), adk.WithCallbacks(cb))
	reply, qerr := CollectAssistantText(iter)
	launcher.MergeSessionValues(sessKV, snap())
	if saveErr := h.store.Save(key, sessKV); saveErr != nil {
		slog.Error("gateway session save", "err", saveErr)
		writeJSON(w, http.StatusInternalServerError, jsonError{Error: "session save failed"})
		return
	}

	if qerr != nil {
		if errors.Is(qerr, context.DeadlineExceeded) || errors.Is(qerr, context.Canceled) ||
			errors.Is(ctx.Err(), context.DeadlineExceeded) {
			writeJSON(w, http.StatusGatewayTimeout, jsonError{Error: "query timeout"})
			return
		}
		slog.Error("gateway query", "err", qerr)
		writeJSON(w, http.StatusBadRequest, jsonError{Error: qerr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, chatResponse{Reply: reply})
}
