CREATE TABLE IF NOT EXISTS users(
    id char(36) PRIMARY KEY,
    username varchar(40) UNIQUE NOT NULL,
    password varchar(60) NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL
)