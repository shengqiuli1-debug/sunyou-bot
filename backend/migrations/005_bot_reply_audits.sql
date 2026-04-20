CREATE TABLE IF NOT EXISTS bot_reply_audits (
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
);

CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_room_created ON bot_reply_audits(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_trigger_message_id ON bot_reply_audits(trigger_message_id);
CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_reply_message_id ON bot_reply_audits(reply_message_id);
CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_reply_source ON bot_reply_audits(reply_source);
CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_bot_role ON bot_reply_audits(bot_role);
