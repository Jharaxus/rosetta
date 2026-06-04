import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toCardinal } from 'n2words/de-DE'
import {
  fetchPracticeNumber,
  fetchNextFailure,
  fetchNumberSettings,
  updateNumberSettings,
  submitDigitSuccess,
  addNumberFailure,
  removeNumberFailure,
} from '../api/numbers'
import { RATING_LABELS } from '../types/words'
import type { Rating } from '../types/words'
import { diffStrings, accuracyToRating } from '../utils/levenshtein'
import type { StringDiff } from '../utils/levenshtein'
import styles from './NumberPracticePage.module.css'

type Mode = 'practice' | 'review'
type Phase = 'input' | 'result'

interface Result {
  accuracy: number
  rating: Rating
  diff: StringDiff
  expected: string
}

export function NumberPracticePage() {
  const [mode, setMode] = useState<Mode>('practice')
  const [digitSize, setDigitSize] = useState(1)
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [phase, setPhase] = useState<Phase>('input')
  const [input, setInput] = useState('')
  const [flipped, setFlipped] = useState(false)
  const [skipTransition, setSkipTransition] = useState(false)
  const [result, setResult] = useState<Result | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const settingsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const queryClient = useQueryClient()

  const { data: settings } = useQuery({
    queryKey: ['number-settings'],
    queryFn: fetchNumberSettings,
    staleTime: Infinity,
    retry: false,
  })

  useEffect(() => {
    if (settings && !settingsLoaded) {
      setDigitSize(settings.number_digit_size)
      setSettingsLoaded(true)
    }
  }, [settings, settingsLoaded])

  const practiceQuery = useQuery({
    queryKey: ['numbers', 'practice', digitSize],
    queryFn: () => fetchPracticeNumber(digitSize),
    enabled: mode === 'practice',
    staleTime: 0,
    retry: false,
  })

  const failureQuery = useQuery({
    queryKey: ['numbers', 'failures', 'next'],
    queryFn: fetchNextFailure,
    enabled: mode === 'review',
    staleTime: 0,
    retry: false,
  })

  const isLoading = mode === 'practice' ? practiceQuery.isLoading : failureQuery.isLoading
  const currentNumber: number | null =
    mode === 'practice'
      ? (practiceQuery.data != null ? practiceQuery.data : null)
      : (failureQuery.data != null ? failureQuery.data : null)
  const noFailures =
    mode === 'review' && failureQuery.isFetched && failureQuery.data === null

  function resetCard(noAnimation = true) {
    setSkipTransition(noAnimation)
    setFlipped(false)
    setPhase('input')
    setInput('')
    setResult(null)
  }

  function handleModeChange(newMode: Mode) {
    if (newMode === mode) return
    setMode(newMode)
    resetCard()
  }

  function handleDigitSizeChange(newSize: number) {
    setDigitSize(newSize)
    resetCard()
    if (settingsTimerRef.current) clearTimeout(settingsTimerRef.current)
    settingsTimerRef.current = setTimeout(() => {
      updateNumberSettings(newSize).catch(() => {})
    }, 500)
  }

  function handleValidate() {
    if (currentNumber === null || phase !== 'input') return
    const expected = toCardinal(currentNumber)
    const diff = diffStrings(input, expected)
    const maxLen = Math.max(Array.from(input).length, Array.from(expected).length)
    const accuracy = maxLen === 0 ? 1 : 1 - diff.distance / maxLen
    const rating = accuracyToRating(accuracy)
    setResult({ accuracy, rating, diff, expected })
    setSkipTransition(false)
    setFlipped(true)
    setPhase('result')

    if (mode === 'practice') {
      if (accuracy === 1) submitDigitSuccess(currentNumber).catch(() => {})
      else addNumberFailure(currentNumber).catch(() => {})
    } else if (accuracy === 1) {
      removeNumberFailure(currentNumber).catch(() => {})
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') handleValidate()
  }

  function handleNext() {
    resetCard()
    const queryKey =
      mode === 'practice'
        ? ['numbers', 'practice', digitSize]
        : ['numbers', 'failures', 'next']
    queryClient.invalidateQueries({ queryKey })
    setTimeout(() => inputRef.current?.focus(), 50)
  }

  const pct = result ? Math.round(result.accuracy * 100) : 0
  const digitLabel = digitSize === 1 ? '1 chiffre' : `${digitSize} chiffres`

  return (
    <main className={styles.page}>
      <div className={styles.topBar}>
        <div className={styles.logoMark}>
          <div className={styles.logoDot} />
          <span className={styles.logoText}>Rosetta</span>
        </div>
        <Link to="/dashboard" className={styles.backLink}>
          ← tableau de bord
        </Link>
      </div>

      <div className={styles.header}>
        <div className={styles.headerLabel}>NOMBRES</div>
        <h1 className={styles.headerTitle}>
          Écrivez le<br />
          <em>nombre</em>.
        </h1>
      </div>

      <div className={styles.modeTabs}>
        <button
          type="button"
          className={`${styles.modeTab} ${mode === 'practice' ? styles.modeTabActive : ''}`}
          onClick={() => handleModeChange('practice')}
        >
          Pratique
        </button>
        <button
          type="button"
          className={`${styles.modeTab} ${mode === 'review' ? styles.modeTabActive : ''}`}
          onClick={() => handleModeChange('review')}
        >
          Révision
        </button>
      </div>

      {mode === 'practice' && (
        <div className={styles.digitControls}>
          <div className={styles.digitLabelRow}>
            <span className={styles.digitLabel}>TAILLE</span>
            <span className={styles.digitValue}>{digitLabel}</span>
          </div>
          <div className={styles.digitRow}>
            <button
              type="button"
              className={styles.btnStepper}
              onClick={() => handleDigitSizeChange(Math.max(1, digitSize - 1))}
              disabled={digitSize <= 1}
              aria-label="Moins de chiffres"
            >
              −
            </button>
            <input
              type="range"
              min={1}
              max={10}
              value={digitSize}
              onChange={(e) => handleDigitSizeChange(Number(e.target.value))}
              className={styles.digitSlider}
              aria-label="Nombre de chiffres"
            />
            <button
              type="button"
              className={styles.btnStepper}
              onClick={() => handleDigitSizeChange(Math.min(10, digitSize + 1))}
              disabled={digitSize >= 10}
              aria-label="Plus de chiffres"
            >
              +
            </button>
          </div>
        </div>
      )}

      {isLoading && (
        <div className={styles.staticCard}>
          <div className={styles.skeleton} />
          <div className={styles.skeletonSm} />
        </div>
      )}

      {!isLoading && noFailures && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>Aucun nombre échoué.</p>
          <p className={styles.emptyHint}>
            Entraînez-vous en mode Pratique pour construire votre liste de révision.
          </p>
        </div>
      )}

      {!isLoading && !noFailures && (mode === 'practice' ? practiceQuery.error : failureQuery.error) && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>Une erreur est survenue.</p>
        </div>
      )}

      {currentNumber !== null && (
        <div className={styles.cardScene}>
          <div
            className={`${styles.cardInner} ${flipped ? styles.flipped : ''} ${skipTransition ? styles.noTransition : ''}`}
          >
            {/* Front: number display + text input */}
            <div className={`${styles.cardFace} ${styles.cardFront}`}>
              <div className={styles.cardLabel}>
                {mode === 'practice' ? `PRATIQUE · ${digitLabel.toUpperCase()}` : 'RÉVISION · NOMBRES ÉCHOUÉS'}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.numberDisplay}>{currentNumber}</p>
                <p className={styles.numberHint}>en allemand</p>
              </div>
              <div className={styles.inputRow}>
                <input
                  ref={inputRef}
                  type="text"
                  className={styles.answerInput}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="nombre en allemand…"
                  autoFocus
                  autoComplete="off"
                  autoCorrect="off"
                  autoCapitalize="off"
                  spellCheck={false}
                />
                <button
                  type="button"
                  className={`${styles.validateBtn} ${input.length > 0 ? styles.validateBtnActive : ''}`}
                  onClick={handleValidate}
                >
                  {input.length === 0 ? 'Inconnu' : 'Valider'}
                </button>
              </div>
            </div>

            {/* Back: expected word + colored user spelling + accuracy */}
            <div className={`${styles.cardFace} ${styles.cardBack}`}>
              <div className={styles.cardLabel}>
                {mode === 'practice' ? `PRATIQUE · ${digitLabel.toUpperCase()}` : 'RÉVISION · NOMBRES ÉCHOUÉS'}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.numberDisplayBack}>{currentNumber}</p>
                {result && (
                  <p className={styles.german}>{result.expected}</p>
                )}
                {result && (
                  <p className={styles.userSpelling} aria-label="Votre saisie">
                    {result.diff.inputTokens.map((token, idx) => (
                      <span
                        key={idx}
                        className={
                          token.op === 'match'
                            ? styles.charMatch
                            : token.op === 'insert'
                              ? styles.charMissing
                              : styles.charWrong
                        }
                      >
                        {token.char}
                      </span>
                    ))}
                  </p>
                )}
              </div>
              {result && (
                <div
                  className={styles.accuracyBadge}
                  data-rating={result.rating}
                >
                  Précision : {pct} % — {RATING_LABELS[result.rating]}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {currentNumber !== null && phase === 'result' && (
        <button
          type="button"
          className={styles.nextBtn}
          onClick={handleNext}
        >
          Nombre suivant <span className={styles.arrow}>→</span>
        </button>
      )}

      {currentNumber !== null && phase === 'input' && (
        <p className={styles.hint}>
          Saisissez le nombre en allemand, puis validez.
        </p>
      )}
    </main>
  )
}
