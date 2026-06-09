import { describe, it, expect } from 'vitest'
import { diffStrings, computeAccuracy, accuracyToRating } from './levenshtein'
import type { DiffToken } from './levenshtein'

// ── Helpers ───────────────────────────────────────────────────────────────────

function ops(tokens: DiffToken[]) {
  return tokens.map((t) => t.op)
}

function chars(tokens: DiffToken[]) {
  return tokens.map((t) => t.char)
}

// ── diffStrings ───────────────────────────────────────────────────────────────

describe('diffStrings', () => {
  // ── Empty inputs ────────────────────────────────────────────────────────────

  describe('empty inputs', () => {
    it('both empty → distance 0, no tokens', () => {
      const r = diffStrings('', '')
      expect(r.distance).toBe(0)
      expect(r.inputTokens).toEqual([])
    })

    it('empty input, non-empty correct → insert placeholder for every missing char', () => {
      const r = diffStrings('', 'ab')
      expect(r.distance).toBe(2)
      expect(r.inputTokens).toHaveLength(2)
      expect(chars(r.inputTokens)).toEqual(['·', '·'])
      expect(ops(r.inputTokens)).toEqual(['insert', 'insert'])
    })

    it('non-empty input, empty correct → delete token for every extra char', () => {
      const r = diffStrings('ab', '')
      expect(r.distance).toBe(2)
      expect(r.inputTokens).toHaveLength(2)
      expect(chars(r.inputTokens)).toEqual(['a', 'b'])
      expect(ops(r.inputTokens)).toEqual(['delete', 'delete'])
    })
  })

  // ── Perfect match ───────────────────────────────────────────────────────────

  describe('perfect match', () => {
    it('single ASCII char → match', () => {
      const r = diffStrings('a', 'a')
      expect(r.distance).toBe(0)
      expect(r.inputTokens).toEqual([{ char: 'a', op: 'match' }])
    })

    it('identical ASCII word → all match tokens', () => {
      const r = diffStrings('Hund', 'Hund')
      expect(r.distance).toBe(0)
      expect(r.inputTokens).toHaveLength(4)
      expect(ops(r.inputTokens)).toEqual(['match', 'match', 'match', 'match'])
      expect(chars(r.inputTokens)).toEqual(['H', 'u', 'n', 'd'])
    })

    it('identical string with German umlaut → all match (Unicode-aware)', () => {
      const r = diffStrings('Tür', 'Tür')
      expect(r.distance).toBe(0)
      expect(r.inputTokens).toHaveLength(3)
      expect(ops(r.inputTokens)).toEqual(['match', 'match', 'match'])
    })

    it('identical string with ß → all match', () => {
      const r = diffStrings('Straße', 'Straße')
      expect(r.distance).toBe(0)
      expect(ops(r.inputTokens)).toEqual(Array(6).fill('match'))
    })

    it('identical multi-word phrase → all match', () => {
      const r = diffStrings('Guten Tag', 'Guten Tag')
      expect(r.distance).toBe(0)
      expect(r.inputTokens).toHaveLength(9)
      expect(ops(r.inputTokens)).toEqual(Array(9).fill('match'))
    })
  })

  // ── Substitution ────────────────────────────────────────────────────────────

  describe('substitution — wrong character typed', () => {
    it('wrong capitalisation of first letter (case-sensitive)', () => {
      const r = diffStrings('hund', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(4)
      expect(r.inputTokens[0]).toEqual({ char: 'h', op: 'substitute' })
      expect(ops(r.inputTokens).slice(1)).toEqual(['match', 'match', 'match'])
    })

    it('wrong last character', () => {
      const r = diffStrings('Hunn', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(4)
      expect(ops(r.inputTokens).slice(0, 3)).toEqual(['match', 'match', 'match'])
      expect(r.inputTokens[3]).toEqual({ char: 'n', op: 'substitute' })
    })

    it('wrong umlaut — u typed instead of ü', () => {
      const r = diffStrings('Tur', 'Tür')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(3)
      expect(r.inputTokens[0]).toEqual({ char: 'T', op: 'match' })
      expect(r.inputTokens[1]).toEqual({ char: 'u', op: 'substitute' })
      expect(r.inputTokens[2]).toEqual({ char: 'r', op: 'match' })
    })

    it('wrong last char of multi-word phrase', () => {
      const r = diffStrings('Guten Tar', 'Guten Tag')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(9)
      expect(ops(r.inputTokens).slice(0, 8)).toEqual(Array(8).fill('match'))
      expect(r.inputTokens[8]).toEqual({ char: 'r', op: 'substitute' })
    })
  })

  // ── Insertion (user omitted a character) ────────────────────────────────────

  describe('insert — character missing from user input', () => {
    it('missing last character → placeholder · appended', () => {
      const r = diffStrings('Hun', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(4) // 3 user chars + 1 placeholder
      expect(ops(r.inputTokens)).toEqual(['match', 'match', 'match', 'insert'])
      expect(r.inputTokens[3]).toEqual({ char: '·', op: 'insert' })
    })

    it('missing middle character → placeholder · at correct position', () => {
      const r = diffStrings('Hnd', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(4) // 3 user chars + 1 placeholder
      expect(r.inputTokens[0]).toEqual({ char: 'H', op: 'match' })
      expect(r.inputTokens[1]).toEqual({ char: '·', op: 'insert' })
      expect(r.inputTokens[2]).toEqual({ char: 'n', op: 'match' })
      expect(r.inputTokens[3]).toEqual({ char: 'd', op: 'match' })
    })

    it('all chars missing (Inconnu / empty input) → only · placeholders', () => {
      const r = diffStrings('', 'Hund')
      expect(r.distance).toBe(4)
      expect(r.inputTokens).toHaveLength(4)
      expect(chars(r.inputTokens)).toEqual(['·', '·', '·', '·'])
      expect(ops(r.inputTokens)).toEqual(['insert', 'insert', 'insert', 'insert'])
    })

    it('insert placeholder char is always ·', () => {
      const r = diffStrings('', 'xyz')
      expect(chars(r.inputTokens)).toEqual(['·', '·', '·'])
    })
  })

  // ── Deletion (user typed an extra character) ─────────────────────────────────

  describe('delete — extra character typed by user', () => {
    it('extra character at end', () => {
      const r = diffStrings('Hunde', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(5)
      expect(ops(r.inputTokens).slice(0, 4)).toEqual(['match', 'match', 'match', 'match'])
      expect(r.inputTokens[4]).toEqual({ char: 'e', op: 'delete' })
    })

    it('extra character in middle', () => {
      const r = diffStrings('Huznd', 'Hund')
      expect(r.distance).toBe(1)
      expect(r.inputTokens).toHaveLength(5)
      expect(r.inputTokens[0]).toEqual({ char: 'H', op: 'match' })
      expect(r.inputTokens[1]).toEqual({ char: 'u', op: 'match' })
      expect(r.inputTokens[2]).toEqual({ char: 'z', op: 'delete' })
      expect(r.inputTokens[3]).toEqual({ char: 'n', op: 'match' })
      expect(r.inputTokens[4]).toEqual({ char: 'd', op: 'match' })
    })
  })

  // ── Multi-error alignment (distance > 1) ────────────────────────────────────
  // These tests verify that the backtracking remains correct when multiple
  // operations are needed, including combinations of different op types.

  describe('multi-error alignment', () => {
    it('two consecutive missing chars in the middle (distance=2)', () => {
      // 'Hd' vs 'Hund': 'u' and 'n' both omitted, so two insert placeholders
      // Verified DP: (2,4)→match d, (1,3)→insert, (1,2)→insert, (1,1)→match H
      const r = diffStrings('Hd', 'Hund')
      expect(r.distance).toBe(2)
      expect(r.inputTokens).toHaveLength(4) // 2 user chars + 2 placeholders
      expect(r.inputTokens[0]).toEqual({ char: 'H', op: 'match' })
      expect(r.inputTokens[1]).toEqual({ char: '·', op: 'insert' })
      expect(r.inputTokens[2]).toEqual({ char: '·', op: 'insert' })
      expect(r.inputTokens[3]).toEqual({ char: 'd', op: 'match' })
    })

    it('one missing char (insert) + one wrong char (substitute) in same string (distance=2)', () => {
      // 'Hnn' vs 'Hund': 'u' is missing (insert ·), second 'n' substitutes 'd'
      // Verified DP: (3,4)→sub n/d, (2,3)→match n, (1,2)→insert, (1,1)→match H
      const r = diffStrings('Hnn', 'Hund')
      expect(r.distance).toBe(2)
      expect(r.inputTokens).toHaveLength(4) // 3 user chars + 1 placeholder
      expect(r.inputTokens[0]).toEqual({ char: 'H', op: 'match' })
      expect(r.inputTokens[1]).toEqual({ char: '·', op: 'insert' })
      expect(r.inputTokens[2]).toEqual({ char: 'n', op: 'match' })
      expect(r.inputTokens[3]).toEqual({ char: 'n', op: 'substitute' })
    })

    it('three consecutive substitutions (distance=3)', () => {
      // 'abc' vs 'xyz': every character is wrong, all three are substitutions
      // Backtracking is unambiguous: diagonal preferred at every cell
      const r = diffStrings('abc', 'xyz')
      expect(r.distance).toBe(3)
      expect(r.inputTokens).toHaveLength(3)
      expect(ops(r.inputTokens)).toEqual(['substitute', 'substitute', 'substitute'])
      expect(chars(r.inputTokens)).toEqual(['a', 'b', 'c'])
    })
  })

  // ── Distance field ───────────────────────────────────────────────────────────

  describe('distance field', () => {
    it('matches classic Levenshtein distances', () => {
      const cases: [string, string, number][] = [
        ['', '', 0],
        ['a', 'b', 1],
        ['abc', 'abc', 0],
        ['Hund', 'hund', 1],
        ['Guten Tag', 'Guten Tar', 1],
        ['kitten', 'sitting', 3],
        ['', 'Guten Tag', 9],
      ]
      for (const [a, b, expected] of cases) {
        expect(diffStrings(a, b).distance).toBe(expected)
      }
    })

    it('distance is symmetric', () => {
      const pairs: [string, string][] = [
        ['Hund', 'hund'],
        ['Guten Tag', 'Guten Tar'],
        ['Hun', 'Hund'],
      ]
      for (const [a, b] of pairs) {
        expect(diffStrings(a, b).distance).toBe(diffStrings(b, a).distance)
      }
    })
  })

  // ── Token count invariant ─────────────────────────────────────────────────────

  describe('token count invariant', () => {
    it('token count = len(input) + number of insert tokens', () => {
      const cases: [string, string][] = [
        ['', 'ab'],
        ['Hun', 'Hund'],
        ['Hnd', 'Hund'],
        ['Hunde', 'Hund'],
        ['Guten Tag', 'Guten Tag'],
        ['Guten Tar', 'Guten Tag'],
        ['', 'Guten Tag'],
        ['Guten Tag', ''],
      ]
      for (const [a, b] of cases) {
        const r = diffStrings(a, b)
        const insertCount = r.inputTokens.filter((t) => t.op === 'insert').length
        expect(r.inputTokens.length).toBe(Array.from(a).length + insertCount)
      }
    })

    it('delete tokens always carry the actual user character, never ·', () => {
      const r = diffStrings('Hunde', 'Hund')
      const deleted = r.inputTokens.filter((t) => t.op === 'delete')
      for (const t of deleted) {
        expect(t.char).not.toBe('·')
      }
    })

    it('insert tokens always carry the · placeholder, never an actual char', () => {
      const r = diffStrings('Hnd', 'Hund')
      const inserted = r.inputTokens.filter((t) => t.op === 'insert')
      for (const t of inserted) {
        expect(t.char).toBe('·')
      }
    })
  })
})

// ── computeAccuracy ───────────────────────────────────────────────────────────

describe('computeAccuracy', () => {
  it('returns 1.0 for identical strings', () => {
    expect(computeAccuracy('Hund', 'Hund')).toBe(1)
  })

  it('returns 1.0 for two empty strings', () => {
    expect(computeAccuracy('', '')).toBe(1)
  })

  it('returns 0 for empty input against non-empty correct', () => {
    // lev('', 'abc') = 3, max(0,3) = 3, accuracy = 1 - 3/3 = 0
    expect(computeAccuracy('', 'abc')).toBe(0)
  })

  it('uses max(len_a, len_b) as the denominator', () => {
    // lev('ab', 'abc') = 1, max(2,3) = 3, accuracy = 1 - 1/3 ≈ 0.6667
    expect(computeAccuracy('ab', 'abc')).toBeCloseTo(1 - 1 / 3)
  })

  it('returns a value in [0, 1]', () => {
    const cases: [string, string][] = [
      ['', ''],
      ['abc', 'xyz'],
      ['Hund', 'Hund'],
      ['', 'Guten Tag'],
    ]
    for (const [a, b] of cases) {
      const acc = computeAccuracy(a, b)
      expect(acc).toBeGreaterThanOrEqual(0)
      expect(acc).toBeLessThanOrEqual(1)
    }
  })
})

// ── parseAlternatives ─────────────────────────────────────────────────────────

describe('parseAlternatives', () => {
  it('handles single value', () =>
    import('./levenshtein').then(({ parseAlternatives }) =>
      expect(parseAlternatives('[lernen]')).toEqual(['lernen'])))

  it('handles two alternatives', () =>
    import('./levenshtein').then(({ parseAlternatives }) =>
      expect(parseAlternatives('[fahren;losfahren]')).toEqual(['fahren', 'losfahren'])))

  it('handles three alternatives', () =>
    import('./levenshtein').then(({ parseAlternatives }) =>
      expect(parseAlternatives('[a;b;c]')).toEqual(['a', 'b', 'c'])))

  it('handles German words with umlauts', () =>
    import('./levenshtein').then(({ parseAlternatives }) =>
      expect(parseAlternatives('[können;wissen]')).toEqual(['können', 'wissen'])))
})

// ── bestDiff ─────────────────────────────────────────────────────────────────

describe('bestDiff', () => {
  it('exact match on first alternative gives distance 0', () =>
    import('./levenshtein').then(({ bestDiff }) =>
      expect(bestDiff('fahren', '[fahren;losfahren]').diff.distance).toBe(0)))

  it('exact match on second alternative gives distance 0', () =>
    import('./levenshtein').then(({ bestDiff }) =>
      expect(bestDiff('losfahren', '[fahren;losfahren]').diff.distance).toBe(0)))

  it('picks the alternative with lowest distance', () =>
    import('./levenshtein').then(({ bestDiff }) => {
      // 'fahre' is closer to 'fahren' (distance 1) than to 'losfahren' (distance 4)
      const { diff, matched } = bestDiff('fahre', '[fahren;losfahren]')
      expect(matched).toBe('fahren')
      expect(diff.distance).toBe(1)
    }))

  it('returns the matched alternative string', () =>
    import('./levenshtein').then(({ bestDiff }) => {
      const { matched } = bestDiff('losfahren', '[fahren;losfahren]')
      expect(matched).toBe('losfahren')
    }))

  it('single alternative behaves like plain diffStrings', () =>
    import('./levenshtein').then(({ bestDiff }) =>
      expect(bestDiff('ohne', '[ohne]').diff.distance).toBe(0)))

  it('distance is 0 when input matches any alternative exactly', () =>
    import('./levenshtein').then(({ bestDiff }) => {
      const { diff } = bestDiff('weggehen', '[fahren;losfahren;weggehen]')
      expect(diff.distance).toBe(0)
    }))
})

// ── accuracyToRating ──────────────────────────────────────────────────────────

describe('accuracyToRating', () => {
  it('returns 1 (Again) for accuracy in [0, 0.75)', () => {
    expect(accuracyToRating(0)).toBe(1)
    expect(accuracyToRating(0.5)).toBe(1)
    expect(accuracyToRating(0.7499)).toBe(1)
  })

  it('returns 2 (Hard) for accuracy in [0.75, 0.9)', () => {
    expect(accuracyToRating(0.75)).toBe(2)
    expect(accuracyToRating(0.8)).toBe(2)
    expect(accuracyToRating(0.8999)).toBe(2)
  })

  it('returns 3 (Good) for accuracy in [0.9, 1.0)', () => {
    expect(accuracyToRating(0.9)).toBe(3)
    expect(accuracyToRating(0.95)).toBe(3)
    expect(accuracyToRating(0.9999)).toBe(3)
  })

  it('returns 4 (Easy) for perfect accuracy (1.0)', () => {
    expect(accuracyToRating(1)).toBe(4)
    expect(accuracyToRating(1.0)).toBe(4)
  })
})
