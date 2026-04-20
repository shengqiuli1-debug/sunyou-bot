ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS bot_role TEXT NULL;

UPDATE messages
SET bot_role = CASE
  WHEN is_bot_message = TRUE AND (bot_role IS NULL OR bot_role = '') AND nickname ILIKE '%阴阳裁判%' THEN 'judge'
  WHEN is_bot_message = TRUE AND (bot_role IS NULL OR bot_role = '') AND nickname ILIKE '%冷面旁白%' THEN 'narrator'
  WHEN is_bot_message = TRUE AND (bot_role IS NULL OR bot_role = '') THEN 'npc'
  ELSE bot_role
END;

CREATE INDEX IF NOT EXISTS idx_messages_bot_role ON messages(bot_role);
