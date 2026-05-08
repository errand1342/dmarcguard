package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	cookieName = "dg_session"
	cookieTTL  = 24 * time.Hour
)

// sessionStore holds active sessions in-memory.
// On Render free tier the instance restarts ~every 15 min when idle,
// so users may be prompted to log in again after inactivity – that is fine.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

var store = &sessionStore{sessions: make(map[string]time.Time)}

func (s *sessionStore) create() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[token] = time.Now().Add(cookieTTL)
	s.mu.Unlock()
	return token
}

func (s *sessionStore) valid(token string) bool {
	s.mu.RLock()
	exp, ok := s.sessions[token]
	s.mu.RUnlock()
	return ok && time.Now().Before(exp)
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// credentialsOK checks the provided credentials against env vars
// AUTH_USERNAME and AUTH_PASSWORD.
// If neither env var is set the auth layer is effectively disabled
// (returns true always) so existing deployments without auth keep working.
func credentialsOK(username, password string) bool {
	wantUser := os.Getenv("AUTH_USERNAME")
	wantPass := os.Getenv("AUTH_PASSWORD")
	if wantUser == "" && wantPass == "" {
		return true
	}
	// Constant-time compare via SHA-256 to avoid timing attacks
	h := func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])
	}
	return h(username) == h(wantUser) && h(password) == h(wantPass)
}

// Middleware wraps an http.Handler and requires a valid session cookie.
// The /login and /api/statistics (Render health check) paths are exempt.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow the health-check path used by Render
		if r.URL.Path == "/api/statistics" {
			next.ServeHTTP(w, r)
			return
		}
		// Allow static login assets so the page can load
	if r.URL.Path == "/login" ||
		r.URL.Path == "/api/login" ||
		r.URL.Path == "/api/logout" ||
		strings.HasPrefix(r.URL.Path, "/assets/") ||
		r.URL.Path == "/favicon.ico" {
			return
		}

		// If auth is not configured, pass through
		if os.Getenv("AUTH_USERNAME") == "" && os.Getenv("AUTH_PASSWORD") == "" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(cookieName)
		if err != nil || !store.valid(cookie.Value) {
			// API requests get 401; browser requests get redirect to /login
			if isAPIRequest(r) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoginHandler handles POST /api/login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" {
		// Also support JSON body
		_ = r.ParseForm()
		username = r.PostFormValue("username")
		password = r.PostFormValue("password")
	}

	if !credentialsOK(username, password) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token := store.create()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(cookieTTL.Seconds()),
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// LogoutHandler handles POST /api/logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		store.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func isAPIRequest(r *http.Request) bool {
	return len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api"
}
