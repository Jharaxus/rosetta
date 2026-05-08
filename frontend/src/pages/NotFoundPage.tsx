import { Link } from 'react-router-dom'
import styles from './NotFoundPage.module.css'

export function NotFoundPage() {
  return (
    <main className={styles.page}>
      <div className={styles.inner}>
        <Link to="/" className={styles.logo}>
          <div className={styles.logoDot} />
          <span className={styles.logoText}>Rosetta</span>
        </Link>

        <div className={styles.hero}>
          <p className={styles.errorLabel}>ERREUR · 404</p>
          <h1 className={styles.number}>
            <em>404</em>
          </h1>
          <p className={styles.title}>Cette page n'existe pas.</p>
          <p className={styles.subtitle}>
            La page que vous cherchez a été déplacée ou n'a jamais existé.
          </p>
        </div>

        <Link to="/" className={styles.btnHome}>
          Retour à l'accueil
          <span className={styles.btnArrow}>→</span>
        </Link>
      </div>
    </main>
  )
}
