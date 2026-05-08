import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../hooks/useAuth'
import { updateAssimilNumber } from '../api/auth'
import styles from './ProfilePage.module.css'

export function ProfilePage() {
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const [value, setValue] = useState(user?.assimil_number ?? 1)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (user?.assimil_number !== undefined) {
      setValue(user.assimil_number)
    }
  }, [user?.assimil_number])

  const mutation = useMutation({
    mutationFn: updateAssimilNumber,
    onSuccess: (updated) => {
      queryClient.setQueryData(['auth', 'me'], updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    mutation.mutate(value)
  }

  const inputError = value < 1 || value > 100 ? 'Entre 1 et 100' : null
  const name = user?.display_name ?? user?.email ?? ''

  return (
    <main className={styles.page}>
      <div className={styles.topBar}>
        <Link to="/dashboard" className={styles.backLink}>← retour</Link>
      </div>

      <div className={styles.header}>
        <div className={styles.pageLabel}>MON PROFIL</div>
        <h1 className={styles.pageTitle}>
          <em>{name}</em>
        </h1>
        <p className={styles.pageSubtitle}>{user?.email}</p>
      </div>

      <div className={styles.card}>
        <div className={styles.cardLabel}>ASSIMIL · LEÇON ATTEINTE</div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="assimil">Leçon en cours</label>
            <input
              id="assimil"
              type="number"
              min={1}
              max={100}
              className={`${styles.input}${inputError ? ` ${styles.inputError}` : ''}`}
              value={value}
              onChange={(e) => {
                setSaved(false)
                setValue(Number(e.target.value))
              }}
            />
            {inputError && <span className={styles.fieldError} role="alert">{inputError}</span>}
          </div>

          {mutation.isError && (
            <p className={styles.serverError} role="alert">
              Une erreur est survenue. Veuillez réessayer.
            </p>
          )}

          <div className={styles.actions}>
            <button
              type="submit"
              className={styles.btnGold}
              disabled={!!inputError || mutation.isPending}
            >
              {mutation.isPending ? 'Enregistrement…' : 'Enregistrer'}
            </button>
            {saved && <span className={styles.savedBadge}>Sauvegardé ✓</span>}
          </div>
        </form>
      </div>
    </main>
  )
}
