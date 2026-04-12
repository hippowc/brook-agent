package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hippowc/brook/pkg/agentconfig"
)

func readBodySnapshot(r *http.Request, maxBytes int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	_ = r.Body.Close()
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, errBodyTooLarge
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return b, nil
}

func authMiddleware(spec *agentconfig.GatewayAuthSpec, maxBodyBytes int64, next http.Handler) http.Handler {
	mode := strings.ToLower(strings.TrimSpace(spec.Mode))
	if mode == "" || mode == "none" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "bearer":
			env := strings.TrimSpace(spec.BearerTokenEnv)
			want := strings.TrimSpace(os.Getenv(env))
			if want == "" {
				http.Error(w, "server misconfiguration: empty bearer token env", http.StatusInternalServerError)
				return
			}
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			got = strings.TrimSpace(got)
			if got != want {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		case "hmac":
			secEnv := strings.TrimSpace(spec.HMACSecretEnv)
			secret := strings.TrimSpace(os.Getenv(secEnv))
			if secret == "" {
				http.Error(w, "server misconfiguration: empty hmac secret env", http.StatusInternalServerError)
				return
			}
			ts := r.Header.Get("X-Brook-Timestamp")
			sig := r.Header.Get("X-Brook-Signature")
			if ts == "" || sig == "" {
				http.Error(w, "missing X-Brook-Timestamp or X-Brook-Signature", http.StatusUnauthorized)
				return
			}
			skew := spec.HMACMaxSkewSeconds
			if skew <= 0 {
				skew = 300
			}
			body, err := readBodySnapshot(r, maxBodyBytes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !verifyHMAC([]byte(secret), ts, body, sig, skew) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}
		default:
			http.Error(w, "unsupported auth mode", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func verifyHMAC(secret []byte, tsStr string, body []byte, sigHex string, maxSkewSec int) bool {
	ts, err := strconv.ParseInt(strings.TrimSpace(tsStr), 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	d := now - ts
	if d < 0 {
		d = -d
	}
	if int(d) > maxSkewSec {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(tsStr))
	mac.Write([]byte("\n"))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	sigHex = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(sigHex), "sha256="))
	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	exp, err := hex.DecodeString(want)
	if err != nil {
		return false
	}
	return hmac.Equal(got, exp)
}
