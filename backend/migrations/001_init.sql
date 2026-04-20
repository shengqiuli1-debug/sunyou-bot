CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  anon_token TEXT UNIQUE NOT NULL,
  nickname TEXT NOT NULL,
  points INT NOT NULL DEFAULT 30,
  free_trial_rooms INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS point_ledger (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  change_amount INT NOT NULL,
  balance_after INT NOT NULL,
  reason TEXT NOT NULL,
  meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_point_ledger_user_created ON point_ledger(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS rooms (
  id UUID PRIMARY KEY,
  owner_user_id UUID NOT NULL REFERENCES users(id),
  share_code TEXT UNIQUE NOT NULL,
  bot_role TEXT NOT NULL,
  fire_level TEXT NOT NULL,
  generate_report BOOLEAN NOT NULL DEFAULT TRUE,
  duration_minutes INT NOT NULL,
  cost_points INT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  mute_bot_until TIMESTAMPTZ NULL,
  end_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ended_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_rooms_status_end ON rooms(status, end_at);
CREATE INDEX IF NOT EXISTS idx_rooms_share_code ON rooms(share_code);

CREATE TABLE IF NOT EXISTS room_members (
  id BIGSERIAL PRIMARY KEY,
  room_id UUID NOT NULL REFERENCES rooms(id),
  user_id UUID NOT NULL REFERENCES users(id),
  identity TEXT NOT NULL,
  is_owner BOOLEAN NOT NULL DEFAULT FALSE,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at TIMESTAMPTZ NULL,
  UNIQUE(room_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_room_members_room ON room_members(room_id);

CREATE TABLE IF NOT EXISTS messages (
  id BIGSERIAL PRIMARY KEY,
  room_id UUID NOT NULL REFERENCES rooms(id),
  user_id UUID NULL REFERENCES users(id),
  sender_type TEXT NOT NULL,
  nickname TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id, id DESC);

CREATE TABLE IF NOT EXISTS room_reports (
  room_id UUID PRIMARY KEY REFERENCES rooms(id),
  hardmouth_label TEXT NOT NULL,
  best_assist_label TEXT NOT NULL,
  quiet_moment_secs INT NOT NULL DEFAULT 0,
  quiet_moment_at TIMESTAMPTZ NULL,
  saveface_label TEXT NOT NULL,
  bot_quote TEXT NOT NULL,
  raw_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS abuse_reports (
  id BIGSERIAL PRIMARY KEY,
  room_id UUID NOT NULL REFERENCES rooms(id),
  reporter_user_id UUID NOT NULL REFERENCES users(id),
  target_user_id UUID NULL REFERENCES users(id),
  message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_room_created ON abuse_reports(room_id, created_at DESC);

CREATE TABLE IF NOT EXISTS risk_logs (
  id BIGSERIAL PRIMARY KEY,
  room_id UUID NULL REFERENCES rooms(id),
  user_id UUID NULL REFERENCES users(id),
  risk_type TEXT NOT NULL,
  content TEXT NOT NULL,
  detail JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_risk_logs_created ON risk_logs(created_at DESC);
