import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../hooks/useAuth'
import { LogoutButton } from '../components/LogoutButton'
import { updateAssimilNumber } from '../api/auth'

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

  return (
    <main>
      <h1>Hello World, {user?.display_name ?? user?.email}!</h1>

      <section>
        <p>Assimil lesson: {current} / 100</p>
        <button type="button" onClick={() => adjust(-1)} disabled={current <= 1 || mutation.isPending}>
          −
        </button>
        <button type="button" onClick={() => adjust(1)} disabled={current >= 100 || mutation.isPending}>
          +
        </button>
      </section>

      <Link to="/profile">Edit profile</Link>
      <LogoutButton />
    </main>
  )
}
