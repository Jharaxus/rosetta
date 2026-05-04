import { useAuth } from '../hooks/useAuth'
import { LogoutButton } from '../components/LogoutButton'

export function DashboardPage() {
  const { user } = useAuth()

  return (
    <main>
      <h1>Hello World, {user?.display_name ?? user?.email}!</h1>
      <LogoutButton />
    </main>
  )
}
