import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { register, ApiError } from '../api/auth'

function validateDisplayName(v: string): string | null {
  if (!v) return 'Display name is required'
  if (v.length < 2) return 'Must be at least 2 characters'
  if (v.length > 100) return 'Must be 100 characters or fewer'
  return null
}

function validateEmail(v: string): string | null {
  if (!v) return 'Email is required'
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return 'Must be a valid email address'
  return null
}

function validatePassword(v: string): string | null {
  if (!v) return 'Password is required'
  if (v.length < 8) return 'Must be at least 8 characters'
  return null
}

function validateConfirm(password: string, confirm: string): string | null {
  if (!confirm) return 'Please confirm your password'
  if (confirm !== password) return 'Passwords do not match'
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

  const errors = {
    displayName: validateDisplayName(displayName),
    email: validateEmail(email),
    password: validatePassword(password),
    confirm: validateConfirm(password, confirm),
  }
  const isFormValid = Object.values(errors).every((e) => e === null)

  const touch = (field: string) => setTouched((t) => ({ ...t, [field]: true }))

  const mutation = useMutation({
    mutationFn: () => register({ email, display_name: displayName, password }),
    onSuccess: () => {
      setSuccess(true)
      setServerError(null)
    },
    onError: (err: unknown) => {
      if (err instanceof ApiError && err.status === 409) {
        setServerError('That email address is already registered.')
      } else {
        setServerError('Something went wrong. Please try again.')
      }
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setTouched({ displayName: true, email: true, password: true, confirm: true })
    if (!isFormValid) return
    setServerError(null)
    mutation.mutate()
  }

  if (success) {
    return (
      <main>
        <h1>Account created!</h1>
        <p>Your account is ready. Sign in to continue.</p>
        <a href="/api/auth/login">Sign in</a>
      </main>
    )
  }

  return (
    <main>
      <h1>Create an account</h1>
      <form onSubmit={handleSubmit} noValidate>
        <div>
          <label htmlFor="displayName">Display name</label>
          <input
            id="displayName"
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            onBlur={() => touch('displayName')}
            autoComplete="name"
          />
          {touched.displayName && errors.displayName && (
            <span role="alert">{errors.displayName}</span>
          )}
        </div>

        <div>
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onBlur={() => touch('email')}
            autoComplete="email"
          />
          {touched.email && errors.email && (
            <span role="alert">{errors.email}</span>
          )}
        </div>

        <div>
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onBlur={() => touch('password')}
            autoComplete="new-password"
          />
          {touched.password && errors.password && (
            <span role="alert">{errors.password}</span>
          )}
        </div>

        <div>
          <label htmlFor="confirm">Confirm password</label>
          <input
            id="confirm"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            onBlur={() => touch('confirm')}
            autoComplete="new-password"
          />
          {touched.confirm && errors.confirm && (
            <span role="alert">{errors.confirm}</span>
          )}
        </div>

        {serverError && <p role="alert">{serverError}</p>}

        <button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? 'Creating account…' : 'Create account'}
        </button>
      </form>

      <p>
        Already have an account? <Link to="/">Sign in</Link>
      </p>
    </main>
  )
}
