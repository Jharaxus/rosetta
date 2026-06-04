/**
 * Dependency contract tests for n2words/de-DE.
 *
 * The toCardinal function produces the expected answer we compare against user
 * input at runtime. If the library changes its output format after an upgrade
 * (different capitalisation, hyphens, spelling variants, etc.), these tests
 * fail before any user sees silently-incorrect accuracy scores.
 */
import { describe, it, expect } from 'vitest'
import { toCardinal } from 'n2words/de-DE'

describe('toCardinal — German number contract (n2words/de-DE)', () => {
  describe('single digits (0–9)', () => {
    it('0 → null', () => expect(toCardinal(0)).toBe('null'))
    it('1 → eins', () => expect(toCardinal(1)).toBe('eins'))
    it('2 → zwei', () => expect(toCardinal(2)).toBe('zwei'))
    it('5 → fünf', () => expect(toCardinal(5)).toBe('fünf'))
    it('7 → sieben', () => expect(toCardinal(7)).toBe('sieben'))
    it('9 → neun', () => expect(toCardinal(9)).toBe('neun'))
  })

  describe('teens (irregular forms)', () => {
    it('11 → elf', () => expect(toCardinal(11)).toBe('elf'))
    it('12 → zwölf', () => expect(toCardinal(12)).toBe('zwölf'))
    it('13 → dreizehn', () => expect(toCardinal(13)).toBe('dreizehn'))
    it('16 → sechzehn (drops the s)', () => expect(toCardinal(16)).toBe('sechzehn'))
    it('19 → neunzehn', () => expect(toCardinal(19)).toBe('neunzehn'))
  })

  describe('tens (some irregular spelling)', () => {
    it('20 → zwanzig', () => expect(toCardinal(20)).toBe('zwanzig'))
    it('30 → dreißig (ß)', () => expect(toCardinal(30)).toBe('dreißig'))
    it('60 → sechzig (drops the s)', () => expect(toCardinal(60)).toBe('sechzig'))
    it('70 → siebzig (drops the en)', () => expect(toCardinal(70)).toBe('siebzig'))
  })

  describe('compound: units-und-tens', () => {
    it('21 → einundzwanzig', () => expect(toCardinal(21)).toBe('einundzwanzig'))
    it('42 → zweiundvierzig', () => expect(toCardinal(42)).toBe('zweiundvierzig'))
    it('99 → neunundneunzig', () => expect(toCardinal(99)).toBe('neunundneunzig'))
  })

  describe('hundreds', () => {
    it('100 → einhundert', () => expect(toCardinal(100)).toBe('einhundert'))
    it('200 → zweihundert', () => expect(toCardinal(200)).toBe('zweihundert'))
    it('123 → einhundertdreiundzwanzig', () =>
      expect(toCardinal(123)).toBe('einhundertdreiundzwanzig'))
  })

  describe('thousands', () => {
    it('1000 → eintausend', () => expect(toCardinal(1000)).toBe('eintausend'))
    it('2000 → zweitausend', () => expect(toCardinal(2000)).toBe('zweitausend'))
    it('1234 → eintausendzweihundertvierunddreißig', () =>
      expect(toCardinal(1234)).toBe('eintausendzweihundertvierunddreißig'))
  })
})
