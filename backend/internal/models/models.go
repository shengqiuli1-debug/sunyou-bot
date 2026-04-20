package models

import "time"

type Identity string

type BotRole string

type FireLevel string

type RoomStatus string

const (
	IdentityNormal Identity = "normal"
	IdentityTarget Identity = "target"
	IdentityImmune Identity = "immune"
)

const (
	RoleJudge    BotRole = "judge"
	RoleNpc      BotRole = "npc"
	RoleNarrator BotRole = "narrator"
)

const (
	FireLow    FireLevel = "low"
	FireMedium FireLevel = "medium"
	FireHigh   FireLevel = "high"
)

const (
	RoomActive  RoomStatus = "active"
	RoomExpired RoomStatus = "expired"
	RoomEnded   RoomStatus = "ended"
)

type User struct {
	ID             string    `json:"id"`
	Token          string    `json:"-"`
	Nickname       string    `json:"nickname"`
	Points         int       `json:"points"`
	FreeTrialRooms int       `json:"freeTrialRooms"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Room struct {
	ID              string     `json:"id"`
	ShareCode       string     `json:"shareCode"`
	OwnerUserID     string     `json:"ownerUserId"`
	BotRole         BotRole    `json:"botRole"`
	FireLevel       FireLevel  `json:"fireLevel"`
	GenerateReport  bool       `json:"generateReport"`
	DurationMinutes int        `json:"durationMinutes"`
	CostPoints      int        `json:"costPoints"`
	Status          RoomStatus `json:"status"`
	MuteBotUntil    *time.Time `json:"muteBotUntil,omitempty"`
	EndAt           time.Time  `json:"endAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	EndedAt         *time.Time `json:"endedAt,omitempty"`
}

type RoomMember struct {
	RoomID   string    `json:"roomId"`
	UserID   string    `json:"userId"`
	Nickname string    `json:"nickname"`
	Identity Identity  `json:"identity"`
	IsOwner  bool      `json:"isOwner"`
	JoinedAt time.Time `json:"joinedAt"`
}

type ChatMessage struct {
	ID                    int64     `json:"id"`
	RoomID                string    `json:"roomId"`
	UserID                *string   `json:"userId,omitempty"`
	Nickname              string    `json:"nickname"`
	SenderType            string    `json:"senderType"`
	Content               string    `json:"content"`
	ClientMsgID           *string   `json:"clientMsgId,omitempty"`
	ReplyToMessageID      *int64    `json:"replyToMessageId,omitempty"`
	ReplyToSenderID       *string   `json:"replyToSenderId,omitempty"`
	ReplyToSenderName     *string   `json:"replyToSenderName,omitempty"`
	ReplyToContentPreview *string   `json:"replyToPreview,omitempty"`
	IsBotMessage          bool      `json:"isBotMessage"`
	BotRole               *BotRole  `json:"botRole,omitempty"`
	ReplySource           *string   `json:"reply_source,omitempty"`
	LLMModel              *string   `json:"llm_model,omitempty"`
	FallbackReason        *string   `json:"fallback_reason,omitempty"`
	TraceID               *string   `json:"trace_id,omitempty"`
	ForceReply            bool      `json:"force_reply"`
	TriggerReason         *string   `json:"trigger_reason,omitempty"`
	HypeScore             *int      `json:"hype_score,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
}

type RoomReport struct {
	RoomID          string     `json:"roomId"`
	HardmouthLabel  string     `json:"hardmouthLabel"`
	BestAssistLabel string     `json:"bestAssistLabel"`
	QuietMomentSecs int        `json:"quietMomentSecs"`
	QuietMomentAt   *time.Time `json:"quietMomentAt,omitempty"`
	SavefaceLabel   string     `json:"savefaceLabel"`
	BotQuote        string     `json:"botQuote"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type AuditMessageBrief struct {
	ID         int64     `json:"id"`
	Nickname   string    `json:"nickname"`
	SenderType string    `json:"sender_type"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type BotReplyAudit struct {
	ID                  int64              `json:"id"`
	TraceID             string             `json:"trace_id"`
	RoomID              string             `json:"room_id"`
	TriggerMessageID    *int64             `json:"trigger_message_id,omitempty"`
	TriggerSenderID     *string            `json:"trigger_sender_id,omitempty"`
	TriggerSenderName   string             `json:"trigger_sender_name"`
	ReplyMessageID      *int64             `json:"reply_message_id,omitempty"`
	ReplySource         string             `json:"reply_source"`
	BotRole             string             `json:"bot_role"`
	FirepowerLevel      string             `json:"firepower_level"`
	TriggerReason       *string            `json:"trigger_reason,omitempty"`
	TriggerType         *string            `json:"trigger_type,omitempty"`
	ForceReply          bool               `json:"force_reply"`
	HypeScore           int                `json:"hype_score"`
	AbsurdityScore      int                `json:"absurdity_score"`
	RiskScore           int                `json:"risk_score"`
	ReplyMode           *string            `json:"reply_mode,omitempty"`
	LLMEnabled          bool               `json:"llm_enabled"`
	ProviderInitialized bool               `json:"provider_initialized"`
	APIKeyPresent       bool               `json:"api_key_present"`
	DisplayableFound    bool               `json:"displayable_content_found"`
	ReasoningOnly       bool               `json:"reasoning_only_response"`
	Provider            *string            `json:"provider,omitempty"`
	Model               *string            `json:"model,omitempty"`
	RequestPrompt       *string            `json:"request_prompt_excerpt,omitempty"`
	ResponseText        *string            `json:"response_text,omitempty"`
	FallbackReason      *string            `json:"fallback_reason,omitempty"`
	HTTPStatus          *int               `json:"http_status,omitempty"`
	LatencyMs           *int               `json:"latency_ms,omitempty"`
	PromptTokens        *int               `json:"prompt_tokens,omitempty"`
	CompletionTokens    *int               `json:"completion_tokens,omitempty"`
	TotalTokens         *int               `json:"total_tokens,omitempty"`
	ErrorMessage        *string            `json:"error_message,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	TriggerMessage      *AuditMessageBrief `json:"trigger_message,omitempty"`
	ReplyMessage        *AuditMessageBrief `json:"reply_message,omitempty"`
}
