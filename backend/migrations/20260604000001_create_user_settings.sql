-- +goose Up
CREATE TABLE user_settings (
    user_id           UUID     PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    number_digit_size SMALLINT NOT NULL DEFAULT 1
        CHECK (number_digit_size BETWEEN 1 AND 10),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_settings;
