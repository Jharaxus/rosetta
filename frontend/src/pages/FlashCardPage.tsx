import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchFlashCard, submitReview } from '../api/words'
import { ApiError } from '../api/auth'
import { RATING_LABELS } from '../types/words'
import type { Rating } from '../types/words'
import styles from './FlashCardPage.module.css'

const RATINGS: Rating[] = [1, 2, 3, 4]

export function FlashCardPage() {
  const [flipped, setFlipped] = useState(false)
  const [skipTransition, setSkipTransition] = useState(false)
  const queryClient = useQueryClient()

  const { data: card, isLoading, error } = useQuery({
    queryKey: ['flashcard'],
    queryFn: fetchFlashCard,
    staleTime: 0,
    retry: false,
  })

  const reviewMutation = useMutation({
    mutationFn: ({ wordId, rating }: { wordId: string; rating: Rating }) =>
      submitReview(wordId, rating),
    onSuccess: () => {
      setSkipTransition(true)
      setFlipped(false)
      queryClient.invalidateQueries({ queryKey: ['flashcard'] })
    },
  })

  const noCardsDue = error instanceof ApiError && error.code === 'no_cards_due'
  const noCardsAvailable = error instanceof ApiError && error.code === 'no_cards_available'

  function handleRate(rating: Rating) {
    if (!card || reviewMutation.isPending) return
    reviewMutation.mutate({ wordId: card.id, rating })
  }

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
        <div className={styles.headerLabel}>CARTES MÉMOIRE</div>
        <h1 className={styles.headerTitle}>
          Révisez votre<br />
          <em>vocabulaire</em>.
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

      {card && (
        <div
          className={styles.cardScene}
          onClick={() => { if (!flipped) { setSkipTransition(false); setFlipped(true) } }}
          role="button"
          aria-label={flipped ? undefined : 'Révéler la traduction'}
          style={{ cursor: flipped ? 'default' : 'pointer' }}
        >
          <div className={`${styles.cardInner} ${flipped ? styles.flipped : ''} ${skipTransition ? styles.noTransition : ''}`}>
            {/* Front: French */}
            <div className={`${styles.cardFace} ${styles.cardFront}`}>
              <div className={styles.cardLabel}>
                CARTE MÉMOIRE · LEÇON {card.assimil_number}
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
              <p className={styles.flipHint}>cliquez pour révéler</p>
            </div>

            {/* Back: German + French small */}
            <div className={`${styles.cardFace} ${styles.cardBack}`}>
              <div className={styles.cardLabel}>
                TRADUCTION · LEÇON {card.assimil_number}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.german}>{card.german}</p>
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
            </div>
          </div>
        </div>
      )}

      {card && flipped && (
        <div className={styles.ratingRow} role="group" aria-label="Évaluer votre souvenir">
          {RATINGS.map((r) => (
            <button
              key={r}
              type="button"
              className={`${styles.ratingBtn} ${styles[`rating${r}`]}`}
              onClick={() => handleRate(r)}
              disabled={reviewMutation.isPending}
            >
              {RATING_LABELS[r]}
            </button>
          ))}
        </div>
      )}

      {card && !flipped && (
        <p className={styles.ratingHint}>
          Retournez la carte, puis évaluez votre souvenir.
        </p>
      )}
    </main>
  )
}
