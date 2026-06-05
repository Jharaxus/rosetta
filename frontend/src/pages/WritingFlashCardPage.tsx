import { useRef, useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWritingFlashCard, submitWritingReview } from '../api/words'
import { synthesizeSpeech } from '../api/tts'
import { ApiError } from '../api/auth'
import { RATING_LABELS } from '../types/words'
import type { Rating } from '../types/words'
import { diffStrings, accuracyToRating } from '../utils/levenshtein'
import type { StringDiff } from '../utils/levenshtein'
import { useAudio } from '../hooks/useAudio'
import styles from './WritingFlashCardPage.module.css'

type Phase = 'input' | 'result'

interface Result {
  accuracy: number
  rating: Rating
  diff: StringDiff
}

export function WritingFlashCardPage() {
  const [phase, setPhase] = useState<Phase>('input')
  const [input, setInput] = useState('')
  const [flipped, setFlipped] = useState(false)
  const [skipTransition, setSkipTransition] = useState(false)
  const [result, setResult] = useState<Result | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const { data: card, isLoading, error } = useQuery({
    queryKey: ['writing-flashcard'],
    queryFn: fetchWritingFlashCard,
    staleTime: 0,
    retry: false,
  })

  const audio = useAudio()

  const reviewMutation = useMutation({
    mutationFn: ({ wordId, rating }: { wordId: string; rating: Rating }) =>
      submitWritingReview(wordId, rating),
  })

  const synthesizeMutation = useMutation({
    mutationFn: () => synthesizeSpeech(card!.german),
    onSuccess: (blob) => audio.playBlob(blob),
  })

  // Preload static audio when card changes
  useEffect(() => {
    if (card?.audio_url) audio.preload(card.audio_url)
  }, [card?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const noCardsDue = error instanceof ApiError && error.code === 'no_cards_due'
  const noCardsAvailable = error instanceof ApiError && error.code === 'no_cards_available'

  function handleValidate() {
    if (!card || phase !== 'input') return
    const diff = diffStrings(input, card.german)
    const maxLen = Math.max(Array.from(input).length, Array.from(card.german).length)
    const accuracy = maxLen === 0 ? 1 : 1 - diff.distance / maxLen
    const rating = accuracyToRating(accuracy)
    setResult({ accuracy, rating, diff })
    setSkipTransition(false)
    setFlipped(true)
    setPhase('result')
    reviewMutation.mutate({ wordId: card.id, rating })
    inputRef.current?.blur()
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') handleValidate()
  }

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Enter' || phase !== 'result') return
      if (e.target instanceof HTMLInputElement) return
      e.preventDefault()
      handleNext()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [phase])

  function handleNext() {
    setSkipTransition(true)
    setFlipped(false)
    setPhase('input')
    setInput('')
    setResult(null)
    queryClient.invalidateQueries({ queryKey: ['writing-flashcard'] })
    // focus the input once the new card loads
    setTimeout(() => inputRef.current?.focus(), 50)
  }

  const pct = result ? Math.round(result.accuracy * 100) : 0

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
        <div className={styles.headerLabel}>ÉCRITURE</div>
        <h1 className={styles.headerTitle}>
          Écrivez la<br />
          <em>traduction</em>.
        </h1>
      </div>

      {isLoading && (
        <div className={styles.staticCard}>
          <div className={styles.skeleton} />
          <div className={styles.skeletonSm} />
        </div>
      )}

      {(noCardsDue || noCardsAvailable) && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>Tout est à jour.</p>
          <p className={styles.emptyHint}>
            {noCardsAvailable
              ? 'Avancez dans vos leçons Assimil pour débloquer des mots.'
              : 'Revenez plus tard pour la prochaine révision.'}
          </p>
        </div>
      )}

      {!isLoading && !noCardsDue && !noCardsAvailable && error && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>Une erreur est survenue.</p>
        </div>
      )}

      {card && !error && (
        <div className={styles.cardScene}>
          <div
            className={`${styles.cardInner} ${flipped ? styles.flipped : ''} ${skipTransition ? styles.noTransition : ''}`}
          >
            {/* Front: French word + text input */}
            <div className={`${styles.cardFace} ${styles.cardFront}`}>
              <div className={styles.cardLabel}>
                ÉCRITURE · LEÇON {card.assimil_number}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.french}>{card.french}</p>
              </div>
              <div className={styles.tags}>
                <span className={styles.tag}>{card.category}</span>
                {card.is_regular !== null && (
                  <span className={styles.tag}>
                    {card.is_regular ? 'régulier' : 'irrégulier'}
                  </span>
                )}
              </div>
              <div className={styles.inputRow}>
                <input
                  ref={inputRef}
                  type="text"
                  className={styles.answerInput}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="traduction allemande…"
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

            {/* Back: correct answer + accuracy */}
            <div className={`${styles.cardFace} ${styles.cardBack}`}>
              <div className={styles.cardLabel}>
                TRADUCTION · LEÇON {card.assimil_number}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.german}>{card.german}</p>
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
                <p className={styles.frenchSm}>{card.french}</p>
              </div>
              <div className={styles.tags}>
                <span className={styles.tag}>{card.category}</span>
                {card.is_regular !== null && (
                  <span className={styles.tag}>
                    {card.is_regular ? 'régulier' : 'irrégulier'}
                  </span>
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
              {phase === 'result' && (
                <div className={styles.audioRow}>
                  {card.audio_url && (
                    <button
                      type="button"
                      className={`${styles.audioBtn} ${audio.isError ? styles.audioBtnError : ''}`}
                      onClick={() => audio.playUrl(card.audio_url)}
                      disabled={audio.isPlaying}
                    >
                      {audio.isError ? 'ERREUR' : 'ÉCOUTER'}
                    </button>
                  )}
                  <button
                    type="button"
                    className={`${styles.audioBtn} ${synthesizeMutation.isError ? styles.audioBtnError : ''}`}
                    onClick={() => synthesizeMutation.mutate()}
                    disabled={synthesizeMutation.isPending || audio.isPlaying}
                  >
                    {synthesizeMutation.isPending ? '…' : synthesizeMutation.isError ? 'INDISPONIBLE' : 'SYNTHÈSE'}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {card && !error && phase === 'result' && (
        <button
          type="button"
          className={styles.nextBtn}
          onClick={handleNext}
        >
          Carte suivante <span className={styles.arrow}>→</span>
        </button>
      )}

      {card && !error && phase === 'input' && (
        <p className={styles.hint}>
          Saisissez la traduction allemande, puis validez.
        </p>
      )}
    </main>
  )
}
