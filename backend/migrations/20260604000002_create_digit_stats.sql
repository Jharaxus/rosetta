-- +goose Up
CREATE TABLE user_digit_stats (
    user_id   UUID     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    digit     SMALLINT NOT NULL CHECK (digit BETWEEN 0 AND 9),
    successes INTEGER  NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, digit)
);

-- +goose Down
DROP TABLE user_digit_stats;
