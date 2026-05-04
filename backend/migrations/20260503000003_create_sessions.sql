-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
  token  TEXT        PRIMARY KEY,
  data   BYTEA       NOT NULL,
  expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_expiry ON sessions (expiry);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS sessions;
