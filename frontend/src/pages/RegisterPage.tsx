import { useState, useRef, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { register, ApiError } from '../api/auth'
import styles from './RegisterPage.module.css'
import 'cap-widget'

// When VITE_CAP_SITE_KEY="dev-bypass" the widget is hidden and a placeholder is shown.
// Backend verification still runs — it hits the cap-mock service in compose.dev.yml,
// which always returns {success:true}. No bypass logic exists in the Go binary.
const CAP_DEV_BYPASS = import.meta.env.VITE_CAP_SITE_KEY === 'not-used-in-dev'
const CAP_ENDPOINT = `/cap-api/${import.meta.env.VITE_CAP_SITE_KEY}/`

function validateDisplayName(v: string): string | null {
  if (!v) return 'Nom requis'
  if (v.length < 2) return 'Au moins 2 caractères'
  if (v.length > 100) return '100 caractères maximum'
  return null
}

function validateEmail(v: string): string | null {
  if (!v) return 'Email requis'
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return 'Adresse email invalide'
  return null
}

function validatePassword(v: string): string | null {
  if (!v) return 'Mot de passe requis'
  if (v.length < 8) return 'Au moins 8 caractères'
  return null
}

function validateConfirm(password: string, confirm: string): string | null {
  if (!confirm) return 'Confirmez votre mot de passe'
  if (confirm !== password) return 'Les mots de passe ne correspondent pas'
  return null
}

export function RegisterPage() {
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [touched, setTouched] = useState<Record<string, boolean>>({})
  const [serverError, setServerError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [capToken, setCapToken] = useState<string | null>(CAP_DEV_BYPASS ? 'dev-bypass' : null)
  const [capError, setCapError] = useState<string | null>(null)
  const [capKey, setCapKey] = useState(0)
  const capRef = useRef<HTMLElement>(null)

  useEffect(() => {
    if (CAP_DEV_BYPASS) return
    const el = capRef.current
    if (!el) return
    const handler = (e: Event) => {
      setCapToken((e as CustomEvent<{ token: string }>).detail.token)
      setCapError(null)
    }
    el.addEventListener('solve', handler)
    return () => el.removeEventListener('solve', handler)
  }, [])

  const errors = {
    displayName: validateDisplayName(displayName),
    email: validateEmail(email),
    password: validatePassword(password),
    confirm: validateConfirm(password, confirm),
  }
  const isFormValid = Object.values(errors).every((e) => e === null)

  const touch = (field: string) => setTouched((t) => ({ ...t, [field]: true }))

  const mutation = useMutation({
    mutationFn: () => register({ email, display_name: displayName, password, cap_token: capToken ?? '' }),
    onSuccess: () => {
      setSuccess(true)
      setServerError(null)
    },
    onError: (err: unknown) => {
      // The cap token was consumed by the backend even when registration fails.
      // Force the user to solve the CAPTCHA again before retrying.
      setCapToken(null)
      setCapKey((k) => k + 1)
      if (err instanceof ApiError && err.status === 409) {
        setServerError('Cette adresse email est déjà utilisée.')
      } else {
        setServerError('Une erreur est survenue. Veuillez réessayer.')
      }
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setTouched({ displayName: true, email: true, password: true, confirm: true })
    if (!isFormValid) return
    if (!capToken) {
      setCapError('Veuillez compléter la vérification.')
      return
    }
    setServerError(null)
    mutation.mutate()
  }

  if (success) {
    return (
      <main className={styles.page}>
        <div className={styles.successCard}>
          <div className={styles.successMark}>✓</div>
          <h1 className={styles.successTitle}>
            Compte créé,<br /><em>{displayName}</em>.
          </h1>
          <p className={styles.successSubtitle}>
            Votre compte est prêt. Connectez-vous pour commencer.
          </p>
          <a href="/api/auth/login">
            <button type="button" className={styles.btnPrimary}>
              Se connecter
              <span className={styles.btnArrow}>→</span>
            </button>
          </a>
        </div>
      </main>
    )
  }

  return (
    <main className={styles.page}>
      <div className={styles.card}>
        <div className={styles.header}>
          <div className={styles.stepLabel}>NOUVEAU COMPTE</div>
          <h1 className={styles.title}>Créer un compte</h1>
        </div>

        <form onSubmit={handleSubmit} noValidate className={styles.form}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="displayName">Prénom ou surnom</label>
            <input
              id="displayName"
              type="text"
              className={`${styles.input}${touched.displayName && errors.displayName ? ` ${styles.inputError}` : ''}`}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              onBlur={() => touch('displayName')}
              autoComplete="name"
              placeholder="Élise"
            />
            {touched.displayName && errors.displayName && (
              <span className={styles.fieldError} role="alert">{errors.displayName}</span>
            )}
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              className={`${styles.input}${touched.email && errors.email ? ` ${styles.inputError}` : ''}`}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onBlur={() => touch('email')}
              autoComplete="email"
              placeholder="elise@exemple.fr"
            />
            {touched.email && errors.email && (
              <span className={styles.fieldError} role="alert">{errors.email}</span>
            )}
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="password">Mot de passe</label>
            <input
              id="password"
              type="password"
              className={`${styles.input}${touched.password && errors.password ? ` ${styles.inputError}` : ''}`}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              onBlur={() => touch('password')}
              autoComplete="new-password"
              placeholder="8 caractères minimum"
            />
            {touched.password && errors.password && (
              <span className={styles.fieldError} role="alert">{errors.password}</span>
            )}
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="confirm">Confirmer le mot de passe</label>
            <input
              id="confirm"
              type="password"
              className={`${styles.input}${touched.confirm && errors.confirm ? ` ${styles.inputError}` : ''}`}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              onBlur={() => touch('confirm')}
              autoComplete="new-password"
              placeholder="Répétez votre mot de passe"
            />
            {touched.confirm && errors.confirm && (
              <span className={styles.fieldError} role="alert">{errors.confirm}</span>
            )}
          </div>

          <div className={styles.field}>
            <label className={styles.label}>Vérification</label>
            {CAP_DEV_BYPASS ? (
              <div className={styles.capBypass}>
                Captcha désactivé — mode développement
              </div>
            ) : (
              <>
                <cap-widget
                  key={capKey}
                  ref={capRef}
                  className={styles.capWidget}
                  data-cap-api-endpoint={CAP_ENDPOINT}
                />
                {capError && (
                  <span className={styles.fieldError} role="alert">{capError}</span>
                )}
              </>
            )}
          </div>

          {serverError && (
            <p className={styles.serverError} role="alert">{serverError}</p>
          )}

          <div className={styles.actions}>
            <Link to="/">
              <button type="button" className={styles.btnGhost}>retour</button>
            </Link>
            <button type="submit" className={styles.btnGold} disabled={mutation.isPending}>
              {mutation.isPending ? 'Création…' : 'Créer mon compte'}
              {!mutation.isPending && <span className={styles.btnArrow}>→</span>}
            </button>
          </div>
        </form>

        <p className={styles.signInLink}>
          Déjà inscrit ? <Link to="/">Se connecter →</Link>
        </p>
      </div>
    </main>
  )
}
