import type { User } from '../types/auth'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
  ) {
    super(code)
  }
}

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

export async function register(payload: {
  email: string
  display_name: string
  password: string
}): Promise<User> {
  const res = await fetch('/api/auth/register', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
  return res.json() as Promise<User>
}

export async function updateAssimilNumber(assimilNumber: number): Promise<User> {
  const res = await fetch('/api/user/profile', {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ assimil_number: assimilNumber }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
  return res.json() as Promise<User>
}
