import { Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

export function LandingPage() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) return <div aria-hidden="true" />
  if (isAuthenticated) return <Navigate to="/dashboard" replace />

  return (
    <main>
      <h1>Welcome to Rosetta</h1>
      <p>Sign in to continue.</p>
      {/* Plain anchor navigation so the browser follows the 302 redirect to Keycloak. */}
      <a href="/api/auth/login">
        <button type="button">Sign in with SSO</button>
      </a>
    </main>
  )
}
