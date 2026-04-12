package gateway

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	buckets  map[string][]time.Time
	maxKeys  int
	evictEvery int
	n        int
}

func newIPLimiter(rpm int, burst int) *ipLimiter {
	if rpm <= 0 {
		rpm = 120
	}
	if burst <= 0 {
		burst = 0
	}
	limit := rpm + burst
	if limit <= 0 {
		limit = rpm
	}
	return &ipLimiter{
		window:     time.Minute,
		limit:      limit,
		buckets:    make(map[string][]time.Time),
		maxKeys:    10000,
		evictEvery: 100,
	}
}

func (l *ipLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.n++
	if l.n%l.evictEvery == 0 && len(l.buckets) > l.maxKeys {
		for k := range l.buckets {
			delete(l.buckets, k)
			break
		}
	}

	ts := l.buckets[key]
	var kept []time.Time
	for _, t := range ts {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.buckets[key] = kept
		return false
	}
	kept = append(kept, now)
	l.buckets[key] = kept
	return true
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		parts := strings.Split(x, ",")
		return strings.TrimSpace(parts[0])
	}
	if x := r.Header.Get("X-Real-IP"); x != "" {
		return strings.TrimSpace(x)
	}
	host, _, ok := strings.Cut(r.RemoteAddr, ":")
	if ok && host != "" {
		return host
	}
	return r.RemoteAddr
}
