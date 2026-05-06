import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../hooks/useAuth'
import { updateAssimilNumber } from '../api/auth'

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
      setTimeout(() => setSaved(false), 2000)
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    mutation.mutate(value)
  }

  const inputError = value < 1 || value > 100 ? 'Must be between 1 and 100' : null

  return (
    <main>
      <h1>Profile</h1>
      <p>{user?.display_name} — {user?.email}</p>

      <form onSubmit={handleSubmit}>
        <label htmlFor="assimil">Assimil lesson reached</label>
        <input
          id="assimil"
          type="number"
          min={1}
          max={100}
          value={value}
          onChange={(e) => {
            setSaved(false)
            setValue(Number(e.target.value))
          }}
        />
        {inputError && <span role="alert">{inputError}</span>}

        {mutation.isError && <p role="alert">Something went wrong. Please try again.</p>}
        {saved && <p>Saved!</p>}

        <button type="submit" disabled={!!inputError || mutation.isPending}>
          {mutation.isPending ? 'Saving…' : 'Save'}
        </button>
      </form>

      <Link to="/dashboard">Back to dashboard</Link>
    </main>
  )
}
