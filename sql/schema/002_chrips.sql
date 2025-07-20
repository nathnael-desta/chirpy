-- +goose Up
CREATE TABLE chirps (
    id UUID Primary key,
    created_at TIMESTAMP NOT NULL, 
    updated_at TIMESTAMP NOT NULL, 
    body TEXT NOT NULL,
    user_id UUID

);

-- +goose Down
DROP TABLE chirps; 