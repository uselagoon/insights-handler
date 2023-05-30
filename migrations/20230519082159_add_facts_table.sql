-- +goose Up
-- +goose StatementBegin
CREATE TABLE facts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value VARCHAR(255) NOT NULL,
    environment INT NOT NULL,
    source VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(255),
    key_fact BOOLEAN,
    type VARCHAR(255)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE facts;
-- +goose StatementEnd