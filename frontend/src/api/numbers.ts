import { ApiError } from './auth'

async function request(path: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(path, { credentials: 'include', ...options })
  if (!res.ok) {
    let code = 'unknown'
    try {
      const body = await res.json()
      code = body.code ?? body.error ?? code
    } catch {}
    throw new ApiError(res.status, code)
  }
  return res
}

export async function fetchPracticeNumber(digits: number): Promise<number> {
  const res = await request(`/api/numbers/practice?digits=${digits}`)
  const body = await res.json()
  return body.number as number
}

export async function submitDigitSuccess(number: number): Promise<void> {
  await request('/api/numbers/digit-success', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ number }),
  })
}

export async function fetchNextFailure(): Promise<number | null> {
  try {
    const res = await request('/api/numbers/failures/next')
    const body = await res.json()
    return body.number as number
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function addNumberFailure(number: number): Promise<void> {
  await request('/api/numbers/failures', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ number }),
  })
}

export async function removeNumberFailure(number: number): Promise<void> {
  await request(`/api/numbers/failures/${number}`, { method: 'DELETE' })
}

export async function fetchNumberSettings(): Promise<{ number_digit_size: number }> {
  const res = await request('/api/user/settings')
  return res.json()
}

export async function updateNumberSettings(digitSize: number): Promise<void> {
  await request('/api/user/settings', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ number_digit_size: digitSize }),
  })
}
