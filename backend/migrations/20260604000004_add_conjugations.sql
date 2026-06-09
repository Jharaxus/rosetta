-- +goose Up
-- +goose StatementBegin
CREATE TYPE verb_tense AS ENUM (
    'praesens_indikativ',
    'praesens_konjunktiv_1',
    'praeteritum_indikativ',
    'praeteritum_konjunktiv_2',
    'perfekt_indikativ',
    'perfekt_konjunktiv_1',
    'plusquamperfekt_indikativ',
    'plusquamperfekt_konjunktiv_2',
    'futur_1_indikativ',
    'futur_1_konjunktiv_1',
    'futur_1_konjunktiv_2',
    'futur_2_indikativ',
    'futur_2_konjunktiv_1',
    'futur_2_konjunktiv_2',
    'imperativ'
);

CREATE TYPE verb_person AS ENUM (
    'p1_sg',  -- ich
    'p2_sg',  -- du
    'p3_sg',  -- er/sie/es
    'p1_pl',  -- wir
    'p2_pl',  -- ihr
    'p3_pl'   -- sie
);

CREATE TABLE conjugations (
    word_id UUID        NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    tense   verb_tense  NOT NULL,
    person  verb_person NOT NULL,
    forms   TEXT        NOT NULL,
    PRIMARY KEY (word_id, tense, person)
);

-- Enforce that word_id must reference a word with category = 'Verb'.
-- A plain FK cannot filter by column value, so a BEFORE trigger is used.
CREATE OR REPLACE FUNCTION enforce_conjugation_word_is_verb()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM words WHERE id = NEW.word_id AND category = 'Verb'
    ) THEN
        RAISE EXCEPTION
            'conjugations.word_id % must reference a Verb, got a non-Verb word',
            NEW.word_id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER conjugations_word_must_be_verb
BEFORE INSERT OR UPDATE OF word_id ON conjugations
FOR EACH ROW EXECUTE FUNCTION enforce_conjugation_word_is_verb();

-- Supports fast lookup of all conjugations for a given word
CREATE INDEX idx_conjugations_word_id ON conjugations (word_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER  IF EXISTS conjugations_word_must_be_verb ON conjugations;
DROP FUNCTION IF EXISTS enforce_conjugation_word_is_verb();
DROP TABLE    IF EXISTS conjugations;
DROP TYPE     IF EXISTS verb_person;
DROP TYPE     IF EXISTS verb_tense;
-- +goose StatementEnd
