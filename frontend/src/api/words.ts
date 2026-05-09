import type { Word } from '../types/words'
import { ApiError } from './auth'

export async function fetchFlashCard(): Promise<Word> {
  const res = await fetch('/api/words/flashcard', { credentials: 'include' })
  if (res.status === 404) {
    throw new ApiError(404, 'no_cards_available')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
  return res.json() as Promise<Word>
}
