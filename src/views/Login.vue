<template>
  <div class="login-root">
    <div class="login-card">
      <div class="brand">
        <span class="shield">🛡️</span>
        <h1>DMARCguard</h1>
        <p class="subtitle">Sign in to your dashboard</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="field">
          <label for="username">Username</label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            autocomplete="username"
            placeholder="admin"
            :disabled="loading"
            required
          />
        </div>

        <div class="field">
          <label for="password">Password</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            :disabled="loading"
            required
          />
        </div>

        <div v-if="error" class="error-msg" role="alert">
          {{ error }}
        </div>

        <button type="submit" class="btn-login" :disabled="loading">
          <span v-if="loading" class="spinner" aria-hidden="true" />
          <span>{{ loading ? 'Signing in…' : 'Sign in' }}</span>
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const body = new URLSearchParams({
      username: form.value.username,
      password: form.value.password,
    })
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      error.value = data.error || 'Invalid username or password.'
      return
    }
    // Redirect to dashboard (or the page they tried to visit)
    const redirect = router.currentRoute.value.query.redirect || '/'
    router.replace(redirect)
  } catch (e) {
    error.value = 'Network error. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-root {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f1117;
  font-family: 'Inter', 'Segoe UI', system-ui, sans-serif;
}

.login-card {
  background: #1a1d27;
  border: 1px solid #2a2d3e;
  border-radius: 16px;
  padding: 2.5rem 2rem;
  width: 100%;
  max-width: 380px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.brand {
  text-align: center;
  margin-bottom: 2rem;
}

.shield {
  font-size: 2.5rem;
  display: block;
  margin-bottom: 0.5rem;
}

.brand h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: #e2e8f0;
  letter-spacing: -0.02em;
}

.subtitle {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
  color: #64748b;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.field label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: #94a3b8;
}

.field input {
  background: #0f1117;
  border: 1px solid #2a2d3e;
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 0.9375rem;
  padding: 0.625rem 0.875rem;
  outline: none;
  transition: border-color 0.15s;
}

.field input:focus {
  border-color: #3b82f6;
}

.field input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error-msg {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  color: #f87171;
  font-size: 0.8125rem;
  padding: 0.625rem 0.875rem;
}

.btn-login {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  font-size: 0.9375rem;
  font-weight: 600;
  padding: 0.75rem;
  transition: background 0.15s, opacity 0.15s;
}

.btn-login:hover:not(:disabled) {
  background: #2563eb;
}

.btn-login:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
