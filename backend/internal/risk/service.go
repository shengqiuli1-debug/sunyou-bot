package risk

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Service struct {
	db             *sql.DB
	redis          *redis.Client
	sensitiveWords []string
	maxMsgPer10s   int
}

func NewService(db *sql.DB, redisCli *redis.Client, words []string, maxMsgPer10s int) *Service {
	return &Service{db: db, redis: redisCli, sensitiveWords: words, maxMsgPer10s: maxMsgPer10s}
}

func (s *Service) CheckRateLimit(ctx context.Context, roomID, userID string) (bool, error) {
	if s.maxMsgPer10s <= 0 {
		return true, nil
	}
	key := "risk:rate:" + roomID + ":" + userID
	cnt, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if cnt == 1 {
		_ = s.redis.Expire(ctx, key, 10*time.Second).Err()
	}
	if int(cnt) > s.maxMsgPer10s {
		_ = s.LogRisk(ctx, roomID, userID, "rate_limit", "message too fast", map[string]any{"count": cnt})
		return false, nil
	}
	return true, nil
}

func (s *Service) ContainsSensitive(content string) (bool, string) {
	lower := strings.ToLower(content)
	for _, w := range s.sensitiveWords {
		ww := strings.ToLower(strings.TrimSpace(w))
		if ww == "" {
			continue
		}
		if strings.Contains(lower, ww) {
			return true, w
		}
	}
	return false, ""
}

func (s *Service) MaskSensitive(content string) string {
	masked := content
	for _, w := range s.sensitiveWords {
		ww := strings.TrimSpace(w)
		if ww == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, ww, "***")
		masked = strings.ReplaceAll(strings.ToLower(masked), strings.ToLower(ww), "***")
	}
	return masked
}

func (s *Service) LogRisk(ctx context.Context, roomID, userID, riskType, content string, detail map[string]any) error {
	payload, _ := json.Marshal(detail)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO risk_logs (room_id, user_id, risk_type, content, detail, created_at)
		 VALUES (NULLIF($1,''), NULLIF($2,''), $3, $4, $5::jsonb, NOW())`,
		roomID, userID, riskType, content, string(payload),
	)
	return err
}
