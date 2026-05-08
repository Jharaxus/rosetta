import { Navigate, Link } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import styles from './LandingPage.module.css'

export function LandingPage() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) return <div aria-hidden="true" />
  if (isAuthenticated) return <Navigate to="/dashboard" replace />

  return (
    <main className={styles.page}>
      <div className={styles.inner}>
        <div className={styles.logo}>
          <div className={styles.logoDot} />
          <span className={styles.logoText}>Rosetta</span>
        </div>

        <div className={styles.hero}>
          <p className={styles.tagline}>
            Apprenez l'allemand en dix minutes par jour.
          </p>
        </div>

        <div className={styles.actions}>
          {/* Plain anchor so the browser follows the 302 redirect to Keycloak */}
          <a href="/api/auth/login" style={{ width: '100%' }}>
            <button type="button" className={styles.btnPrimary}>
              Se connecter via SSO
              <span className={styles.btnArrow}>→</span>
            </button>
          </a>

          <p className={styles.registerLink}>
            Pas encore de compte ?{' '}
            <Link to="/register">Créer un compte →</Link>
          </p>
        </div>
      </div>
    </main>
  )
}
