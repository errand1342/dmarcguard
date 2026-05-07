package api

// auth_routes.go  – drop this file into internal/api/
// It adds the login/logout routes and wraps the server's HTTP handler
// with the auth middleware WITHOUT touching any existing files.
//
// Usage in main.go (after creating the server):
//
//   import "github.com/meysam81/parse-dmarc/internal/auth"
//
//   server := api.NewServer(store, cfg.Server.Host, cfg.Server.Port, m, log)
//   server.WithAuth(auth.Middleware, auth.LoginHandler, auth.LogoutHandler)

import "net/http"

// WithAuth registers the login/logout handlers and stores the middleware
// so that Start() can apply it.  Call this immediately after NewServer().
func (s *Server) WithAuth(
	middleware func(http.Handler) http.Handler,
	loginHandler http.HandlerFunc,
	logoutHandler http.HandlerFunc,
) {
	// Store for use in Start()
	s.authMiddleware = middleware

	// Register auth API routes on whatever mux the server uses.
	// The field name below must match the actual mux field in server.go.
	// Common names: s.mux, s.router, s.handler, s.r
	// Check your server.go and adjust ONE of the lines that matches:
	s.mux.HandleFunc("/api/login", loginHandler)
	s.mux.HandleFunc("/api/logout", logoutHandler)
}
