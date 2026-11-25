package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
)

// maxTrackedClients bounds the limiter's memory. When it is full of active
// clients, new clients are refused until old entries refill and are dropped.
const maxTrackedClients = 10000

// RateLimiter is a per-client token bucket: each client starts with Burst
// requests and earns one more every Every. A nil *RateLimiter allows all
// requests, so tests and tools can build the router without one.
type RateLimiter struct {
	burst    float64
	every    time.Duration
	clientIP func(*http.Request) string
	now      func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimiter(burst int, every time.Duration, clientIP func(*http.Request) string) *RateLimiter {
	return &RateLimiter{
		burst:    float64(burst),
		every:    every,
		clientIP: clientIP,
		now:      time.Now,
		buckets:  make(map[string]*bucket),
	}
}

// allow takes one token for key, or reports how long until one is available.
func (l *RateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= maxTrackedClients {
			return false, l.every
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+float64(now.Sub(b.last))/float64(l.every))
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) * float64(l.every))
}

// sweep drops buckets that have refilled completely, at most once a minute.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	full := time.Duration(l.burst * float64(l.every))
	for key, b := range l.buckets {
		if now.Sub(b.last) >= full {
			delete(l.buckets, key)
		}
	}
}

// Limit answers 429 RATE_LIMITED with Retry-After once a client runs out.
func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	if l == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.allow(l.clientIP(r))
		if !ok {
			seconds := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			httpx.Error(w, r, &domain.Error{
				Kind:    domain.KindRateLimited,
				Message: fmt.Sprintf("Too many attempts. Try again in %s.", humanSeconds(seconds)),
				Details: map[string]any{"retryAfterSeconds": seconds},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func humanSeconds(s int) string {
	if s >= 120 {
		return fmt.Sprintf("%d minutes", (s+59)/60)
	}
	if s == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", s)
}

// ClientIP returns how to identify the caller. With trustProxy, the API runs
// behind Caddy and Next.js: Caddy replaces X-Forwarded-For with the address
// it saw and Next.js passes it on, so the last entry is the real client.
// Without it (the API reachable directly), forwarded headers are ignored
// because any client could forge them.
func ClientIP(trustProxy bool) func(*http.Request) string {
	return func(r *http.Request) string {
		if trustProxy {
			forwarded := r.Header.Values("X-Forwarded-For")
			if len(forwarded) > 0 {
				parts := strings.Split(forwarded[len(forwarded)-1], ",")
				if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
					return ip
				}
			}
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
}
