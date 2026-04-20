package database

import (
	"context"
	"database/sql"
)

func EnsureSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_message_id BIGINT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender_id UUID NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender_name TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content_preview TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_bot_message BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS bot_role TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_source TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS llm_model TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS fallback_reason TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS trace_id TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS force_reply BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS trigger_reason TEXT NULL`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS hype_score INT NULL`,
		`UPDATE messages SET is_bot_message = TRUE WHERE sender_type = 'bot' AND is_bot_message = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_messages_reply_to_message_id ON messages(reply_to_message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_trace_id ON messages(trace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_bot_role ON messages(bot_role)`,
		`CREATE TABLE IF NOT EXISTS bot_reply_audits (
		  id BIGSERIAL PRIMARY KEY,
		  trace_id TEXT NOT NULL UNIQUE,
		  room_id UUID NOT NULL REFERENCES rooms(id),
		  trigger_message_id BIGINT NOT NULL REFERENCES messages(id),
		  trigger_sender_id UUID NULL REFERENCES users(id),
		  trigger_sender_name TEXT NOT NULL DEFAULT '',
		  reply_message_id BIGINT NULL REFERENCES messages(id),
		  reply_source TEXT NOT NULL DEFAULT 'template',
		  bot_role TEXT NOT NULL,
		  firepower_level TEXT NOT NULL,
		  trigger_reason TEXT NULL,
		  force_reply BOOLEAN NOT NULL DEFAULT FALSE,
		  hype_score INT NOT NULL DEFAULT 0,
		  provider TEXT NULL,
		  model TEXT NULL,
		  request_prompt_excerpt TEXT NULL,
		  response_text TEXT NULL,
		  fallback_reason TEXT NULL,
		  http_status INT NULL,
		  latency_ms INT NULL,
		  prompt_tokens INT NULL,
		  completion_tokens INT NULL,
		  total_tokens INT NULL,
		  error_message TEXT NULL,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_room_created ON bot_reply_audits(room_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_trigger_message_id ON bot_reply_audits(trigger_message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_reply_message_id ON bot_reply_audits(reply_message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_reply_source ON bot_reply_audits(reply_source)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_bot_role ON bot_reply_audits(bot_role)`,
		`ALTER TABLE bot_reply_audits ALTER COLUMN trigger_message_id DROP NOT NULL`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS trigger_type TEXT NOT NULL DEFAULT 'user_message'`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS llm_enabled BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS provider_initialized BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS api_key_present BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS absurdity_score INT NOT NULL DEFAULT 0`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS risk_score INT NOT NULL DEFAULT 0`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS reply_mode TEXT NULL`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS displayable_content_found BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE bot_reply_audits ADD COLUMN IF NOT EXISTS reasoning_only_response BOOLEAN NOT NULL DEFAULT FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_trigger_type ON bot_reply_audits(trigger_type)`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
