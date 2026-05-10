import type { Word, Rating } from '../types/words'
import { ApiError } from './auth'

export async function fetchFlashCard(): Promise<Word> {
  const res = await fetch('/api/words/flashcard', { credentials: 'include' })
  if (res.status === 404) {
    const body = await res.json().catch(() => ({ error: 'no_cards_due' }))
    throw new ApiError(404, (body as { error?: string }).error ?? 'no_cards_due')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
  return res.json() as Promise<Word>
}

export async function clearCards(): Promise<void> {
  const res = await fetch('/api/user/cards', {
    method: 'DELETE',
    credentials: 'include',
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
}

export async function submitReview(wordId: string, rating: Rating): Promise<void> {
  const res = await fetch(`/api/words/${wordId}/review`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ rating }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: 'unknown_error' }))
    throw new ApiError(res.status, (body as { error?: string }).error ?? 'unknown_error')
  }
}
