import { Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

interface Props {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: Props) {
  const { user, isLoading, isAuthenticated } = useAuth()

  if (isLoading) return <div aria-hidden="true" />
  if (!isAuthenticated || !user) return <Navigate to="/" replace />

  return <>{children}</>
}
