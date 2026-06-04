-- +goose Up
CREATE TABLE number_failures (
    user_id    UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number     INTEGER NOT NULL CHECK (number >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, number)
);

-- +goose Down
DROP TABLE number_failures;
