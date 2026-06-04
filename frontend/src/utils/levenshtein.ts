import type { Rating } from '../types/words'

// ── Types ────────────────────────────────────────────────────────────────────

export type EditOp = 'match' | 'substitute' | 'delete' | 'insert'

export interface DiffToken {
  // The character to display. For 'insert' ops (a char the user missed),
  // this is '·' — a visual placeholder indicating a missing position.
  char: string
  op: EditOp
}

export interface StringDiff {
  distance: number
  // One token per display position in the user's spelling, including '·'
  // placeholders for characters the user omitted (insert ops).
  inputTokens: DiffToken[]
}

// ── Core algorithm ───────────────────────────────────────────────────────────

// O(n·m) Levenshtein with the full DP table retained for backtracking.
// Comparison is case-sensitive: teaches German noun capitalisation.
export function diffStrings(input: string, correct: string): StringDiff {
  const sa = Array.from(input)   // Unicode code-point array
  const sb = Array.from(correct)
  const m = sa.length
  const n = sb.length

  // Build the complete DP table so we can backtrack the optimal alignment.
  const dp: number[][] = Array.from({ length: m + 1 }, (_, i) =>
    Array.from({ length: n + 1 }, (_, j) => (i === 0 ? j : j === 0 ? i : 0)),
  )
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      const cost = sa[i - 1] === sb[j - 1] ? 0 : 1
      dp[i][j] = Math.min(dp[i - 1][j] + 1, dp[i][j - 1] + 1, dp[i - 1][j - 1] + cost)
    }
  }

  // Backtrack from (m, n) → (0, 0) to recover the edit sequence.
  // Priority: match/substitute (diagonal) > delete > insert.
  // This prefers aligning characters over skipping them, giving the most
  // compact and human-readable diff for typical German words.
  const ops: EditOp[] = []
  let i = m
  let j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0) {
      const cost = sa[i - 1] === sb[j - 1] ? 0 : 1
      if (dp[i][j] === dp[i - 1][j - 1] + cost) {
        ops.push(cost === 0 ? 'match' : 'substitute')
        i--
        j--
        continue
      }
    }
    if (i > 0 && dp[i][j] === dp[i - 1][j] + 1) {
      ops.push('delete')
      i--
    } else {
      ops.push('insert')
      j--
    }
  }
  ops.reverse()

  // Map edit ops to display tokens over the user's input characters.
  // Insert ops get a '·' placeholder (the user typed nothing at that position).
  const inputTokens: DiffToken[] = []
  let si = 0
  for (const op of ops) {
    inputTokens.push({ char: op === 'insert' ? '·' : sa[si++], op })
  }

  return { distance: dp[m][n], inputTokens }
}

// ── Public helpers ───────────────────────────────────────────────────────────

// accuracy = 1 - lev(input, correct) / max(|input|, |correct|)
// Returns 1.0 when both strings are empty.
export function computeAccuracy(input: string, correct: string): number {
  const maxLen = Math.max(Array.from(input).length, Array.from(correct).length)
  if (maxLen === 0) return 1
  return 1 - diffStrings(input, correct).distance / maxLen
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
