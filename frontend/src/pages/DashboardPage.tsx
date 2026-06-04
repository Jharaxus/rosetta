import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../hooks/useAuth'
import { updateAssimilNumber } from '../api/auth'
import styles from './DashboardPage.module.css'

export function DashboardPage() {
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: updateAssimilNumber,
    onSuccess: (updated) => {
      queryClient.setQueryData(['auth', 'me'], updated)
    },
  })

  const current = user?.assimil_number ?? 1

  function adjust(delta: number) {
    const next = Math.min(100, Math.max(1, current + delta))
    if (next !== current) mutation.mutate(next)
  }

  async function handleLogout() {
    const res = await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' })
    if (res.ok) {
      const { redirect } = await res.json() as { redirect: string }
      window.location.href = redirect
    } else {
      window.location.href = '/'
    }
  }

  const name = user?.display_name ?? user?.email ?? ''

  return (
    <main className={styles.page}>
      <div className={styles.topBar}>
        <div className={styles.logoMark}>
          <div className={styles.logoDot} />
          <span className={styles.logoText}>Rosetta</span>
        </div>
        <button type="button" className={styles.logoutBtn} onClick={handleLogout}>
          déconnexion
        </button>
      </div>

      <div className={styles.greeting}>
        <div className={styles.greetingLabel}>AUJOURD'HUI</div>
        <h1 className={styles.greetingTitle}>
          Bonjour,<br />
          <em>{name}</em>.
        </h1>
        <p className={styles.greetingSubtitle}>
          Continuez là où vous vous êtes arrêté.
        </p>
      </div>

      <div className={styles.card}>
        <div className={styles.cardLabel}>ASSIMIL · LEÇON DU JOUR</div>
        <p className={styles.cardTitle}>Progression en cours</p>

        <div className={styles.lessonRow}>
          <span className={styles.lessonNumber}>{current}</span>
          <span className={styles.lessonOf}>/ 100</span>
        </div>

        <div className={styles.progressBar}>
          <div className={styles.progressFill} style={{ width: `${current}%` }} />
        </div>

        <div className={styles.stepper}>
          <button
            type="button"
            className={styles.btnStepper}
            onClick={() => adjust(-1)}
            disabled={current <= 1 || mutation.isPending}
            aria-label="Leçon précédente"
          >
            −
          </button>
          <button
            type="button"
            className={styles.btnStepper}
            onClick={() => adjust(1)}
            disabled={current >= 100 || mutation.isPending}
            aria-label="Leçon suivante"
          >
            +
          </button>
        </div>
      </div>

      <Link to="/flashcards" className={styles.flashCardsBtn}>
        Cartes mémoire <span className={styles.arrow}>→</span>
      </Link>

      <Link to="/writing-flashcards" className={styles.writingCardsBtn}>
        Écriture <span className={styles.arrow}>→</span>
      </Link>

      <Link to="/numbers" className={styles.numbersBtn}>
        Nombres <span className={styles.arrow}>→</span>
      </Link>

      <Link to="/profile" className={styles.profileLink}>
        Mon profil →
      </Link>
    </main>
  )
}
