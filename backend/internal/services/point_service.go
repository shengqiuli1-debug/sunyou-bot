package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"sunyou-bot/backend/internal/models"
)

var ErrInsufficientPoints = errors.New("insufficient points")

type PointService struct {
	db *sql.DB
}

func NewPointService(db *sql.DB) *PointService {
	return &PointService{db: db}
}

func RoomCost(duration int, fire models.FireLevel) int {
	base := 2
	switch duration {
	case 15:
		base = 5
	case 30:
		base = 8
	}
	extra := 0
	if fire == models.FireHigh {
		extra = 2
	}
	if fire == models.FireLow {
		extra = -1
	}
	cost := base + extra
	if cost < 1 {
		cost = 1
	}
	return cost
}

func (s *PointService) Recharge(ctx context.Context, userID string, amount int, channel string) (int, error) {
	if amount <= 0 {
		amount = 10
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var points int
	if err := tx.QueryRowContext(ctx, `SELECT points FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&points); err != nil {
		return 0, err
	}
	points += amount
	if _, err := tx.ExecContext(ctx, `UPDATE users SET points=$1 WHERE id=$2`, points, userID); err != nil {
		return 0, err
	}
	meta, _ := json.Marshal(map[string]any{"channel": channel, "amount": amount})
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO point_ledger (user_id, change_amount, balance_after, reason, meta, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, NOW())`,
		userID, amount, points, "mock_recharge", string(meta),
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return points, nil
}

func (s *PointService) DeductForRoom(ctx context.Context, userID string, duration int, fire models.FireLevel, roomID string) (cost int, useFree bool, remaining int, err error) {
	cost = RoomCost(duration, fire)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, 0, err
	}
	defer tx.Rollback()

	var points int
	var freeTrial int
	if err = tx.QueryRowContext(ctx, `SELECT points, free_trial_rooms FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&points, &freeTrial); err != nil {
		return 0, false, 0, err
	}

	change := -cost
	if freeTrial > 0 {
		useFree = true
		change = 0
		freeTrial--
	}

	if !useFree && points < cost {
		return 0, false, points, ErrInsufficientPoints
	}

	remaining = points + change
	if _, err = tx.ExecContext(ctx,
		`UPDATE users SET points=$1, free_trial_rooms=$2 WHERE id=$3`,
		remaining, freeTrial, userID,
	); err != nil {
		return 0, false, 0, err
	}

	reason := "create_room"
	if useFree {
		reason = "create_room_free_trial"
	}
	meta, _ := json.Marshal(map[string]any{"roomId": roomID, "duration": duration, "fireLevel": fire})
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO point_ledger (user_id, change_amount, balance_after, reason, meta, created_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, NOW())`,
		userID, change, remaining, reason, string(meta),
	); err != nil {
		return 0, false, 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, false, 0, err
	}
	return cost, useFree, remaining, nil
}

func (s *PointService) EnsureBalance(ctx context.Context, userID string) (int, error) {
	var points int
	if err := s.db.QueryRowContext(ctx, `SELECT points FROM users WHERE id=$1`, userID).Scan(&points); err != nil {
		return 0, fmt.Errorf("load points: %w", err)
	}
	return points, nil
}
