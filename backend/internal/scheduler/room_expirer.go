package scheduler

import (
	"context"
	"log/slog"
	"time"

	"sunyou-bot/backend/internal/services"
)

type RoomExpirer struct {
	logger *slog.Logger
	rooms  *services.RoomService
	quit   chan struct{}
}

func NewRoomExpirer(logger *slog.Logger, rooms *services.RoomService) *RoomExpirer {
	return &RoomExpirer{logger: logger, rooms: rooms, quit: make(chan struct{})}
}

func (e *RoomExpirer) Start() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := e.rooms.ExpireDueRooms(context.Background()); err != nil {
					e.logger.Error("expire rooms failed", "error", err)
				}
			case <-e.quit:
				return
			}
		}
	}()
}

func (e *RoomExpirer) Stop() {
	close(e.quit)
}
