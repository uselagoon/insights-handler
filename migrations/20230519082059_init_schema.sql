-- +goose Up
-- +goose StatementBegin
CREATE TABLE lagoons (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cluster VARCHAR(255)
);

CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    lagoon_id INT NOT NULL,
    FOREIGN KEY (lagoon_id) REFERENCES lagoons(id)
);

CREATE TABLE environments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE projects;
DROP TABLE environments;
DROP TABLE lagoons;
-- +goose StatementEnd