package api

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// cors allows browser requests from the configured origins (e.g. a separate
// frontend). With no origins configured it adds no headers, so only
// same-origin pages (like the built-in dashboard) can call the API.
func cors(allowed []string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		if origin = strings.TrimRight(strings.TrimSpace(origin), "/"); origin != "" {
			set[origin] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := set[origin]; ok && origin != "" {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			h.Set("Access-Control-Max-Age", "600")
			h.Add("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// rateLimiter is a per-client token bucket: each client gets `burst`
// requests up front, refilled at `perSecond`. Idle buckets are swept so
// memory doesn't grow with every visitor.
type rateLimiter struct {
	mu        sync.Mutex
	perSecond float64
	burst     float64
	buckets   map[string]*bucket
	lastSweep time.Time
	now       func() time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func newRateLimiter(perSecond float64, burst int) *rateLimiter {
	return &rateLimiter{
		perSecond: perSecond,
		burst:     float64(burst),
		buckets:   make(map[string]*bucket),
		now:       time.Now,
	}
}

func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	if now.Sub(r.lastSweep) > time.Minute {
		for k, b := range r.buckets {
			if now.Sub(b.seen) > 10*time.Minute {
				delete(r.buckets, k)
			}
		}
		r.lastSweep = now
	}

	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{tokens: r.burst, seen: now}
		r.buckets[key] = b
	}
	b.tokens += now.Sub(b.seen).Seconds() * r.perSecond
	if b.tokens > r.burst {
		b.tokens = r.burst
	}
	b.seen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (r *rateLimiter) middleware(trustProxy bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.allow(clientIP(c.Request, trustProxy)) {
			c.Header("Retry-After", "1")
			respondError(c, http.StatusTooManyRequests, "too many requests, slow down")
			c.Abort()
			return
		}
		c.Next()
	}
}

// clientIP identifies the caller. Behind a hosting proxy (trustProxy), the
// socket address is the proxy, so use the last X-Forwarded-For entry: the
// one our proxy appended. Entries to its left can be forged by the client.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}
