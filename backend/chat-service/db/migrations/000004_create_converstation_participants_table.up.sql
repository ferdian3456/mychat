CREATE TABLE IF NOT EXISTS conversation_participants (
    conversation_id INTEGER NOT NULL,
    user_id char(36) NOT NULL,
    last_read_message_id INTEGER, -- optional for tracking read status
    PRIMARY KEY (conversation_id, user_id)
);
