package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type policy struct {
	name   string
	burst  float64
	period time.Duration
}

type bucket struct {
	tokens  float64
	updated time.Time
	seen    time.Time
}

type limiter struct {
	mu          sync.Mutex
	buckets     map[string]*bucket
	lastCleanup time.Time
}

const maxBuckets = 25000

func RateLimit(next http.Handler) http.Handler {
	limiter := &limiter{buckets: make(map[string]*bucket)}
	// Behind a proxy every request shares its address, which would make all users share one limit.
	trustProxy := os.Getenv("KYMBER_TRUST_PROXY") == "true"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, trustProxy)
		if !strings.HasPrefix(r.URL.Path, "/api/agent/") {
			if ok, wait := limiter.allow(ip+"/all", policy{name: "all", burst: 120, period: time.Minute}); !ok {
				reject(w, wait)
				return
			}
		}
		if p, ok := sensitivePolicy(r); ok {
			if allowed, wait := limiter.allow(ip+"/"+p.name, p); !allowed {
				reject(w, wait)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func sensitivePolicy(r *http.Request) (policy, bool) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/onboarding":
		return policy{name: "onboarding", burst: 3, period: time.Minute}, true
	case r.Method == http.MethodPost && r.URL.Path == "/api/auth/login":
		return policy{name: "login", burst: 5, period: time.Minute}, true
	case r.Method == http.MethodPost && (r.URL.Path == "/api/auth/register" || r.URL.Path == "/api/invitations/accept"):
		return policy{name: "account-create", burst: 5, period: time.Minute}, true
	case r.Method == http.MethodPut && (r.URL.Path == "/api/account/password" || r.URL.Path == "/api/account/profile"):
		return policy{name: "account-change", burst: 5, period: time.Minute}, true
	case r.Method == http.MethodPut && r.URL.Path == "/api/settings/registration":
		return policy{name: "platform-settings", burst: 5, period: time.Minute}, true
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/invitations"):
		return policy{name: "send-invitation", burst: 10, period: time.Minute}, true
	case strings.HasPrefix(r.URL.Path, "/api/agent/"):
		return policy{name: "agent", burst: 180, period: time.Minute}, true
	case isWorkspaceMembersPath(r.URL.Path) && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete):
		return policy{name: "workspace-members", burst: 10, period: time.Minute}, true
	case r.Method == http.MethodDelete && isWorkspacePath(r.URL.Path):
		return policy{name: "workspace-delete", burst: 3, period: time.Minute}, true
	case r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete:
		return policy{name: "write", burst: 60, period: time.Minute}, true
	default:
		return policy{}, false
	}
}

func isWorkspaceMembersPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) >= 4 && parts[0] == "api" && parts[1] == "workspaces" && parts[3] == "members"
}

func isWorkspacePath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "api" && parts[1] == "workspaces"
}

func (l *limiter) allow(key string, p policy) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b == nil {
		if len(l.buckets) >= maxBuckets {
			if now.Sub(l.lastCleanup) > time.Minute {
				for existing, candidate := range l.buckets {
					if now.Sub(candidate.seen) > 10*time.Minute {
						delete(l.buckets, existing)
					}
				}
				l.lastCleanup = now
			}
		}
		if len(l.buckets) >= maxBuckets {
			key = "overflow/" + p.name
			b = l.buckets[key]
		}
		if b == nil {
			b = &bucket{tokens: p.burst, updated: now, seen: now}
			l.buckets[key] = b
		}
	}
	if elapsed := now.Sub(b.updated); elapsed > 0 {
		b.tokens = math.Min(p.burst, b.tokens+float64(elapsed)/float64(p.period)*p.burst)
		b.updated = now
	}
	b.seen = now
	if b.tokens < 1 {
		return false, time.Duration(math.Ceil(float64(time.Second) * (1 - b.tokens) * float64(p.period) / p.burst))
	}
	b.tokens--
	return true, 0
}

// The nearest proxy appends the address it saw last, so entries further left can be forged by the client.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if entries := strings.Split(r.Header.Get("X-Forwarded-For"), ","); net.ParseIP(strings.TrimSpace(entries[len(entries)-1])) != nil {
			return strings.TrimSpace(entries[len(entries)-1])
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func reject(w http.ResponseWriter, wait time.Duration) {
	seconds := int(math.Ceil(wait.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
}
