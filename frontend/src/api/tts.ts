import { ApiError } from './auth'

export async function synthesizeSpeech(text: string): Promise<Blob> {
  const res = await fetch('/api/tts/synthesize', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'tts_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'tts_error')
  }
  return res.blob()
}
