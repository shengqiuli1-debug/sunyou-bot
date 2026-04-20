ALTER TABLE bot_reply_audits
  ALTER COLUMN trigger_message_id DROP NOT NULL;

ALTER TABLE bot_reply_audits
  ADD COLUMN IF NOT EXISTS trigger_type TEXT NOT NULL DEFAULT 'user_message',
  ADD COLUMN IF NOT EXISTS llm_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS provider_initialized BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS api_key_present BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_trigger_type ON bot_reply_audits(trigger_type);
