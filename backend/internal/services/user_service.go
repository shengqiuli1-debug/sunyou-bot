package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"sunyou-bot/backend/internal/models"
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateGuest(ctx context.Context, nickname string) (*models.User, error) {
	if strings.TrimSpace(nickname) == "" {
		nickname = fmt.Sprintf("游客%s", time.Now().Format("150405"))
	}
	uid := uuid.NewString()
	token := uuid.NewString() + uuid.NewString()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, anon_token, nickname, points, free_trial_rooms, created_at)
		 VALUES ($1, $2, $3, 30, 1, NOW())`,
		uid, token, nickname,
	)
	if err != nil {
		return nil, err
	}
	return s.GetByToken(ctx, token)
}

func (s *UserService) GetByToken(ctx context.Context, token string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, anon_token, nickname, points, free_trial_rooms, created_at
		 FROM users WHERE anon_token=$1`, token,
	)
	var u models.User
	if err := row.Scan(&u.ID, &u.Token, &u.Nickname, &u.Points, &u.FreeTrialRooms, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, anon_token, nickname, points, free_trial_rooms, created_at
		 FROM users WHERE id=$1`, id,
	)
	var u models.User
	if err := row.Scan(&u.ID, &u.Token, &u.Nickname, &u.Points, &u.FreeTrialRooms, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) ListPointLedger(ctx context.Context, userID string, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, change_amount, balance_after, reason, meta, created_at
		 FROM point_ledger WHERE user_id=$1 ORDER BY id DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var change int
		var balance int
		var reason string
		var meta []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &change, &balance, &reason, &meta, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":           id,
			"changeAmount": change,
			"balanceAfter": balance,
			"reason":       reason,
			"meta":         string(meta),
			"createdAt":    createdAt,
		})
	}
	return out, rows.Err()
}

func (s *UserService) ListRoomRecords(ctx context.Context, userID string, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT r.id, r.share_code, r.bot_role, r.fire_level, r.duration_minutes, r.status, r.created_at, r.end_at
		 FROM rooms r
		 JOIN room_members rm ON rm.room_id=r.id
		 WHERE rm.user_id=$1
		 ORDER BY r.created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var (
			id, shareCode, botRole, fireLevel, status string
			duration                                  int
			createdAt, endAt                          time.Time
		)
		if err := rows.Scan(&id, &shareCode, &botRole, &fireLevel, &duration, &status, &createdAt, &endAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"roomId":          id,
			"shareCode":       shareCode,
			"botRole":         botRole,
			"fireLevel":       fireLevel,
			"durationMinutes": duration,
			"status":          status,
			"createdAt":       createdAt,
			"endAt":           endAt,
		})
	}
	return out, rows.Err()
}
