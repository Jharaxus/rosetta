-- +goose Up
-- +goose StatementBegin

-- 0 = no lessons unlocked yet (new default semantic)
ALTER TABLE users
    ALTER COLUMN assimil_number SET DEFAULT 0;
ALTER TABLE users
    DROP CONSTRAINT users_assimil_number_check;
ALTER TABLE users
    ADD CONSTRAINT users_assimil_number_check
        CHECK (assimil_number BETWEEN 0 AND 100);

-- FSRS scheduling state per (user, word). Rows are inserted lazily when
-- assimil_number is incremented — absence of a row means "not yet unlocked".
CREATE TABLE cards (
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

-- Drives GetNextDueCard: filter by user, order by due ASC
CREATE INDEX idx_cards_user_due ON cards (user_id, due);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_cards_user_due;
DROP TABLE IF EXISTS cards;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_assimil_number_check;
ALTER TABLE users ADD CONSTRAINT users_assimil_number_check
    CHECK (assimil_number BETWEEN 1 AND 100);
ALTER TABLE users ALTER COLUMN assimil_number SET DEFAULT 1;

-- +goose StatementEnd
