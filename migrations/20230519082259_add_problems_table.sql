-- +goose Up
-- +goose StatementBegin
CREATE TABLE problems (
    id          SERIAL PRIMARY KEY,
    identifier  VARCHAR(255) NOT NULL,
    value       VARCHAR(255) NOT NULL,
    environment VARCHAR(255) NOT NULL,
    source      VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category    VARCHAR(255) NOT NULL,
    key_fact    BOOLEAN NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE problems;
-- +goose StatementEnd
