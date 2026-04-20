package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"

	"sunyou-bot/backend/internal/bot"
	"sunyou-bot/backend/internal/bot/character"
	"sunyou-bot/backend/internal/models"
	"sunyou-bot/backend/internal/risk"
)

var (
	ErrRoomEnded          = errors.New("room ended")
	ErrTargetNeedConfirm  = errors.New("target identity requires confirm")
	ErrNotOwner           = errors.New("only owner can do this action")
	ErrNotRoomMember      = errors.New("not room member")
	ErrInvalidIdentity    = errors.New("invalid identity")
	ErrInvalidRoomOptions = errors.New("invalid room options")
)

type CreateRoomInput struct {
	DurationMinutes int              `json:"durationMinutes"`
	BotRole         models.BotRole   `json:"botRole"`
	FireLevel       models.FireLevel `json:"fireLevel"`
	GenerateReport  bool             `json:"generateReport"`
}

type HostControlInput struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

type ChatUser struct {
	ID       string
	Nickname string
	Identity models.Identity
}

type ChatPayload struct {
	Content           string
	ReplyToMessageID  *int64
	ReplyToSenderName string
	ReplyToPreview    string
	ClientMsgID       string
}

type RoomService struct {
	db            *sql.DB
	redis         *redis.Client
	bot           *bot.Engine
	audits        *BotAuditService
	risk          *risk.Service
	reports       *ReportService
	coldSeconds   int64
	logger        *slog.Logger
	debugMinReply bool
}

func NewRoomService(
	db *sql.DB,
	redisCli *redis.Client,
	botEngine *bot.Engine,
	auditSvc *BotAuditService,
	riskSvc *risk.Service,
	reportSvc *ReportService,
	coldSeconds int,
	debugMinReply bool,
	logger *slog.Logger,
) *RoomService {
	return &RoomService{
		db: db, redis: redisCli, bot: botEngine, audits: auditSvc, risk: riskSvc, reports: reportSvc,
		coldSeconds:   int64(coldSeconds),
		debugMinReply: debugMinReply,
		logger:        logger,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, ownerID string, in CreateRoomInput) (*models.Room, bool, int, error) {
	if in.DurationMinutes != 5 && in.DurationMinutes != 15 && in.DurationMinutes != 30 {
		return nil, false, 0, ErrInvalidRoomOptions
	}
	if in.BotRole == "" {
		in.BotRole = models.RoleJudge
	}
	if in.FireLevel == "" {
		in.FireLevel = models.FireMedium
	}

	roomID := uuid.NewString()
	shareCode := genShareCode()
	cost := RoomCost(in.DurationMinutes, in.FireLevel)
	endAt := time.Now().Add(time.Duration(in.DurationMinutes) * time.Minute)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, 0, err
	}
	defer tx.Rollback()

	var points int
	var freeTrial int
	if err := tx.QueryRowContext(ctx, `SELECT points, free_trial_rooms FROM users WHERE id=$1 FOR UPDATE`, ownerID).Scan(&points, &freeTrial); err != nil {
		return nil, false, 0, err
	}

	useFree := false
	change := -cost
	if freeTrial > 0 {
		useFree = true
		change = 0
		freeTrial--
	}
	if !useFree && points < cost {
		return nil, false, points, ErrInsufficientPoints
	}
	remaining := points + change

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rooms (id, owner_user_id, share_code, bot_role, fire_level, generate_report, duration_minutes, cost_points, status, end_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active',$9,NOW())`,
		roomID, ownerID, shareCode, in.BotRole, in.FireLevel, in.GenerateReport, in.DurationMinutes, cost, endAt,
	); err != nil {
		return nil, false, 0, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO room_members (room_id, user_id, identity, is_owner, joined_at)
		 VALUES ($1,$2,$3,true,NOW())
		 ON CONFLICT (room_id, user_id) DO UPDATE SET identity=EXCLUDED.identity, is_owner=true`,
		roomID, ownerID, models.IdentityNormal,
	); err != nil {
		return nil, false, 0, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET points=$1, free_trial_rooms=$2 WHERE id=$3`,
		remaining, freeTrial, ownerID,
	); err != nil {
		return nil, false, 0, err
	}
	reason := "create_room"
	if useFree {
		reason = "create_room_free_trial"
	}
	meta, _ := json.Marshal(map[string]any{"roomId": roomID, "duration": in.DurationMinutes, "fireLevel": in.FireLevel})
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO point_ledger (user_id, change_amount, balance_after, reason, meta, created_at)
		 VALUES ($1,$2,$3,$4,$5::jsonb,NOW())`,
		ownerID, change, remaining, reason, string(meta),
	); err != nil {
		return nil, false, 0, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, 0, err
	}

	room := &models.Room{
		ID:              roomID,
		ShareCode:       shareCode,
		OwnerUserID:     ownerID,
		BotRole:         in.BotRole,
		FireLevel:       in.FireLevel,
		GenerateReport:  in.GenerateReport,
		DurationMinutes: in.DurationMinutes,
		CostPoints:      cost,
		Status:          models.RoomActive,
		EndAt:           endAt,
		CreatedAt:       time.Now(),
	}

	if err := s.cacheRoomMeta(ctx, room); err != nil {
		return nil, false, 0, err
	}
	if err := s.redis.HSet(ctx, s.membersKey(roomID), ownerID, string(models.IdentityNormal)).Err(); err == nil {
		_ = s.redis.ExpireAt(ctx, s.membersKey(roomID), endAt).Err()
	}

	return room, useFree, remaining, nil
}

func (s *RoomService) JoinRoom(ctx context.Context, roomID, userID string, identity models.Identity, confirmTarget bool) (*models.RoomMember, error) {
	if !isValidIdentity(identity) {
		return nil, ErrInvalidIdentity
	}
	if identity == models.IdentityTarget && !confirmTarget {
		return nil, ErrTargetNeedConfirm
	}

	room, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomActive || room.EndAt.Before(time.Now()) {
		_ = s.EndRoom(ctx, roomID, "expired")
		return nil, ErrRoomEnded
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO room_members (room_id, user_id, identity, is_owner, joined_at)
		 VALUES ($1,$2,$3,false,NOW())
		 ON CONFLICT (room_id, user_id)
		 DO UPDATE SET identity=EXCLUDED.identity, joined_at=COALESCE(room_members.joined_at, NOW())`,
		roomID, userID, identity,
	)
	if err != nil {
		return nil, err
	}

	if err := s.redis.HSet(ctx, s.membersKey(roomID), userID, string(identity)).Err(); err == nil {
		_ = s.redis.ExpireAt(ctx, s.membersKey(roomID), room.EndAt).Err()
	}

	return s.GetMember(ctx, roomID, userID)
}

func (s *RoomService) UpdateIdentity(ctx context.Context, roomID, userID string, identity models.Identity, confirmTarget bool) (*models.RoomMember, error) {
	if !isValidIdentity(identity) {
		return nil, ErrInvalidIdentity
	}
	if identity == models.IdentityTarget && !confirmTarget {
		return nil, ErrTargetNeedConfirm
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE room_members SET identity=$1 WHERE room_id=$2 AND user_id=$3`,
		identity, roomID, userID,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, ErrNotRoomMember
	}

	_ = s.redis.HSet(ctx, s.membersKey(roomID), userID, string(identity)).Err()
	return s.GetMember(ctx, roomID, userID)
}

func (s *RoomService) GetRoom(ctx context.Context, roomID string) (*models.Room, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, share_code, owner_user_id, bot_role, fire_level, generate_report, duration_minutes, cost_points, status, mute_bot_until, end_at, created_at, ended_at
		 FROM rooms WHERE id=$1`,
		roomID,
	)
	var room models.Room
	if err := row.Scan(
		&room.ID,
		&room.ShareCode,
		&room.OwnerUserID,
		&room.BotRole,
		&room.FireLevel,
		&room.GenerateReport,
		&room.DurationMinutes,
		&room.CostPoints,
		&room.Status,
		&room.MuteBotUntil,
		&room.EndAt,
		&room.CreatedAt,
		&room.EndedAt,
	); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *RoomService) GetRoomByShareCode(ctx context.Context, code string) (*models.Room, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, share_code, owner_user_id, bot_role, fire_level, generate_report, duration_minutes, cost_points, status, mute_bot_until, end_at, created_at, ended_at
		 FROM rooms WHERE share_code=$1`,
		strings.ToUpper(strings.TrimSpace(code)),
	)
	var room models.Room
	if err := row.Scan(
		&room.ID,
		&room.ShareCode,
		&room.OwnerUserID,
		&room.BotRole,
		&room.FireLevel,
		&room.GenerateReport,
		&room.DurationMinutes,
		&room.CostPoints,
		&room.Status,
		&room.MuteBotUntil,
		&room.EndAt,
		&room.CreatedAt,
		&room.EndedAt,
	); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *RoomService) GetMember(ctx context.Context, roomID, userID string) (*models.RoomMember, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT rm.room_id, rm.user_id, u.nickname, rm.identity, rm.is_owner, rm.joined_at
		 FROM room_members rm
		 JOIN users u ON u.id=rm.user_id
		 WHERE rm.room_id=$1 AND rm.user_id=$2`,
		roomID, userID,
	)
	var m models.RoomMember
	if err := row.Scan(&m.RoomID, &m.UserID, &m.Nickname, &m.Identity, &m.IsOwner, &m.JoinedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *RoomService) ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT rm.room_id, rm.user_id, u.nickname, rm.identity, rm.is_owner, rm.joined_at
		 FROM room_members rm
		 JOIN users u ON u.id=rm.user_id
		 WHERE rm.room_id=$1 ORDER BY rm.joined_at ASC`, roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.RoomMember, 0)
	for rows.Next() {
		var m models.RoomMember
		if err := rows.Scan(&m.RoomID, &m.UserID, &m.Nickname, &m.Identity, &m.IsOwner, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *RoomService) IsOwner(ctx context.Context, roomID, userID string) (bool, error) {
	var ownerID string
	if err := s.db.QueryRowContext(ctx, `SELECT owner_user_id FROM rooms WHERE id=$1`, roomID).Scan(&ownerID); err != nil {
		return false, err
	}
	return ownerID == userID, nil
}

func (s *RoomService) HandleChat(ctx context.Context, roomID string, user ChatUser, payload ChatPayload) (*models.ChatMessage, *bot.Input, string, error) {
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		return nil, nil, "empty", nil
	}
	if len([]rune(content)) > 300 {
		runes := []rune(content)
		content = string(runes[:300])
	}

	ok, err := s.risk.CheckRateLimit(ctx, roomID, user.ID)
	if err != nil {
		return nil, nil, "", err
	}
	if !ok {
		return nil, nil, "你发太快啦，等一下再来一句。", nil
	}

	if bad, hit := s.risk.ContainsSensitive(content); bad {
		_ = s.risk.LogRisk(ctx, roomID, user.ID, "sensitive", content, map[string]any{"hit": hit})
		return nil, nil, "这句可能会伤人，换个更有梗的说法吧。", nil
	}

	room, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, nil, "", err
	}
	if room.Status != models.RoomActive || room.EndAt.Before(time.Now()) {
		_ = s.EndRoom(ctx, roomID, "expired")
		return nil, nil, "房间已经结束。", ErrRoomEnded
	}

	var replyMsg *models.ChatMessage
	if payload.ReplyToMessageID != nil {
		if msg, err := s.GetMessageByID(ctx, roomID, *payload.ReplyToMessageID); err == nil && msg.SenderType != "system" {
			replyMsg = msg
		}
	}

	msgIn := createMessageInput{
		RoomID:       roomID,
		UserID:       &user.ID,
		Nickname:     user.Nickname,
		SenderType:   "user",
		Content:      content,
		ClientMsgID:  optString(payload.ClientMsgID),
		IsBotMessage: false,
	}
	if replyMsg != nil {
		msgIn.ReplyToMessageID = &replyMsg.ID
		msgIn.ReplyToSenderID = replyMsg.UserID
		name := replyMsg.Nickname
		preview := trimReplyPreview(replyMsg.Content)
		msgIn.ReplyToSenderName = &name
		msgIn.ReplyToContentPreview = &preview
	}

	userMsg, err := s.createMessage(ctx, msgIn)
	if err != nil {
		return nil, nil, "", err
	}
	s.updateStatsOnUserMsg(ctx, roomID, user.ID, content)
	s.cacheMessage(ctx, roomID, userMsg)

	recentMessages, _ := s.LatestMessages(ctx, roomID, 18)
	isCold := s.isColdScene(ctx, roomID)
	recentCount := s.recentMsgCount(ctx, roomID)
	lastTarget, _ := s.redis.Get(ctx, s.lastBotTargetKey(roomID)).Result()

	if s.logger != nil {
		s.logger.Info("[CHAT] chat.message.saved",
			"room_id", roomID,
			"message_id", userMsg.ID,
			"sender_id", user.ID,
			"sender_name", user.Nickname,
			"content", trimReplyPreview(content),
			"reply_to_message_id", payload.ReplyToMessageID,
		)
	}

	botInput := bot.Input{
		RoomID:           roomID,
		SpeakerID:        user.ID,
		SpeakerName:      user.Nickname,
		SpeakerIdentity:  user.Identity,
		SpeakerMessageID: userMsg.ID,
		ReplyToMessageID: payload.ReplyToMessageID,
		Content:          content,
		Role:             room.BotRole,
		FireLevel:        room.FireLevel,
		IsColdScene:      isCold,
		RecentMsgCount:   recentCount,
		ConsecutiveHitID: lastTarget,
		ReplyToMessage:   replyMsg,
		RecentMessages:   recentMessages,
		TargetMode:       user.Identity == models.IdentityTarget,
		ImmuneMode:       user.Identity == models.IdentityImmune,
	}
	return userMsg, &botInput, "", nil
}

func (s *RoomService) HandleBotReply(ctx context.Context, roomID string, userMsg *models.ChatMessage, botInput bot.Input) (*models.ChatMessage, error) {
	if userMsg == nil {
		return nil, nil
	}
	room, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomActive || room.EndAt.Before(time.Now()) {
		return nil, ErrRoomEnded
	}

	botOut := s.bot.MaybeReply(ctx, botInput)
	if botOut.Triggered {
		_ = s.redis.Del(ctx, s.noReplyStreakKey(roomID, botInput.SpeakerID)).Err()
	} else if s.debugMinReply {
		streak, _ := s.redis.Incr(ctx, s.noReplyStreakKey(roomID, botInput.SpeakerID)).Result()
		_ = s.redis.Expire(ctx, s.noReplyStreakKey(roomID, botInput.SpeakerID), 30*time.Minute).Err()
		if streak >= 3 {
			botOut = s.bot.RunMinReply(ctx, botInput)
			_ = s.redis.Del(ctx, s.noReplyStreakKey(roomID, botInput.SpeakerID)).Err()
		}
	}
	if !botOut.Triggered {
		return nil, nil
	}
	if bad, _ := s.risk.ContainsSensitive(botOut.Content); bad {
		botOut.Content = "【系统保护】这波先收敛一点，我们继续聊梗不聊人。"
	}

	role := character.NormalizeRole(botInput.Role)
	botName := botDisplayName(role)
	botIn := createMessageInput{
		RoomID:       roomID,
		Nickname:     botName,
		SenderType:   "bot",
		Content:      botOut.Content,
		IsBotMessage: true,
		BotRole:      &role,
	}
	botIn.ReplySource = optString(botOut.ReplySource)
	botIn.LLMModel = optString(botOut.LLMModel)
	botIn.FallbackReason = optString(botOut.FallbackReason)
	botIn.TraceID = optString(botOut.TraceID)
	botIn.ForceReply = botOut.ForceReply
	botIn.TriggerReason = optString(botOut.TriggerReason)
	botIn.HypeScore = &botOut.HypeScore
	botIn.ReplyToMessageID = &userMsg.ID
	botIn.ReplyToSenderID = userMsg.UserID
	senderName := userMsg.Nickname
	preview := trimReplyPreview(userMsg.Content)
	botIn.ReplyToSenderName = &senderName
	botIn.ReplyToContentPreview = &preview

	botMsg, err := s.createMessage(ctx, botIn)
	if err != nil {
		return nil, err
	}
	if botMsg.TraceID != nil && s.audits != nil {
		if err := s.audits.BindReplyMessage(ctx, *botMsg.TraceID, botMsg.ID); err != nil && s.logger != nil {
			s.logger.Error("bind bot audit reply message failed", "trace_id", *botMsg.TraceID, "reply_message_id", botMsg.ID, "error", err)
		}
	}
	if s.logger != nil {
		s.logger.Info("[CHAT] chat.message.saved",
			"room_id", roomID,
			"message_id", botMsg.ID,
			"sender_id", "",
			"sender_name", botName,
			"content", trimReplyPreview(botOut.Content),
			"reply_to_message_id", botIn.ReplyToMessageID,
		)
	}
	s.cacheMessage(ctx, roomID, botMsg)
	s.updateStatsOnBotMsg(ctx, roomID, botMsg.Content)
	if botOut.TargetID != "" {
		_ = s.redis.Set(ctx, s.lastBotTargetKey(roomID), botOut.TargetID, 10*time.Minute).Err()
	}
	return botMsg, nil
}

func (s *RoomService) DebugLLMStatus() bot.DebugStatus {
	return s.bot.DebugStatus()
}

func (s *RoomService) RunLLMTest(ctx context.Context, roomID, userID string) (*models.ChatMessage, error) {
	isOwner, err := s.IsOwner(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, ErrNotOwner
	}
	room, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.Status != models.RoomActive || room.EndAt.Before(time.Now()) {
		return nil, ErrRoomEnded
	}
	member, err := s.GetMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	recentMessages, _ := s.LatestMessages(ctx, roomID, 18)
	lastTarget, _ := s.redis.Get(ctx, s.lastBotTargetKey(roomID)).Result()
	out := s.bot.RunLLMTest(ctx, bot.Input{
		RoomID:           roomID,
		SpeakerID:        member.UserID,
		SpeakerName:      member.Nickname,
		SpeakerIdentity:  member.Identity,
		SpeakerMessageID: 0,
		Content:          "房主触发 LLM 测试，请输出一条短句",
		Role:             room.BotRole,
		FireLevel:        room.FireLevel,
		IsColdScene:      false,
		RecentMsgCount:   s.recentMsgCount(ctx, roomID),
		ConsecutiveHitID: lastTarget,
		RecentMessages:   recentMessages,
		TargetMode:       member.Identity == models.IdentityTarget,
		ImmuneMode:       member.Identity == models.IdentityImmune,
	})
	if !out.Triggered {
		return nil, nil
	}

	botIn := createMessageInput{
		RoomID:         roomID,
		Nickname:       botDisplayName(room.BotRole),
		SenderType:     "bot",
		Content:        out.Content,
		IsBotMessage:   true,
		BotRole:        &room.BotRole,
		ReplySource:    optString(out.ReplySource),
		LLMModel:       optString(out.LLMModel),
		FallbackReason: optString(out.FallbackReason),
		TraceID:        optString(out.TraceID),
		ForceReply:     out.ForceReply,
		TriggerReason:  optString(out.TriggerReason),
		HypeScore:      &out.HypeScore,
	}
	msg, err := s.createMessage(ctx, botIn)
	if err != nil {
		return nil, err
	}
	if msg.TraceID != nil && s.audits != nil {
		if err := s.audits.BindReplyMessage(ctx, *msg.TraceID, msg.ID); err != nil && s.logger != nil {
			s.logger.Error("bind bot audit reply message failed", "trace_id", *msg.TraceID, "reply_message_id", msg.ID, "error", err)
		}
	}
	s.cacheMessage(ctx, roomID, msg)
	s.updateStatsOnBotMsg(ctx, roomID, msg.Content)
	return msg, nil
}

func (s *RoomService) HostControl(ctx context.Context, roomID, userID string, in HostControlInput) (*models.Room, error) {
	isOwner, err := s.IsOwner(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, ErrNotOwner
	}

	switch in.Action {
	case "switch_role":
		role := models.BotRole(in.Value)
		if role == "" {
			role = models.RoleJudge
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE rooms SET bot_role=$1 WHERE id=$2`, role, roomID); err != nil {
			return nil, err
		}
		_ = s.redis.HSet(ctx, s.metaKey(roomID), "bot_role", string(role)).Err()
	case "switch_fire":
		fire := models.FireLevel(in.Value)
		if fire == "" {
			fire = models.FireMedium
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE rooms SET fire_level=$1 WHERE id=$2`, fire, roomID); err != nil {
			return nil, err
		}
		_ = s.redis.HSet(ctx, s.metaKey(roomID), "fire_level", string(fire)).Err()
	case "mute_bot":
		sec := 20
		if v, err := strconv.Atoi(in.Value); err == nil && v > 0 && v <= 120 {
			sec = v
		}
		until := time.Now().Add(time.Duration(sec) * time.Second)
		if _, err := s.db.ExecContext(ctx, `UPDATE rooms SET mute_bot_until=$1 WHERE id=$2`, until, roomID); err != nil {
			return nil, err
		}
		_ = s.redis.Set(ctx, s.botMutedKey(roomID), "1", time.Duration(sec)*time.Second).Err()
		_ = s.redis.HSet(ctx, s.metaKey(roomID), "mute_bot_until", until.Format(time.RFC3339)).Err()
	case "end_room":
		if err := s.EndRoom(ctx, roomID, "manual"); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown action: %s", in.Action)
	}

	return s.GetRoom(ctx, roomID)
}

func (s *RoomService) EndRoom(ctx context.Context, roomID, reason string) error {
	room, err := s.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if room.Status != models.RoomActive {
		return nil
	}

	status := models.RoomEnded
	if reason == "expired" {
		status = models.RoomExpired
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE rooms SET status=$1, ended_at=NOW() WHERE id=$2 AND status='active'`,
		status, roomID,
	)
	if err != nil {
		return err
	}
	_ = s.redis.HSet(ctx, s.metaKey(roomID), "status", string(status)).Err()
	_ = s.redis.Expire(ctx, s.metaKey(roomID), 2*time.Hour).Err()
	_ = s.redis.Expire(ctx, s.membersKey(roomID), 2*time.Hour).Err()
	_ = s.redis.Expire(ctx, s.msgKey(roomID), 2*time.Hour).Err()
	_ = s.redis.Expire(ctx, s.statsKey(roomID), 2*time.Hour).Err()

	if room.GenerateReport {
		_, _ = s.reports.BuildAndPersist(ctx, roomID)
	}
	return nil
}

func (s *RoomService) ExpireDueRooms(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM rooms WHERE status='active' AND end_at <= NOW() LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		_ = s.EndRoom(ctx, id, "expired")
	}
	return nil
}

func (s *RoomService) LatestMessages(ctx context.Context, roomID string, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, room_id, user_id, nickname, sender_type, content,
		        reply_to_message_id, reply_to_sender_id, reply_to_sender_name, reply_to_content_preview,
		        is_bot_message, bot_role, reply_source, llm_model, fallback_reason, trace_id, force_reply, trigger_reason, hype_score, created_at
		 FROM messages WHERE room_id=$1 ORDER BY id DESC LIMIT $2`,
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]models.ChatMessage, 0)
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(
			&m.ID, &m.RoomID, &m.UserID, &m.Nickname, &m.SenderType, &m.Content,
			&m.ReplyToMessageID, &m.ReplyToSenderID, &m.ReplyToSenderName, &m.ReplyToContentPreview,
			&m.IsBotMessage, &m.BotRole, &m.ReplySource, &m.LLMModel, &m.FallbackReason, &m.TraceID, &m.ForceReply, &m.TriggerReason, &m.HypeScore, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if m.IsBotMessage && (m.BotRole == nil || strings.TrimSpace(string(*m.BotRole)) == "") {
			role := character.InferRoleFromNickname(m.Nickname)
			m.BotRole = &role
		}
		list = append([]models.ChatMessage{m}, list...)
	}
	return list, rows.Err()
}

func (s *RoomService) GetMessageByID(ctx context.Context, roomID string, id int64) (*models.ChatMessage, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, room_id, user_id, nickname, sender_type, content,
		        reply_to_message_id, reply_to_sender_id, reply_to_sender_name, reply_to_content_preview,
		        is_bot_message, bot_role, reply_source, llm_model, fallback_reason, trace_id, force_reply, trigger_reason, hype_score, created_at
		 FROM messages WHERE room_id=$1 AND id=$2`,
		roomID, id,
	)
	var m models.ChatMessage
	if err := row.Scan(
		&m.ID, &m.RoomID, &m.UserID, &m.Nickname, &m.SenderType, &m.Content,
		&m.ReplyToMessageID, &m.ReplyToSenderID, &m.ReplyToSenderName, &m.ReplyToContentPreview,
		&m.IsBotMessage, &m.BotRole, &m.ReplySource, &m.LLMModel, &m.FallbackReason, &m.TraceID, &m.ForceReply, &m.TriggerReason, &m.HypeScore, &m.CreatedAt,
	); err != nil {
		return nil, err
	}
	if m.IsBotMessage && (m.BotRole == nil || strings.TrimSpace(string(*m.BotRole)) == "") {
		role := character.InferRoleFromNickname(m.Nickname)
		m.BotRole = &role
	}
	return &m, nil
}

func (s *RoomService) SubmitAbuseReport(ctx context.Context, roomID, reporterID, targetUserID, message string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO abuse_reports (room_id, reporter_user_id, target_user_id, message, created_at)
		 VALUES ($1, $2, NULLIF($3,''), $4, NOW())`,
		roomID, reporterID, targetUserID, message,
	)
	return err
}

func (s *RoomService) GetOrBuildReport(ctx context.Context, roomID string) (*models.RoomReport, error) {
	r, err := s.reports.Get(ctx, roomID)
	if err == nil {
		return r, nil
	}
	return s.reports.BuildAndPersist(ctx, roomID)
}

type createMessageInput struct {
	RoomID                string
	UserID                *string
	Nickname              string
	SenderType            string
	Content               string
	ClientMsgID           *string
	ReplyToMessageID      *int64
	ReplyToSenderID       *string
	ReplyToSenderName     *string
	ReplyToContentPreview *string
	IsBotMessage          bool
	BotRole               *models.BotRole
	ReplySource           *string
	LLMModel              *string
	FallbackReason        *string
	TraceID               *string
	ForceReply            bool
	TriggerReason         *string
	HypeScore             *int
}

func (s *RoomService) createMessage(ctx context.Context, in createMessageInput) (*models.ChatMessage, error) {
	dbBotRole := normalizeBotRolePtr(in.BotRole)
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO messages (
		    room_id, user_id, sender_type, nickname, content,
		    reply_to_message_id, reply_to_sender_id, reply_to_sender_name, reply_to_content_preview,
		    is_bot_message, bot_role, reply_source, llm_model, fallback_reason, trace_id, force_reply, trigger_reason, hype_score, created_at
		  )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW())
		 RETURNING id, created_at`,
		in.RoomID, sanitizeUUIDPtr(in.UserID), in.SenderType, in.Nickname, in.Content,
		in.ReplyToMessageID, sanitizeUUIDPtr(in.ReplyToSenderID), in.ReplyToSenderName, in.ReplyToContentPreview,
		in.IsBotMessage, dbBotRole, in.ReplySource, in.LLMModel, in.FallbackReason, in.TraceID, in.ForceReply, in.TriggerReason, in.HypeScore,
	)
	msg := &models.ChatMessage{
		RoomID:                in.RoomID,
		UserID:                in.UserID,
		SenderType:            in.SenderType,
		Nickname:              in.Nickname,
		Content:               in.Content,
		ClientMsgID:           in.ClientMsgID,
		ReplyToMessageID:      in.ReplyToMessageID,
		ReplyToSenderID:       in.ReplyToSenderID,
		ReplyToSenderName:     in.ReplyToSenderName,
		ReplyToContentPreview: in.ReplyToContentPreview,
		IsBotMessage:          in.IsBotMessage,
		BotRole:               normalizeBotRoleModelPtr(in.BotRole),
		ReplySource:           in.ReplySource,
		LLMModel:              in.LLMModel,
		FallbackReason:        in.FallbackReason,
		TraceID:               in.TraceID,
		ForceReply:            in.ForceReply,
		TriggerReason:         in.TriggerReason,
		HypeScore:             in.HypeScore,
	}
	if err := row.Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return nil, err
	}
	return msg, nil
}

func sanitizeUUIDPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil
	}
	out := trimmed
	return &out
}

func (s *RoomService) cacheMessage(ctx context.Context, roomID string, msg *models.ChatMessage) {
	data, _ := json.Marshal(msg)
	pipe := s.redis.TxPipeline()
	pipe.LPush(ctx, s.msgKey(roomID), string(data))
	pipe.LTrim(ctx, s.msgKey(roomID), 0, 199)
	pipe.Expire(ctx, s.msgKey(roomID), 2*time.Hour)
	pipe.Incr(ctx, s.msgCountKey(roomID))
	pipe.Expire(ctx, s.msgCountKey(roomID), 2*time.Hour)
	pipe.Set(ctx, s.lastMsgAtKey(roomID), time.Now().Unix(), 2*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *RoomService) updateStatsOnUserMsg(ctx context.Context, roomID, userID, content string) {
	pipe := s.redis.TxPipeline()
	lower := strings.ToLower(content)
	if containsAny(lower, []string{"都行", "随便", "?", "？？", "懂了", "呵", "行吧"}) {
		pipe.HIncrBy(ctx, s.statsKey(roomID), "hardmouth:"+userID, 1)
	}
	if containsAny(lower, []string{"哈哈", "+1", "确实", "绝了", "笑死", "补刀"}) {
		pipe.HIncrBy(ctx, s.statsKey(roomID), "assist:"+userID, 1)
	}
	if containsAny(lower, []string{"其实", "我的意思是", "本来想", "你们误会了"}) {
		pipe.HIncrBy(ctx, s.statsKey(roomID), "saveface:"+userID, 1)
	}

	lastVal, _ := s.redis.Get(ctx, s.lastMsgAtKey(roomID)).Result()
	lastUnix, _ := strconv.ParseInt(lastVal, 10, 64)
	if lastUnix > 0 {
		gap := int(time.Now().Unix() - lastUnix)
		curMax, _ := s.redis.HGet(ctx, s.statsKey(roomID), "quiet_max_secs").Int()
		if gap > curMax {
			pipe.HSet(ctx, s.statsKey(roomID), "quiet_max_secs", gap)
			pipe.HSet(ctx, s.statsKey(roomID), "quiet_at_unix", time.Now().Unix())
		}
	}
	pipe.Expire(ctx, s.statsKey(roomID), 2*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *RoomService) updateStatsOnBotMsg(ctx context.Context, roomID, content string) {
	_ = s.redis.HSet(ctx, s.statsKey(roomID), "bot_quote", content).Err()
	_ = s.redis.Expire(ctx, s.statsKey(roomID), 2*time.Hour).Err()
}

func (s *RoomService) isColdScene(ctx context.Context, roomID string) bool {
	lastVal, err := s.redis.Get(ctx, s.lastMsgAtKey(roomID)).Result()
	if err != nil {
		return false
	}
	lastUnix, _ := strconv.ParseInt(lastVal, 10, 64)
	if lastUnix == 0 {
		return false
	}
	threshold := s.coldSeconds
	if threshold <= 0 {
		threshold = 18
	}
	return time.Now().Unix()-lastUnix > threshold
}

func (s *RoomService) recentMsgCount(ctx context.Context, roomID string) int {
	v, _ := s.redis.Get(ctx, s.msgCountKey(roomID)).Int()
	return v
}

func (s *RoomService) cacheRoomMeta(ctx context.Context, room *models.Room) error {
	values := map[string]any{
		"id":              room.ID,
		"share_code":      room.ShareCode,
		"owner_user_id":   room.OwnerUserID,
		"bot_role":        string(room.BotRole),
		"fire_level":      string(room.FireLevel),
		"status":          string(room.Status),
		"generate_report": room.GenerateReport,
		"end_at":          room.EndAt.Unix(),
	}
	if err := s.redis.HSet(ctx, s.metaKey(room.ID), values).Err(); err != nil {
		return err
	}
	_ = s.redis.ExpireAt(ctx, s.metaKey(room.ID), room.EndAt).Err()
	return nil
}

func (s *RoomService) metaKey(roomID string) string      { return "room:" + roomID + ":meta" }
func (s *RoomService) membersKey(roomID string) string   { return "room:" + roomID + ":members" }
func (s *RoomService) onlineKey(roomID string) string    { return "room:" + roomID + ":online" }
func (s *RoomService) msgKey(roomID string) string       { return "room:" + roomID + ":messages" }
func (s *RoomService) statsKey(roomID string) string     { return "room:" + roomID + ":stats" }
func (s *RoomService) msgCountKey(roomID string) string  { return "room:" + roomID + ":msg_count" }
func (s *RoomService) lastMsgAtKey(roomID string) string { return "room:" + roomID + ":last_msg_at" }
func (s *RoomService) botMutedKey(roomID string) string  { return "room:" + roomID + ":bot:muted" }
func (s *RoomService) noReplyStreakKey(roomID, userID string) string {
	return "room:" + roomID + ":bot:no_reply_streak:" + userID
}
func (s *RoomService) lastBotTargetKey(roomID string) string {
	return "room:" + roomID + ":bot:last_target"
}

func isValidIdentity(id models.Identity) bool {
	return id == models.IdentityNormal || id == models.IdentityTarget || id == models.IdentityImmune
}

func genShareCode() string {
	letters := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")
	b := make([]rune, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func botDisplayName(role models.BotRole) string {
	return character.DisplayName(character.NormalizeRole(role))
}

func containsAny(content string, keys []string) bool {
	for _, k := range keys {
		if strings.Contains(content, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func trimReplyPreview(content string) string {
	content = strings.TrimSpace(content)
	r := []rune(content)
	if len(r) <= 42 {
		return content
	}
	return string(r[:42]) + "..."
}

func optString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	cp := v
	return &cp
}

func normalizeBotRolePtr(v *models.BotRole) *string {
	if v == nil {
		return nil
	}
	role := string(character.NormalizeRole(*v))
	if strings.TrimSpace(role) == "" {
		return nil
	}
	return &role
}

func normalizeBotRoleModelPtr(v *models.BotRole) *models.BotRole {
	if v == nil {
		return nil
	}
	role := character.NormalizeRole(*v)
	return &role
}

func (s *RoomService) MarkOnline(ctx context.Context, roomID, userID string) {
	pipe := s.redis.TxPipeline()
	pipe.SAdd(ctx, s.onlineKey(roomID), userID)
	pipe.Expire(ctx, s.onlineKey(roomID), 2*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *RoomService) MarkOffline(ctx context.Context, roomID, userID string) {
	_ = s.redis.SRem(ctx, s.onlineKey(roomID), userID).Err()
}
