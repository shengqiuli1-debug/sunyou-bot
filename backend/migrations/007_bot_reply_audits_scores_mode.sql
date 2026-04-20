ALTER TABLE bot_reply_audits
  ADD COLUMN IF NOT EXISTS absurdity_score INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS risk_score INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reply_mode TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_bot_reply_audits_reply_mode ON bot_reply_audits(reply_mode);
