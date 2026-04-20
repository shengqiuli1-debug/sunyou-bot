import { http } from './http'
import type { BotReplyAudit, BotRole, FireLevel, Identity, Room, RoomMember, RoomReport, User } from '@/types'

export const api = {
  createGuest: (nickname: string) =>
    http.post<{ user: User; token: string }>('/users/guest', { nickname }),
  me: () => http.get<{ user: User }>('/users/me'),
  points: () => http.get<{ points: number }>('/points'),
  recharge: (amount: number) => http.post<{ balance: number }>('/points/recharge', { amount, channel: 'mock_wechat' }),
  pointLedger: () => http.get<{ items: Array<Record<string, unknown>> }>('/points/ledger'),
  roomRecords: () => http.get<{ items: Array<Record<string, unknown>> }>('/rooms/records'),

  createRoom: (payload: {
    durationMinutes: number
    botRole: BotRole
    fireLevel: FireLevel
    generateReport: boolean
  }) => http.post<{ room: Room; freeTrialUsed: boolean; remainingPointBalance: number }>('/rooms', payload),

  getRoomByShare: (code: string) => http.get<{ room: Room }>(`/rooms/by-share/${code}`),
  getRoom: (id: string) => http.get<{ room: Room; members: RoomMember[]; self: RoomMember | null }>(`/rooms/${id}`),
  joinRoom: (id: string, identity: Identity, confirmTarget = false) =>
    http.post<{ member: RoomMember }>(`/rooms/${id}/join`, { identity, confirmTarget }),
  updateIdentity: (id: string, identity: Identity, confirmTarget = false) =>
    http.post<{ member: RoomMember }>(`/rooms/${id}/identity`, { identity, confirmTarget }),
  roomControl: (id: string, action: string, value = '') =>
    http.post<{ room: Room }>(`/rooms/${id}/control`, { action, value }),
  endRoom: (id: string) => http.post<{ report: RoomReport }>(`/rooms/${id}/end`),
  roomMessages: (id: string) => http.get<{ items: Array<Record<string, unknown>> }>(`/rooms/${id}/messages`),
  roomReport: (id: string) => http.get<{ report: RoomReport }>(`/rooms/${id}/report`),
  reportAbuse: (id: string, message: string, targetUserId = '') =>
    http.post<{ ok: boolean }>(`/rooms/${id}/report-abuse`, { message, targetUserId }),
  debugLLMStatus: () =>
    http.get<{
      llm_enabled: boolean
      provider_initialized: boolean
      provider_name: string
      base_url: string
      model: string
      api_key_present: boolean
      timeout_seconds: number
      debug_force_llm: boolean
      last_llm_success_at?: string
      last_llm_error?: string
      last_fallback_reason?: string
    }>('/debug/llm-status'),
  debugLLMTest: (id: string) => http.post<{ message: Record<string, unknown> | null }>(`/rooms/${id}/debug/llm-test`),
  debugRoomBotAudits: (roomId: string, query?: { page?: number; pageSize?: number; replySource?: string; botRole?: string }) =>
    http.get<{ items: BotReplyAudit[]; total: number; page: number; pageSize: number }>(
      `/debug/rooms/${roomId}/bot-audits?${new URLSearchParams({
        page: String(query?.page || 1),
        pageSize: String(query?.pageSize || 20),
        replySource: query?.replySource || '',
        botRole: query?.botRole || ''
      }).toString()}`
    ),
  debugMessageBotAudit: (messageId: number) =>
    http.get<{ item: BotReplyAudit }>(`/debug/messages/${messageId}/bot-audit`),
  debugBotAuditById: (id: number) =>
    http.get<{ item: BotReplyAudit }>(`/debug/bot-audits/${id}`)
}
