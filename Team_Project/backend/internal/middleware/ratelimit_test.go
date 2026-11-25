package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterBurstRefillAndClients(t *testing.T) {
	clock := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	l := NewRateLimiter(3, 10*time.Second, ClientIP(false))
	l.now = func() time.Time { return clock }
	h := l.Limit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	send := func(remote string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = remote
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := range 3 {
		if rec := send("198.51.100.7:4000"); rec.Code != http.StatusNoContent {
			t.Fatalf("request %d within burst: %d", i+1, rec.Code)
		}
	}
	rec := send("198.51.100.7:4001")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "10" {
		t.Fatalf("over the limit: %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	var env struct {
		Error struct{ Code, Message string }
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error.Code != "RATE_LIMITED" || env.Error.Message != "Too many attempts. Try again in 10 seconds." {
		t.Fatalf("error body: %s", rec.Body.String())
	}
	if rec := send("203.0.113.9:4000"); rec.Code != http.StatusNoContent {
		t.Fatalf("another client must have its own budget: %d", rec.Code)
	}

	clock = clock.Add(10 * time.Second)
	if rec := send("198.51.100.7:4000"); rec.Code != http.StatusNoContent {
		t.Fatalf("one request refilled after 10s: %d", rec.Code)
	}
	if rec := send("198.51.100.7:4000"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("only one token refilled: %d", rec.Code)
	}
}

func TestRateLimiterBoundsMemory(t *testing.T) {
	clock := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	l := NewRateLimiter(1, time.Hour, nil)
	l.now = func() time.Time { return clock }
	for i := range maxTrackedClients {
		if ok, _ := l.allow(fmt.Sprint(i)); !ok {
			t.Fatalf("client %d refused before the table was full", i)
		}
	}
	if ok, _ := l.allow("one-more"); ok {
		t.Fatal("a full table must refuse new clients")
	}
	// Once the buckets have refilled, the sweep drops them.
	clock = clock.Add(time.Hour)
	if ok, _ := l.allow("one-more"); !ok || len(l.buckets) != 1 {
		t.Fatalf("sweep did not free the table: %d entries", len(l.buckets))
	}
}

func TestNilRateLimiterAllows(t *testing.T) {
	var l *RateLimiter
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	rec := httptest.NewRecorder()
	l.Limit(next).ServeHTTP(rec, httptest.NewRequest("POST", "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("nil limiter: %d", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "172.18.0.3:51000"
	req.Header.Add("X-Forwarded-For", "10.9.9.9, 198.51.100.7")

	if got := ClientIP(false)(req); got != "172.18.0.3" {
		t.Errorf("untrusted proxy must ignore X-Forwarded-For: %s", got)
	}
	if got := ClientIP(true)(req); got != "198.51.100.7" {
		t.Errorf("trusted proxy uses the last forwarded address: %s", got)
	}
	req.Header.Del("X-Forwarded-For")
	if got := ClientIP(true)(req); got != "172.18.0.3" {
		t.Errorf("no header falls back to the peer: %s", got)
	}
}
