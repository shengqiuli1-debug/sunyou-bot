package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"

	"sunyou-bot/backend/internal/models"
)

type ReportService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewReportService(db *sql.DB, redisCli *redis.Client) *ReportService {
	return &ReportService{db: db, redis: redisCli}
}

func (s *ReportService) statsKey(roomID string) string {
	return "room:" + roomID + ":stats"
}

func (s *ReportService) BuildAndPersist(ctx context.Context, roomID string) (*models.RoomReport, error) {
	report, err := s.Build(ctx, roomID)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(report)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO room_reports (room_id, hardmouth_label, best_assist_label, quiet_moment_secs, quiet_moment_at, saveface_label, bot_quote, raw_json, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,NOW())
		 ON CONFLICT (room_id) DO UPDATE SET
		 hardmouth_label=EXCLUDED.hardmouth_label,
		 best_assist_label=EXCLUDED.best_assist_label,
		 quiet_moment_secs=EXCLUDED.quiet_moment_secs,
		 quiet_moment_at=EXCLUDED.quiet_moment_at,
		 saveface_label=EXCLUDED.saveface_label,
		 bot_quote=EXCLUDED.bot_quote,
		 raw_json=EXCLUDED.raw_json,
		 created_at=NOW()`,
		report.RoomID,
		report.HardmouthLabel,
		report.BestAssistLabel,
		report.QuietMomentSecs,
		report.QuietMomentAt,
		report.SavefaceLabel,
		report.BotQuote,
		string(raw),
	)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (s *ReportService) Build(ctx context.Context, roomID string) (*models.RoomReport, error) {
	stats, err := s.redis.HGetAll(ctx, s.statsKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	hardmouthUserID, hardmouthScore := winnerByPrefix(stats, "hardmouth:")
	assistUserID, assistScore := winnerByPrefix(stats, "assist:")
	savefaceUserID, savefaceScore := winnerByPrefix(stats, "saveface:")
	quietSecs := toInt(stats["quiet_max_secs"])
	quietAtUnix := toInt64(stats["quiet_at_unix"])
	botQuote := stats["bot_quote"]
	if strings.TrimSpace(botQuote) == "" {
		botQuote = "今天我话不多，但每句都算数。"
	}

	report := &models.RoomReport{
		RoomID:          roomID,
		HardmouthLabel:  s.labelFor(ctx, hardmouthUserID, "今日最嘴硬", hardmouthScore),
		BestAssistLabel: s.labelFor(ctx, assistUserID, "今日最佳补刀", assistScore),
		SavefaceLabel:   s.labelFor(ctx, savefaceUserID, "今日最会挽尊", savefaceScore),
		QuietMomentSecs: quietSecs,
		BotQuote:        botQuote,
		CreatedAt:       time.Now(),
	}
	if quietAtUnix > 0 {
		t := time.Unix(quietAtUnix, 0)
		report.QuietMomentAt = &t
	}
	return report, nil
}

func (s *ReportService) Get(ctx context.Context, roomID string) (*models.RoomReport, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT room_id, hardmouth_label, best_assist_label, quiet_moment_secs, quiet_moment_at, saveface_label, bot_quote, created_at
		 FROM room_reports WHERE room_id=$1`, roomID,
	)
	var r models.RoomReport
	if err := row.Scan(&r.RoomID, &r.HardmouthLabel, &r.BestAssistLabel, &r.QuietMomentSecs, &r.QuietMomentAt, &r.SavefaceLabel, &r.BotQuote, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *ReportService) labelFor(ctx context.Context, userID, title string, score int) string {
	if userID == "" || score == 0 {
		return fmt.Sprintf("%s：全员保守发挥", title)
	}
	var nickname string
	if err := s.db.QueryRowContext(ctx, `SELECT nickname FROM users WHERE id=$1`, userID).Scan(&nickname); err != nil {
		nickname = "某位选手"
	}
	return fmt.Sprintf("%s：%s（%d 次）", title, nickname, score)
}

func winnerByPrefix(stats map[string]string, prefix string) (string, int) {
	winnerID := ""
	winnerScore := 0
	for key, val := range stats {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		score := toInt(val)
		if score > winnerScore {
			winnerScore = score
			winnerID = strings.TrimPrefix(key, prefix)
		}
	}
	return winnerID, winnerScore
}

func toInt(v string) int {
	i, _ := strconv.Atoi(v)
	return i
}

func toInt64(v string) int64 {
	i, _ := strconv.ParseInt(v, 10, 64)
	return i
}
