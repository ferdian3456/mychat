CREATE TABLE IF NOT EXISTS messages (
    id char(36) PRIMARY KEY,
    conversation_id INTEGER NOT NULL,
    sender_id VARCHAR(36) NOT NULL,
    text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);
