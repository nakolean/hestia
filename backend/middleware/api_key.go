package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"artemis/models"
)

// ── Key Generation ──

func GenerateRawKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return fmt.Sprintf("sk-%s", hex.EncodeToString(bytes)), nil
}

func HashKey(rawKey string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(rawKey), 12)
}

// ── Timestamp helpers ──

func parseTimestamp(raw []byte) *time.Time {
	if len(raw) == 0 {
		return nil
	}
	formats := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range formats {
		t, err := time.Parse(layout, string(raw))
		if err == nil {
			return &t
		}
	}
	return nil
}

// ── Key Lookup ──

func findKeyByRawKey(db *sql.DB, rawKey string) (*models.APIKey, error) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, key_hash, name, permissions, ip_whitelist, expires_at, last_used_at, created_at FROM api_keys",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key models.APIKey
		var rawHash, rawExpires, rawLastUsed, rawIP []byte
		if err := rows.Scan(
			&key.ID, &rawHash, &key.Name, &key.Permissions, &rawIP, &rawExpires, &rawLastUsed, &key.CreatedAt,
		); err != nil {
			return nil, err
		}

		key.ExpiresAt = parseTimestamp(rawExpires)
		key.LastUsedAt = parseTimestamp(rawLastUsed)
		if len(rawIP) > 0 {
			key.IPWhitelist = string(rawIP)
		}

		if bcrypt.CompareHashAndPassword(rawHash, []byte(rawKey)) == nil {
			return &key, nil
		}
	}
	return nil, nil
}

func listAPIKeys(db *sql.DB) ([]models.APIKey, error) {
	rows, err := db.QueryContext(
		context.Background(),
		"SELECT id, name, permissions, ip_whitelist, expires_at, last_used_at, created_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var key models.APIKey
		var rawExpires, rawLastUsed, rawIP []byte
		if err := rows.Scan(&key.ID, &key.Name, &key.Permissions, &rawIP, &rawExpires, &rawLastUsed, &key.CreatedAt); err != nil {
			return nil, err
		}
		key.ExpiresAt = parseTimestamp(rawExpires)
		key.LastUsedAt = parseTimestamp(rawLastUsed)
		if len(rawIP) > 0 {
			key.IPWhitelist = string(rawIP)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func updateLastUsed(db *sql.DB, keyID int) error {
	_, err := db.ExecContext(context.Background(), "UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), keyID)
	return err
}

// ── Auth Middleware ──

func RequireAPIKey(db *sql.DB, limiter *RateLimiter, defaultTokens float64, defaultRefill float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"missing api key"}`)
				return
			}

			if !strings.HasPrefix(rawKey, "sk-") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"invalid api key format"}`)
				return
			}

			// Rate limiting
			if limiter != nil {
				allowed, retryAfter := limiter.Allow(rawKey, defaultTokens, defaultRefill)
				if !allowed {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
					w.WriteHeader(http.StatusTooManyRequests)
					fmt.Fprintln(w, `{"error":"rate limit exceeded"}`)
					return
				}
			}

			// Lookup by bcrypt hash comparison
			key, err := findKeyByRawKey(db, rawKey)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `{"error":"internal error"}`)
				return
			}
			if key == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"invalid api key"}`)
				return
			}

			// Expiry check
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintln(w, `{"error":"api key expired"}`)
				return
			}

			// IP whitelist check
			if key.IPWhitelist != "" {
				clientIP := extractIP(r)
				if !checkIPWhitelist(key.IPWhitelist, clientIP) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprintln(w, `{"error":"ip not whitelisted"}`)
					return
				}
			}

			// Update last used
			_ = updateLastUsed(db, key.ID)

			// Store key in context
			ctx := context.WithValue(r.Context(), apiKeyContextKey{}, key)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission checks that the authenticated API key has the required permission level.
func RequirePermission(level string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getAPIKeyFromContext(r.Context())
			if key == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error":"unauthorized"}`)
				return
			}

			if !hasPermission(key.Permissions, level) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintln(w, `{"error":"forbidden: insufficient permissions"}`)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasPermission(keyPerm, required string) bool {
	key := strings.ToLower(strings.TrimSpace(keyPerm))
	req := strings.ToLower(strings.TrimSpace(required))
	if key == "full" {
		return true
	}
	if key == "write" && (req == "read" || req == "write") {
		return true
	}
	return key == req
}

func extractIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

func checkIPWhitelist(whitelist, clientIP string) bool {
	clientParsed := net.ParseIP(clientIP)
	if clientParsed == nil {
		return false
	}
	clientIP = clientParsed.String()

	for _, entry := range strings.Split(whitelist, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err != nil {
				continue
			}
			if ipNet.Contains(clientParsed) {
				return true
			}
		} else {
			parsed := net.ParseIP(entry)
			if parsed != nil && parsed.String() == clientIP {
				return true
			}
		}
	}
	return false
}

// ── Context key ──

type apiKeyContextKey struct{}

// getAPIKeyFromContext retrieves the API key from the request context.
func getAPIKeyFromContext(ctx context.Context) *models.APIKey {
	val := ctx.Value(apiKeyContextKey{})
	if val == nil {
		return nil
	}
	key, _ := val.(*models.APIKey)
	return key
}

// ── Rate Limiter ──

// RateLimiter implements a per-key token-bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*Bucket
}

// Bucket represents the token bucket for a single API key.
type Bucket struct {
	tokens   float64
	max      float64
	refill   float64 // tokens per second
	lastTime time.Time
}

// NewRateLimiter creates a new rate limiter. The cleanup goroutine runs periodically.
func NewRateLimiter(cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*Bucket),
	}
	go rl.cleanupLoop(cleanupInterval)
	return rl
}

// Allow checks if the given key is allowed to proceed.
// maxTokens and refill are the bucket parameters for this request.
func (rl *RateLimiter) Allow(key string, maxTokens, refill float64) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &Bucket{
			tokens:   maxTokens,
			max:      maxTokens,
			refill:   refill,
			lastTime: time.Now(),
		}
		rl.buckets[key] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += elapsed * bucket.refill
	if bucket.tokens > bucket.max {
		bucket.tokens = bucket.max
	}
	bucket.lastTime = now

	if bucket.tokens < 1 {
		retryAfter := time.Duration(float64(time.Second) * (1 - bucket.tokens) / bucket.refill)
		return false, retryAfter
	}
	bucket.tokens--
	return true, 0
}

func (rl *RateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rl.mu.Lock()
		for k, bucket := range rl.buckets {
			if time.Since(bucket.lastTime) > 10*time.Minute {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// AllowMethod restricts routes to specific HTTP methods.
func AllowMethod(allowed ...string) func(http.Handler) http.Handler {
	methodSet := make(map[string]bool)
	for _, method := range allowed {
		methodSet[method] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !methodSet[r.Method] {
				w.Header().Set("Allow", "GET, OPTIONS")
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintln(w, `{"error":"method not allowed"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
