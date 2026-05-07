# Adding Login to DMARCguard – Integration Guide

## Overview

This adds a **simple username/password login page** to DMARCguard using:
- A Go session-cookie middleware (`internal/auth/middleware.go`)
- Two new API routes: `POST /api/login` and `POST /api/logout`
- A Vue 3 login page (`src/views/Login.vue`)
- Two new env vars: `AUTH_USERNAME` and `AUTH_PASSWORD`

Auth is opt-in: if neither env var is set, the middleware is a no-op and
existing deployments keep working unchanged.

---

## Step 1 – Copy the new files

```
# From this repo's root:
cp <these-files>/internal/auth/middleware.go  internal/auth/middleware.go
cp <these-files>/src/views/Login.vue          src/views/Login.vue
cp <these-files>/render.yaml                  render.yaml
```

---

## Step 2 – Wire the auth middleware into the Go server

### 2a. Edit `main.go`

Add the import:
```go
"github.com/meysam81/parse-dmarc/internal/auth"
```

Find where `api.NewServer(...)` is called and add ONE line after it:
```go
server := api.NewServer(store, cfg.Server.Host, cfg.Server.Port, m, log)
server.WithAuth(auth.Middleware, auth.LoginHandler, auth.LogoutHandler)  // ← add this
```

### 2b. Copy `internal/api/auth_routes.go`

This file adds the `WithAuth()` method to the server struct without
modifying any existing file.

**Important:** open `internal/api/server.go` and find the name of the
HTTP mux field. It might be `s.mux`, `s.router`, `s.r`, or `s.handler`.
Update the two `s.mux.HandleFunc(...)` lines in `auth_routes.go` to match.

Also add the `authMiddleware` field to the `Server` struct in `server.go`:
```go
type Server struct {
    // ... existing fields ...
    authMiddleware func(http.Handler) http.Handler  // ← add this
}
```

### 2c. Apply the middleware in `Start()`

In `internal/api/server.go`, find where the `http.Server` is created.
It will look something like:
```go
httpServer := &http.Server{
    Handler: s.mux,   // or s.router, etc.
}
```

Wrap the handler with the middleware:
```go
var handler http.Handler = s.mux
if s.authMiddleware != nil {
    handler = s.authMiddleware(handler)
}
httpServer := &http.Server{
    Handler: handler,
}
```

---

## Step 3 – Add the Vue login route

Edit `src/router/index.js` (or wherever your router is defined):

```js
// 1. Import Login view (add at top)
import Login from '@/views/Login.vue'

// 2. Add to routes array
{
  path: '/login',
  name: 'Login',
  component: Login,
  meta: { public: true },
},

// 3. Add navigation guard (after router is created, before export)
router.beforeEach(async (to) => {
  if (to.meta.public) return true

  try {
    const res = await fetch('/api/statistics', { credentials: 'include' })
    if (res.status === 401) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  } catch (_) { /* network error – pass through */ }
  return true
})
```

---

## Step 4 – Handle 401s in API calls (optional but recommended)

In any Vue composable or store that calls `/api/*`, add a global handler:

```js
// src/utils/api.js  (create or add to existing)
export async function apiFetch(url, opts = {}) {
  const res = await fetch(url, { credentials: 'include', ...opts })
  if (res.status === 401) {
    window.location.href = '/login'
    return
  }
  return res
}
```

Replace `fetch('/api/...')` calls with `apiFetch('/api/...')`.

---

## Step 5 – Add a Logout button

In your main layout/nav component, add:

```vue
<button @click="logout">Sign out</button>

<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()

async function logout() {
  await fetch('/api/logout', { method: 'POST', credentials: 'include' })
  router.push('/login')
}
</script>
```

---

## Step 6 – Deploy to Render (free tier)

1. Push your changes to GitHub.
2. In the Render dashboard, **before** first deploy, set env vars:
   - `AUTH_USERNAME` → your chosen username
   - `AUTH_PASSWORD` → a strong password (use a password manager)
3. The `render.yaml` already has `plan: free` and `sync: false` for
   the credentials, so they are never committed to the repo.

> **Note:** Render free tier containers spin down after ~15 minutes of
> inactivity. Sessions are in-memory, so users will need to log in again
> after a cold start. This is expected behaviour for free tier.

---

## Security notes

- Sessions are stored in-memory and expire after 24 hours.
- Passwords are compared via SHA-256 to avoid naive timing attacks.
- The `/api/statistics` path is exempt (Render health check).
- Set `AUTH_PASSWORD` to something strong (≥16 random chars).
- For production use consider a proper auth solution (OAuth, etc.).
