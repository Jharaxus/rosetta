import { useQuery } from '@tanstack/react-query'
import { getMe } from '../api/auth'
import type { User } from '../types/auth'

export function useAuth(): { user: User | undefined; isLoading: boolean; isAuthenticated: boolean } {
  const { data: user, isLoading, isError } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: getMe,
    retry: false,        // 401 is not transient — don't spam the server
    staleTime: 5 * 60_000,
  })

  return {
    user,
    isLoading,
    isAuthenticated: !!user && !isError,
  }
}
