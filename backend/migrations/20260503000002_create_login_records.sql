-- +goose Up
-- +goose StatementBegin
CREATE TABLE login_records (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  logged_in_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ip_address   INET,
  user_agent   TEXT,
  session_id   TEXT
);

CREATE INDEX idx_login_records_user_id ON login_records (user_id, logged_in_at DESC);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS login_records;
