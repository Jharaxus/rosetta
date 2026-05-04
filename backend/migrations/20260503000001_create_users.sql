-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  subject      TEXT        NOT NULL UNIQUE,
  email        TEXT        NOT NULL,
  display_name TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS users;
