import type { Rating } from '../types/words'

// Standard O(n·m) DP Levenshtein distance, Unicode-aware (iterates code points).
// Comparison is case-sensitive: teaches German noun capitalisation.
export function levenshtein(a: string, b: string): number {
  const sa = Array.from(a)
  const sb = Array.from(b)
  const m = sa.length
  const n = sb.length

  // Single-row optimisation: we only need the previous and current row.
  let prev = Array.from({ length: n + 1 }, (_, i) => i)
  let curr = new Array<number>(n + 1)

  for (let i = 1; i <= m; i++) {
    curr[0] = i
    for (let j = 1; j <= n; j++) {
      const cost = sa[i - 1] === sb[j - 1] ? 0 : 1
      curr[j] = Math.min(
        prev[j] + 1,      // deletion
        curr[j - 1] + 1,  // insertion
        prev[j - 1] + cost, // substitution
      )
    }
    ;[prev, curr] = [curr, prev]
  }

  return prev[n]
}

// accuracy = 1 - lev(input, correct) / max(|input|, |correct|)
// Returns 1.0 when both strings are empty.
export function computeAccuracy(input: string, correct: string): number {
  const aLen = Array.from(input).length
  const bLen = Array.from(correct).length
  const maxLen = Math.max(aLen, bLen)
  if (maxLen === 0) return 1
  return 1 - levenshtein(input, correct) / maxLen
}

// Maps accuracy to an FSRS rating using the agreed thresholds:
//   [0, 0.75)  → Again (1)
//   [0.75, 0.9) → Hard (2)
//   [0.9, 1.0)  → Good (3)
//   1.0         → Easy (4)
export function accuracyToRating(accuracy: number): Rating {
  if (accuracy >= 1) return 4
  if (accuracy >= 0.9) return 3
  if (accuracy >= 0.75) return 2
  return 1
}
