export type Identity = 'normal' | 'target' | 'immune'
export type BotRole = 'judge' | 'npc' | 'narrator'
export type FireLevel = 'low' | 'medium' | 'high'
export type RoomStatus = 'active' | 'expired' | 'ended'

export interface User {
  id: string
  nickname: string
  points: number
  freeTrialRooms: number
}

export interface Room {
  id: string
  shareCode: string
  ownerUserId: string
  botRole: BotRole
  fireLevel: FireLevel
  generateReport: boolean
  durationMinutes: number
  costPoints: number
  status: RoomStatus
  endAt: string
  createdAt: string
}

export interface RoomMember {
  roomId: string
  userId: string
  nickname: string
  identity: Identity
  isOwner: boolean
  joinedAt: string
}

export interface ChatMessage {
  id: number
  roomId: string
  userId?: string
  nickname: string
  senderType: 'user' | 'bot' | 'system'
  content: string
  clientMsgId?: string
  pending?: boolean
  replyToMessageId?: number
  replyToSenderId?: string
  replyToSenderName?: string
  replyToPreview?: string
  isBotMessage?: boolean
  botRole?: BotRole
  replySource?: 'llm' | 'template'
  llmModel?: string
  fallbackReason?: string
  traceId?: string
  forceReply?: boolean
  triggerReason?: string
  hypeScore?: number
  createdAt: string
}

export interface RoomReport {
  roomId: string
  hardmouthLabel: string
  bestAssistLabel: string
  quietMomentSecs: number
  quietMomentAt?: string
  savefaceLabel: string
  botQuote: string
  createdAt: string
}

export interface BotDebugEvent {
  time: string
  event: string
  roomId?: string
  messageId?: number
  senderId?: string
  senderName?: string
  content?: string
  replyToMessageId?: number
  replyToIsBot?: boolean
  botRole?: string
  triggerReason?: string
  triggerType?: string
  skipReason?: string
  botReplySkipped?: boolean
  forceReply?: boolean
  hypeScore?: number
  absurdityScore?: number
  riskScore?: number
  replyMode?: string
  modelPool?: string
  candidateModels?: string
  skippedModels?: string
  triedModels?: string
  selectedModel?: string
  modelFailures?: string
  lastErrorType?: string
  circuitOpenUntil?: string
  llmAttempted?: boolean
  replySource?: string
  fallbackReason?: string
  usedLLM?: boolean
  llmEnabled?: boolean
  providerInitialized?: boolean
  traceId?: string
  model?: string
  provider?: string
  baseUrl?: string
  apiKeyPresent?: boolean
  httpStatus?: number
  latencyMs?: number
  promptTokens?: number
  completionTokens?: number
  totalTokens?: number
  requestPromptExcerpt?: string
  responseText?: string
  errorMessage?: string
  displayableContentFound?: boolean
  reasoningOnlyResponse?: boolean
}

export interface AuditMessageBrief {
  id: number
  nickname: string
  senderType: 'user' | 'bot' | 'system'
  content: string
  createdAt: string
}

export interface BotReplyAudit {
  id: number
  traceId: string
  roomId: string
  triggerMessageId?: number
  triggerSenderId?: string
  triggerSenderName: string
  replyMessageId?: number
  replySource: 'llm' | 'template'
  botRole: string
  firepowerLevel: string
  triggerReason?: string
  triggerType?: string
  forceReply: boolean
  hypeScore: number
  absurdityScore?: number
  riskScore?: number
  replyMode?: string
  llmEnabled?: boolean
  providerInitialized?: boolean
  apiKeyPresent?: boolean
  provider?: string
  model?: string
  requestPromptExcerpt?: string
  responseText?: string
  fallbackReason?: string
  httpStatus?: number
  latencyMs?: number
  promptTokens?: number
  completionTokens?: number
  totalTokens?: number
  errorMessage?: string
  displayableContentFound?: boolean
  reasoningOnlyResponse?: boolean
  createdAt: string
  updatedAt: string
  triggerMessage?: AuditMessageBrief
  replyMessage?: AuditMessageBrief
}
