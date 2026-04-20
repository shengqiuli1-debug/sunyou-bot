package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"sunyou-bot/backend/internal/bot"
	"sunyou-bot/backend/internal/models"
)

type BotAuditListFilter struct {
	Page        int
	PageSize    int
	ReplySource string
	BotRole     string
}

type BotAuditService struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewBotAuditService(db *sql.DB, logger *slog.Logger) *BotAuditService {
	return &BotAuditService{db: db, logger: logger}
}

func (s *BotAuditService) CaptureDebugEvent(ctx context.Context, evt bot.DebugEvent) error {
	traceID := strings.TrimSpace(evt.TraceID)
	if traceID == "" {
		return nil
	}
	switch evt.Event {
	case "bot.trigger.enter":
		return s.upsertTriggerEnter(ctx, evt)
	case "bot.trigger.skip":
		if err := s.upsertTriggerEnter(ctx, evt); err != nil {
			return err
		}
		return s.updateTriggerSkip(ctx, evt)
	case "bot.trigger.hit":
		if strings.TrimSpace(evt.RoomID) == "" {
			return nil
		}
		return s.upsertTriggerHit(ctx, evt)
	case "bot.llm.prepare":
		return s.updateLLMPrepare(ctx, evt)
	case "bot.llm.success":
		return s.updateLLMSuccess(ctx, evt)
	case "bot.llm.error":
		return s.updateLLMError(ctx, evt)
	case "bot.llm.fallback":
		return s.updateLLMFallback(ctx, evt)
	case "bot.reply.done":
		return s.updateReplyDone(ctx, evt)
	default:
		return nil
	}
}

func (s *BotAuditService) BindReplyMessage(ctx context.Context, traceID string, replyMessageID int64) error {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" || replyMessageID <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE bot_reply_audits
		 SET reply_message_id=$2, updated_at=NOW()
		 WHERE trace_id=$1`,
		traceID, replyMessageID,
	)
	return err
}

func (s *BotAuditService) upsertTriggerEnter(ctx context.Context, evt bot.DebugEvent) error {
	triggerType := auditTriggerType(evt)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO bot_reply_audits (
  trace_id, room_id, trigger_message_id, trigger_sender_id, trigger_sender_name,
  reply_source, bot_role, firepower_level, trigger_reason, trigger_type, force_reply, hype_score,
  absurdity_score, risk_score, reply_mode,
  llm_enabled, provider_initialized, api_key_present, displayable_content_found, reasoning_only_response, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,'template',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NOW(),NOW())
ON CONFLICT (trace_id) DO UPDATE SET
  room_id=COALESCE(NULLIF(EXCLUDED.room_id,''), bot_reply_audits.room_id),
  trigger_message_id=COALESCE(EXCLUDED.trigger_message_id, bot_reply_audits.trigger_message_id),
  trigger_sender_id=COALESCE(EXCLUDED.trigger_sender_id, bot_reply_audits.trigger_sender_id),
  trigger_sender_name=COALESCE(NULLIF(EXCLUDED.trigger_sender_name,''), bot_reply_audits.trigger_sender_name),
  bot_role=COALESCE(NULLIF(EXCLUDED.bot_role,''), bot_reply_audits.bot_role),
  firepower_level=COALESCE(NULLIF(EXCLUDED.firepower_level,''), bot_reply_audits.firepower_level),
  trigger_reason=COALESCE(NULLIF(EXCLUDED.trigger_reason,''), bot_reply_audits.trigger_reason),
  trigger_type=COALESCE(NULLIF(EXCLUDED.trigger_type,''), bot_reply_audits.trigger_type),
  force_reply=EXCLUDED.force_reply,
  hype_score=EXCLUDED.hype_score,
  absurdity_score=EXCLUDED.absurdity_score,
  risk_score=EXCLUDED.risk_score,
  reply_mode=COALESCE(NULLIF(EXCLUDED.reply_mode,''), bot_reply_audits.reply_mode),
  llm_enabled=EXCLUDED.llm_enabled,
  provider_initialized=EXCLUDED.provider_initialized,
  api_key_present=EXCLUDED.api_key_present,
  displayable_content_found=EXCLUDED.displayable_content_found,
  reasoning_only_response=EXCLUDED.reasoning_only_response,
  updated_at=NOW()`,
		strings.TrimSpace(evt.TraceID),
		strings.TrimSpace(evt.RoomID),
		nullableMessageID(evt.MessageID),
		nullableUUID(evt.SenderID),
		strings.TrimSpace(evt.SenderName),
		orDefault(evt.BotRole, "judge"),
		orDefault(evt.FirepowerLevel, "medium"),
		nullableString(evt.TriggerReason),
		triggerType,
		evt.ForceReply,
		evt.HypeScore,
		evt.AbsurdityScore,
		evt.RiskScore,
		nullableString(evt.ReplyMode),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateTriggerSkip(ctx context.Context, evt bot.DebugEvent) error {
	triggerType := auditTriggerType(evt)
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET trigger_reason=COALESCE(NULLIF($2,''), trigger_reason),
    trigger_type=COALESCE(NULLIF($3,''), trigger_type),
    force_reply=$4,
    hype_score=$5,
    absurdity_score=$6,
    risk_score=$7,
    reply_mode=COALESCE(NULLIF($8,''), reply_mode),
    llm_enabled=$9,
    provider_initialized=$10,
    api_key_present=$11,
    displayable_content_found=$12,
    reasoning_only_response=$13,
    fallback_reason=COALESCE(NULLIF($14,''), fallback_reason),
    error_message=COALESCE(NULLIF($15,''), error_message),
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.TriggerReason),
		triggerType,
		evt.ForceReply,
		evt.HypeScore,
		evt.AbsurdityScore,
		evt.RiskScore,
		nullableString(evt.ReplyMode),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
		nullableString(evt.SkipReason),
		nullableString(skipErrorMessage(evt.SkipReason)),
	)
	return err
}

func (s *BotAuditService) ListRoomAudits(ctx context.Context, roomID string, in BotAuditListFilter) ([]models.BotReplyAudit, int, error) {
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	}
	if in.PageSize > 100 {
		in.PageSize = 100
	}
	offset := (in.Page - 1) * in.PageSize

	conditions := []string{"a.room_id = $1"}
	args := []any{roomID}
	nextArg := 2
	if v := strings.TrimSpace(in.ReplySource); v != "" {
		conditions = append(conditions, fmt.Sprintf("a.reply_source = $%d", nextArg))
		args = append(args, v)
		nextArg++
	}
	if v := strings.TrimSpace(in.BotRole); v != "" {
		conditions = append(conditions, fmt.Sprintf("a.bot_role = $%d", nextArg))
		args = append(args, v)
		nextArg++
	}
	where := strings.Join(conditions, " AND ")

	countSQL := fmt.Sprintf(`SELECT COUNT(1) FROM bot_reply_audits a WHERE %s`, where)
	var total int
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, in.PageSize, offset)
	querySQL := fmt.Sprintf(`
SELECT
  a.id, a.trace_id, a.room_id, a.trigger_message_id, a.trigger_sender_id, a.trigger_sender_name,
  a.reply_message_id, a.reply_source, a.bot_role, a.firepower_level, a.trigger_reason, a.trigger_type, a.force_reply,
  a.hype_score, a.absurdity_score, a.risk_score, a.reply_mode, a.provider, a.model, a.request_prompt_excerpt, a.response_text, a.fallback_reason,
  a.http_status, a.latency_ms, a.prompt_tokens, a.completion_tokens, a.total_tokens, a.error_message,
  a.llm_enabled, a.provider_initialized, a.api_key_present, a.displayable_content_found, a.reasoning_only_response,
  a.created_at, a.updated_at,
  tm.id, tm.nickname, tm.sender_type, tm.content, tm.created_at,
  rm.id, rm.nickname, rm.sender_type, rm.content, rm.created_at
FROM bot_reply_audits a
LEFT JOIN messages tm ON tm.id = a.trigger_message_id
LEFT JOIN messages rm ON rm.id = a.reply_message_id
WHERE %s
ORDER BY a.created_at DESC
LIMIT $%d OFFSET $%d`,
		where, nextArg, nextArg+1)

	rows, err := s.db.QueryContext(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]models.BotReplyAudit, 0, in.PageSize)
	for rows.Next() {
		item, err := scanAuditRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *BotAuditService) GetByMessageID(ctx context.Context, messageID int64) (*models.BotReplyAudit, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
  a.id, a.trace_id, a.room_id, a.trigger_message_id, a.trigger_sender_id, a.trigger_sender_name,
  a.reply_message_id, a.reply_source, a.bot_role, a.firepower_level, a.trigger_reason, a.trigger_type, a.force_reply,
  a.hype_score, a.absurdity_score, a.risk_score, a.reply_mode, a.provider, a.model, a.request_prompt_excerpt, a.response_text, a.fallback_reason,
  a.http_status, a.latency_ms, a.prompt_tokens, a.completion_tokens, a.total_tokens, a.error_message,
  a.llm_enabled, a.provider_initialized, a.api_key_present, a.displayable_content_found, a.reasoning_only_response,
  a.created_at, a.updated_at,
  tm.id, tm.nickname, tm.sender_type, tm.content, tm.created_at,
  rm.id, rm.nickname, rm.sender_type, rm.content, rm.created_at
FROM bot_reply_audits a
LEFT JOIN messages tm ON tm.id = a.trigger_message_id
LEFT JOIN messages rm ON rm.id = a.reply_message_id
WHERE a.trigger_message_id=$1 OR a.reply_message_id=$1
ORDER BY a.created_at DESC
LIMIT 1`, messageID)
	item, err := scanAuditRow(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *BotAuditService) GetByID(ctx context.Context, id int64) (*models.BotReplyAudit, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
  a.id, a.trace_id, a.room_id, a.trigger_message_id, a.trigger_sender_id, a.trigger_sender_name,
  a.reply_message_id, a.reply_source, a.bot_role, a.firepower_level, a.trigger_reason, a.trigger_type, a.force_reply,
  a.hype_score, a.absurdity_score, a.risk_score, a.reply_mode, a.provider, a.model, a.request_prompt_excerpt, a.response_text, a.fallback_reason,
  a.http_status, a.latency_ms, a.prompt_tokens, a.completion_tokens, a.total_tokens, a.error_message,
  a.llm_enabled, a.provider_initialized, a.api_key_present, a.displayable_content_found, a.reasoning_only_response,
  a.created_at, a.updated_at,
  tm.id, tm.nickname, tm.sender_type, tm.content, tm.created_at,
  rm.id, rm.nickname, rm.sender_type, rm.content, rm.created_at
FROM bot_reply_audits a
LEFT JOIN messages tm ON tm.id = a.trigger_message_id
LEFT JOIN messages rm ON rm.id = a.reply_message_id
WHERE a.id=$1
LIMIT 1`, id)
	item, err := scanAuditRow(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *BotAuditService) upsertTriggerHit(ctx context.Context, evt bot.DebugEvent) error {
	triggerType := auditTriggerType(evt)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO bot_reply_audits (
  trace_id, room_id, trigger_message_id, trigger_sender_id, trigger_sender_name,
  reply_source, bot_role, firepower_level, trigger_reason, trigger_type, force_reply, hype_score,
  absurdity_score, risk_score, reply_mode,
  llm_enabled, provider_initialized, api_key_present, displayable_content_found, reasoning_only_response, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,'template',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NOW(),NOW())
ON CONFLICT (trace_id) DO UPDATE SET
  room_id=EXCLUDED.room_id,
  trigger_message_id=COALESCE(EXCLUDED.trigger_message_id, bot_reply_audits.trigger_message_id),
  trigger_sender_id=COALESCE(EXCLUDED.trigger_sender_id, bot_reply_audits.trigger_sender_id),
  trigger_sender_name=COALESCE(NULLIF(EXCLUDED.trigger_sender_name,''), bot_reply_audits.trigger_sender_name),
  bot_role=COALESCE(NULLIF(EXCLUDED.bot_role,''), bot_reply_audits.bot_role),
  firepower_level=COALESCE(NULLIF(EXCLUDED.firepower_level,''), bot_reply_audits.firepower_level),
  trigger_reason=COALESCE(NULLIF(EXCLUDED.trigger_reason,''), bot_reply_audits.trigger_reason),
  trigger_type=COALESCE(NULLIF(EXCLUDED.trigger_type,''), bot_reply_audits.trigger_type),
  force_reply=EXCLUDED.force_reply,
  hype_score=EXCLUDED.hype_score,
  absurdity_score=EXCLUDED.absurdity_score,
  risk_score=EXCLUDED.risk_score,
  reply_mode=COALESCE(NULLIF(EXCLUDED.reply_mode,''), bot_reply_audits.reply_mode),
  llm_enabled=EXCLUDED.llm_enabled,
  provider_initialized=EXCLUDED.provider_initialized,
  api_key_present=EXCLUDED.api_key_present,
  displayable_content_found=EXCLUDED.displayable_content_found,
  reasoning_only_response=EXCLUDED.reasoning_only_response,
  updated_at=NOW()`,
		strings.TrimSpace(evt.TraceID),
		strings.TrimSpace(evt.RoomID),
		nullableMessageID(evt.MessageID),
		nullableUUID(evt.SenderID),
		strings.TrimSpace(evt.SenderName),
		orDefault(evt.BotRole, "judge"),
		orDefault(evt.FirepowerLevel, "medium"),
		nullableString(evt.TriggerReason),
		triggerType,
		evt.ForceReply,
		evt.HypeScore,
		evt.AbsurdityScore,
		evt.RiskScore,
		nullableString(evt.ReplyMode),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateLLMPrepare(ctx context.Context, evt bot.DebugEvent) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET provider=COALESCE(NULLIF($2,''), provider),
    model=COALESCE(NULLIF($3,''), model),
    request_prompt_excerpt=COALESCE(NULLIF($4,''), request_prompt_excerpt),
    llm_enabled=$5,
    provider_initialized=$6,
    api_key_present=$7,
    displayable_content_found=$8,
    reasoning_only_response=$9,
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.Provider),
		nullableString(evt.Model),
		nullableString(cutText(evt.RequestPromptExcerpt, 1000)),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateLLMSuccess(ctx context.Context, evt bot.DebugEvent) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET reply_source='llm',
    provider=COALESCE(NULLIF($2,''), provider),
    model=COALESCE(NULLIF($3,''), model),
    http_status=COALESCE(NULLIF($4,0), http_status),
    latency_ms=COALESCE(NULLIF($5,0), latency_ms),
    prompt_tokens=COALESCE(NULLIF($6,0), prompt_tokens),
    completion_tokens=COALESCE(NULLIF($7,0), completion_tokens),
    total_tokens=COALESCE(NULLIF($8,0), total_tokens),
    llm_enabled=$9,
    provider_initialized=$10,
    api_key_present=$11,
    displayable_content_found=$12,
    reasoning_only_response=$13,
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.Provider),
		nullableString(evt.Model),
		evt.HTTPStatus,
		evt.LatencyMs,
		evt.PromptTokens,
		evt.CompletionTokens,
		evt.TotalTokens,
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateLLMError(ctx context.Context, evt bot.DebugEvent) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET provider=COALESCE(NULLIF($2,''), provider),
    model=COALESCE(NULLIF($3,''), model),
    request_prompt_excerpt=COALESCE(NULLIF($4,''), request_prompt_excerpt),
    fallback_reason=COALESCE(NULLIF($5,''), fallback_reason),
    http_status=COALESCE(NULLIF($6,0), http_status),
    latency_ms=COALESCE(NULLIF($7,0), latency_ms),
    prompt_tokens=COALESCE(NULLIF($8,0), prompt_tokens),
    completion_tokens=COALESCE(NULLIF($9,0), completion_tokens),
    total_tokens=COALESCE(NULLIF($10,0), total_tokens),
    error_message=COALESCE(NULLIF($11,''), error_message),
    llm_enabled=$12,
    provider_initialized=$13,
    api_key_present=$14,
    displayable_content_found=$15,
    reasoning_only_response=$16,
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.Provider),
		nullableString(evt.Model),
		nullableString(cutText(evt.RequestPromptExcerpt, 1000)),
		nullableString(evt.FallbackReason),
		evt.HTTPStatus,
		evt.LatencyMs,
		evt.PromptTokens,
		evt.CompletionTokens,
		evt.TotalTokens,
		nullableString(cutText(evt.ErrorMessage, 1000)),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateLLMFallback(ctx context.Context, evt bot.DebugEvent) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET reply_source='template',
    fallback_reason=COALESCE(NULLIF($2,''), fallback_reason),
    error_message=COALESCE(NULLIF($3,''), error_message),
    llm_enabled=$4,
    provider_initialized=$5,
    api_key_present=$6,
    displayable_content_found=$7,
    reasoning_only_response=$8,
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.FallbackReason),
		nullableString(cutText(evt.ErrorMessage, 1000)),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

func (s *BotAuditService) updateReplyDone(ctx context.Context, evt bot.DebugEvent) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bot_reply_audits
SET reply_source=COALESCE(NULLIF($2,''), reply_source),
    provider=COALESCE(NULLIF($3,''), provider),
    model=COALESCE(NULLIF($4,''), model),
    fallback_reason=COALESCE(NULLIF($5,''), fallback_reason),
    trigger_reason=COALESCE(NULLIF($6,''), trigger_reason),
    trigger_type=COALESCE(NULLIF($7,''), trigger_type),
    force_reply=$8,
    hype_score=$9,
    absurdity_score=$10,
    risk_score=$11,
    reply_mode=COALESCE(NULLIF($12,''), reply_mode),
    response_text=COALESCE(NULLIF($13,''), response_text),
    request_prompt_excerpt=COALESCE(NULLIF($14,''), request_prompt_excerpt),
    http_status=COALESCE(NULLIF($15,0), http_status),
    latency_ms=COALESCE(NULLIF($16,0), latency_ms),
    prompt_tokens=COALESCE(NULLIF($17,0), prompt_tokens),
    completion_tokens=COALESCE(NULLIF($18,0), completion_tokens),
    total_tokens=COALESCE(NULLIF($19,0), total_tokens),
    error_message=COALESCE(NULLIF($20,''), error_message),
    llm_enabled=$21,
    provider_initialized=$22,
    api_key_present=$23,
    displayable_content_found=$24,
    reasoning_only_response=$25,
    updated_at=NOW()
WHERE trace_id=$1`,
		strings.TrimSpace(evt.TraceID),
		nullableString(evt.ReplySource),
		nullableString(evt.Provider),
		nullableString(evt.Model),
		nullableString(evt.FallbackReason),
		nullableString(evt.TriggerReason),
		nullableString(auditTriggerType(evt)),
		evt.ForceReply,
		evt.HypeScore,
		evt.AbsurdityScore,
		evt.RiskScore,
		nullableString(evt.ReplyMode),
		nullableString(cutText(evt.ResponseText, 1000)),
		nullableString(cutText(evt.RequestPromptExcerpt, 1000)),
		evt.HTTPStatus,
		evt.LatencyMs,
		evt.PromptTokens,
		evt.CompletionTokens,
		evt.TotalTokens,
		nullableString(cutText(evt.ErrorMessage, 1000)),
		evt.LLMEnabled,
		evt.ProviderInitialized,
		evt.APIKeyPresent,
		evt.DisplayableFound,
		evt.ReasoningOnly,
	)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAuditRow(scanner rowScanner) (models.BotReplyAudit, error) {
	var item models.BotReplyAudit
	var triggerMsgID sql.NullInt64
	var triggerSenderID sql.NullString
	var replyMsgID sql.NullInt64
	var triggerReason sql.NullString
	var triggerType sql.NullString
	var replyMode sql.NullString
	var provider sql.NullString
	var model sql.NullString
	var reqPrompt sql.NullString
	var responseText sql.NullString
	var fallback sql.NullString
	var httpStatus sql.NullInt64
	var latencyMs sql.NullInt64
	var promptTokens sql.NullInt64
	var completionTokens sql.NullInt64
	var totalTokens sql.NullInt64
	var errMessage sql.NullString

	var tmID sql.NullInt64
	var tmNick sql.NullString
	var tmSenderType sql.NullString
	var tmContent sql.NullString
	var tmCreatedAt sql.NullTime

	var rmID sql.NullInt64
	var rmNick sql.NullString
	var rmSenderType sql.NullString
	var rmContent sql.NullString
	var rmCreatedAt sql.NullTime

	if err := scanner.Scan(
		&item.ID, &item.TraceID, &item.RoomID, &triggerMsgID, &triggerSenderID, &item.TriggerSenderName,
		&replyMsgID, &item.ReplySource, &item.BotRole, &item.FirepowerLevel, &triggerReason, &triggerType, &item.ForceReply,
		&item.HypeScore, &item.AbsurdityScore, &item.RiskScore, &replyMode, &provider, &model, &reqPrompt, &responseText, &fallback,
		&httpStatus, &latencyMs, &promptTokens, &completionTokens, &totalTokens, &errMessage,
		&item.LLMEnabled, &item.ProviderInitialized, &item.APIKeyPresent, &item.DisplayableFound, &item.ReasoningOnly,
		&item.CreatedAt, &item.UpdatedAt,
		&tmID, &tmNick, &tmSenderType, &tmContent, &tmCreatedAt,
		&rmID, &rmNick, &rmSenderType, &rmContent, &rmCreatedAt,
	); err != nil {
		return models.BotReplyAudit{}, err
	}

	item.TriggerMessageID = nullInt64Ptr(triggerMsgID)
	item.TriggerSenderID = nullStringPtr(triggerSenderID)
	item.ReplyMessageID = nullInt64Ptr(replyMsgID)
	item.TriggerReason = nullStringPtr(triggerReason)
	item.TriggerType = nullStringPtr(triggerType)
	item.ReplyMode = nullStringPtr(replyMode)
	item.Provider = nullStringPtr(provider)
	item.Model = nullStringPtr(model)
	item.RequestPrompt = nullStringPtr(reqPrompt)
	item.ResponseText = nullStringPtr(responseText)
	item.FallbackReason = nullStringPtr(fallback)
	item.HTTPStatus = nullIntPtr(httpStatus)
	item.LatencyMs = nullIntPtr(latencyMs)
	item.PromptTokens = nullIntPtr(promptTokens)
	item.CompletionTokens = nullIntPtr(completionTokens)
	item.TotalTokens = nullIntPtr(totalTokens)
	item.ErrorMessage = nullStringPtr(errMessage)

	if tmID.Valid {
		item.TriggerMessage = &models.AuditMessageBrief{
			ID:         tmID.Int64,
			Nickname:   tmNick.String,
			SenderType: tmSenderType.String,
			Content:    tmContent.String,
		}
		if tmCreatedAt.Valid {
			item.TriggerMessage.CreatedAt = tmCreatedAt.Time
		}
	}
	if rmID.Valid {
		item.ReplyMessage = &models.AuditMessageBrief{
			ID:         rmID.Int64,
			Nickname:   rmNick.String,
			SenderType: rmSenderType.String,
			Content:    rmContent.String,
		}
		if rmCreatedAt.Valid {
			item.ReplyMessage.CreatedAt = rmCreatedAt.Time
		}
	}
	return item, nil
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func nullableUUID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if _, err := uuid.Parse(v); err != nil {
		return nil
	}
	return v
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	out := v.String
	return &out
}

func nullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	out := int(v.Int64)
	return &out
}

func nullableMessageID(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func cutText(v string, max int) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if max <= 0 {
		max = 1000
	}
	rs := []rune(v)
	if len(rs) > max {
		return string(rs[:max]) + "..."
	}
	return v
}

func orDefault(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func auditTriggerType(evt bot.DebugEvent) string {
	v := strings.TrimSpace(evt.TriggerType)
	if v != "" {
		return v
	}
	if strings.EqualFold(strings.TrimSpace(evt.TriggerReason), "manual_test") {
		return "manual_test"
	}
	if strings.EqualFold(strings.TrimSpace(evt.TriggerReason), "cold_start") {
		return "cold_start"
	}
	if evt.ReplyToIsBot || strings.EqualFold(strings.TrimSpace(evt.TriggerReason), "quote_bot") {
		return "quote_bot"
	}
	if evt.MessageID <= 0 || strings.TrimSpace(evt.SenderID) == "" {
		return "system_event"
	}
	return "user_message"
}

func skipErrorMessage(skipReason string) string {
	skipReason = strings.TrimSpace(skipReason)
	if skipReason == "" {
		return ""
	}
	return "skip:" + skipReason
}
