-- Extend messages for reply-chain and bot metadata
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_message_id BIGINT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender_id UUID NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender_name TEXT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content_preview TEXT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_bot_message BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE messages SET is_bot_message = TRUE WHERE sender_type = 'bot' AND is_bot_message = FALSE;

CREATE INDEX IF NOT EXISTS idx_messages_reply_to_message_id ON messages(reply_to_message_id);
