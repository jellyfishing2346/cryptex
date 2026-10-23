package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jellyfishing2346/cryptex/internal/models"
)

func TestRateLimiterRefillsOverTime(t *testing.T) {
	now := time.Unix(1_000, 0)
	limiter := newRateLimiter(1, 2) // 1 per second, burst 2
	limiter.now = func() time.Time { return now }

	for i := 1; i <= 2; i++ {
		if !limiter.allow("a") {
			t.Fatalf("request %d of the burst should be allowed", i)
		}
	}
	if limiter.allow("a") {
		t.Fatal("third immediate request should be limited")
	}
	if !limiter.allow("b") {
		t.Fatal("clients are limited independently")
	}
	now = now.Add(1100 * time.Millisecond)
	if !limiter.allow("a") {
		t.Fatal("one token should refill after a second")
	}
}

func TestRateLimiterForgetsIdleClients(t *testing.T) {
	now := time.Unix(1_000, 0)
	limiter := newRateLimiter(1, 1)
	limiter.now = func() time.Time { return now }
	limiter.allow("idle")
	now = now.Add(11 * time.Minute)
	limiter.allow("other")
	if _, ok := limiter.buckets["idle"]; ok {
		t.Fatal("idle bucket should be swept")
	}
}

func TestClientIPIgnoresSpoofedForwardedEntries(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")

	if got := clientIP(req, true); got != "203.0.113.9" {
		t.Fatalf("behind proxy: got %q", got)
	}
	if got := clientIP(req, false); got != "10.0.0.1" {
		t.Fatalf("direct: got %q", got)
	}
	req.RemoteAddr = "[::1]:5555"
	if got := clientIP(req, false); got != "::1" {
		t.Fatalf("ipv6: got %q", got)
	}
}

func TestTradeLogKeepsNewestWithinCapacity(t *testing.T) {
	log := newTradeLog(3)
	if log.last() != nil || len(log.recent(10)) != 0 {
		t.Fatal("empty log")
	}
	for i := 1; i <= 5; i++ {
		log.add(&models.Trade{Price: float64(i)})
	}
	got := log.recent(10)
	if len(got) != 3 || got[0].Price != 5 || got[1].Price != 4 || got[2].Price != 3 {
		t.Fatalf("expected 5,4,3 got %v", []float64{got[0].Price, got[1].Price, got[2].Price})
	}
	if log.last().Price != 5 {
		t.Fatal("last should be the newest")
	}
	if len(log.recent(2)) != 2 {
		t.Fatal("limit respected")
	}
}
