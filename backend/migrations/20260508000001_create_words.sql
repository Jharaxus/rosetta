-- +goose Up
-- +goose StatementBegin
CREATE TYPE word_category AS ENUM (
    'Noun',
    'Verb',
    'Adjective/Adverb',
    'Personal pronoun',
    'Possessive adjective',
    'Article',
    'Preposition',
    'Conjunction',
    'Interrogative',
    'Number/Numeral',
    'Modal particle',
    'Fixed expression'
);

CREATE TABLE words (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    french         TEXT          NOT NULL,
    german         TEXT          NOT NULL,
    assimil_number INTEGER       NOT NULL CHECK (assimil_number BETWEEN 1 AND 100),
    category       word_category NOT NULL,
    is_regular     BOOLEAN,
    annotation     TEXT,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT words_unique UNIQUE (french, german, assimil_number),
    -- Only verbs carry a regularity value; all other categories must leave it NULL.
    CONSTRAINT words_regularity_check CHECK (
        (category = 'Verb' AND is_regular IS NOT NULL)
        OR (category <> 'Verb' AND is_regular IS NULL)
    )
);

CREATE INDEX idx_words_category       ON words (category);
CREATE INDEX idx_words_assimil_number ON words (assimil_number);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS words;
DROP TYPE IF EXISTS word_category;
-- +goose StatementEnd
