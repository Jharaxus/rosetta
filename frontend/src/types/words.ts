export interface Word {
  id: string
  french: string
  german: string
  assimil_number: number
  category: string
  is_regular: boolean | null
  audio_url: string
}

export type Rating = 1 | 2 | 3 | 4

export const RATING_LABELS: Record<Rating, string> = {
  1: 'Encore',
  2: 'Difficile',
  3: 'Bien',
  4: 'Facile',
}
