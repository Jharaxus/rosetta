import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchFlashCard } from '../api/words'
import { ApiError } from '../api/auth'
import styles from './FlashCardPage.module.css'

export function FlashCardPage() {
  const [seed, setSeed] = useState(0)
  const [flipped, setFlipped] = useState(false)

  const { data: card, isLoading, error } = useQuery({
    queryKey: ['flashcard', seed],
    queryFn: fetchFlashCard,
    staleTime: 0,
    retry: false,
  })

  const noCards = error instanceof ApiError && error.code === 'no_cards_available'

  function handleNext() {
    setFlipped(false)
    setSeed((s) => s + 1)
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

      {noCards && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>
            Aucune carte disponible pour votre niveau actuel.
          </p>
          <p className={styles.emptyHint}>
            Avancez dans vos leçons Assimil pour débloquer des mots.
          </p>
        </div>
      )}

      {!isLoading && !noCards && error && (
        <div className={styles.staticCard}>
          <p className={styles.emptyText}>Une erreur est survenue.</p>
        </div>
      )}

      {card && (
        <div
          className={styles.cardScene}
          onClick={() => setFlipped((f) => !f)}
          role="button"
          aria-label={flipped ? 'Masquer la traduction' : 'Révéler la traduction'}
        >
          <div className={`${styles.cardInner} ${flipped ? styles.flipped : ''}`}>
            {/* Front: German */}
            <div className={`${styles.cardFace} ${styles.cardFront}`}>
              <div className={styles.cardLabel}>
                CARTE MÉMOIRE · LEÇON {card.assimil_number}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.german}>{card.german}</p>
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

            {/* Back: French + full details */}
            <div className={`${styles.cardFace} ${styles.cardBack}`}>
              <div className={styles.cardLabel}>
                TRADUCTION · LEÇON {card.assimil_number}
              </div>
              <div className={styles.wordBlock}>
                <p className={styles.french}>{card.french}</p>
                <p className={styles.germanSm}>{card.german}</p>
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

      <button
        type="button"
        className={styles.nextBtn}
        onClick={handleNext}
        disabled={isLoading || noCards}
      >
        Suivant <span className={styles.arrow}>→</span>
      </button>
    </main>
  )
}
