-- +goose Up
-- +goose StatementBegin

CREATE TABLE writing_cards (
    user_id     UUID             NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    word_id     UUID             NOT NULL REFERENCES words(id)  ON DELETE CASCADE,
    stability   DOUBLE PRECISION NOT NULL DEFAULT 0,
    difficulty  DOUBLE PRECISION NOT NULL DEFAULT 0,
    state       SMALLINT         NOT NULL DEFAULT 1,  -- 1=Learning 2=Review 3=Relearning
    step        INTEGER          NOT NULL DEFAULT 0,
    due         TIMESTAMPTZ      NOT NULL DEFAULT now(),
    last_review TIMESTAMPTZ,                           -- NULL until first review
    reps        INTEGER          NOT NULL DEFAULT 0,
    lapses      INTEGER          NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, word_id)
);

-- Drives GetNextDueWritingCard: filter by user, order by due ASC
CREATE INDEX idx_writing_cards_user_due ON writing_cards (user_id, due);

-- Backfill: every existing (user, word) pair in cards gets a fresh writing_cards row.
-- New users will receive rows via UpdateAssimilNumberTx going forward.
INSERT INTO writing_cards (user_id, word_id, due)
SELECT user_id, word_id, now()
FROM cards
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_writing_cards_user_due;
DROP TABLE IF EXISTS writing_cards;

-- +goose StatementEnd
