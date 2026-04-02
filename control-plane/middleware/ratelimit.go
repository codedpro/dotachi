package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterStore holds per-IP rate limiters.
type RateLimiterStore struct {
	mu       sync.Mutex
	limiters sync.Map
	rps      rate.Limit
	burst    int
}

// NewRateLimiterStore creates a store that issues limiters at the given
// requests-per-minute rate. Stale entries are cleaned every 5 minutes.
func NewRateLimiterStore(requestsPerMinute int) *RateLimiterStore {
	burst := requestsPerMinute / 3
	if burst < 3 {
		burst = 3
	}
	s := &RateLimiterStore{
		rps:   rate.Limit(float64(requestsPerMinute) / 60.0),
		burst: burst, // burst = 1/3 of per-minute cap for tighter enforcement
	}
	go s.cleanup()
	return s
}

func (s *RateLimiterStore) getLimiter(ip string) *rate.Limiter {
	val, ok := s.limiters.Load(ip)
	if ok {
		entry := val.(*ipLimiter)
		s.mu.Lock()
		entry.lastSeen = time.Now()
		s.mu.Unlock()
		return entry.limiter
	}

	limiter := rate.NewLimiter(s.rps, s.burst)
	s.limiters.Store(ip, &ipLimiter{limiter: limiter, lastSeen: time.Now()})
	return limiter
}

func (s *RateLimiterStore) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		now := time.Now()
		s.limiters.Range(func(key, value interface{}) bool {
			entry := value.(*ipLimiter)
			s.mu.Lock()
			if now.Sub(entry.lastSeen) > 5*time.Minute {
				s.limiters.Delete(key)
			}
			s.mu.Unlock()
			return true
		})
	}
}

// RateLimit returns middleware that enforces per-IP rate limiting using
// the provided store. Returns 429 Too Many Requests with a JSON body
// when the limit is exceeded.
func RateLimit(store *RateLimiterStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			// chi's RealIP middleware sets X-Real-IP / X-Forwarded-For
			if forwarded := r.Header.Get("X-Real-Ip"); forwarded != "" {
				ip = forwarded
			}
			// Strip port from address — r.RemoteAddr is "ip:port"
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip = host
			}

			limiter := store.getLimiter(ip)
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded, please try again later"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
