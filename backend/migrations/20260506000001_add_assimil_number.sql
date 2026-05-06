-- +goose Up
ALTER TABLE users ADD COLUMN assimil_number INTEGER NOT NULL DEFAULT 1;
ALTER TABLE users ADD CONSTRAINT users_assimil_number_check CHECK (assimil_number BETWEEN 1 AND 100);

-- +goose Down
ALTER TABLE users DROP COLUMN assimil_number;
