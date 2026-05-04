import type { User } from '../types/auth'

export async function getMe(): Promise<User> {
  const res = await fetch('/api/auth/me', { credentials: 'include' })
  if (res.status === 401) {
    throw new Error('unauthenticated')
  }
  if (!res.ok) {
    throw new Error(`unexpected status ${res.status}`)
  }
  return res.json() as Promise<User>
}
