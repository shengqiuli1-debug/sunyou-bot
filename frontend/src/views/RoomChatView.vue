<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { RoomSocket } from '@/api/ws'
import { useUserStore } from '@/stores/user'
import { useRoomStore } from '@/stores/room'
import type { BotDebugEvent, BotReplyAudit, BotRole, ChatMessage, Identity, Room, RoomMember, RoomReport } from '@/types'
import MessageBubble from '@/components/MessageBubble.vue'
import RoomHeader from '@/components/RoomHeader.vue'
import RoomMoreMenu from '@/components/RoomMoreMenu.vue'
import InviteSheet from '@/components/InviteSheet.vue'
import MemberListSheet from '@/components/MemberListSheet.vue'
import MemberActionSheet from '@/components/MemberActionSheet.vue'
import HostControlSheet from '@/components/HostControlSheet.vue'
import ChatComposer from '@/components/ChatComposer.vue'
import BotDebugDrawer from '@/components/BotDebugDrawer.vue'
import ThemePickerSheet from '@/components/ThemePickerSheet.vue'
import BotProfileCard from '@/components/BotProfileCard.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const roomStore = useRoomStore()

const roomId = String(route.params.id)
const input = ref('')
const loading = ref(false)
const error = ref('')
const socket = ref<RoomSocket | null>(null)
const listRef = ref<HTMLElement | null>(null)
const replyingTo = ref<ChatMessage | null>(null)
let localPendingSeq = -1

const showMoreMenu = ref(false)
const showInviteSheet = ref(false)
const showMembersSheet = ref(false)
const showMemberActionSheet = ref(false)
const showHostToolSheet = ref(false)
const showGuideSheet = ref(false)
const showDebugDrawer = ref(false)
const showThemeSheet = ref(false)
const showBotProfile = ref(false)
const selectedMember = ref<RoomMember | null>(null)
const activeBotProfile = ref<{
  role: BotRole
  name: string
  bio: string
  identity: string
  duty: string
} | null>(null)
const botAudits = ref<BotReplyAudit[]>([])
const auditLoading = ref(false)
const auditError = ref('')

const notice = ref<{ text: string; type: 'info' | 'ok' | 'error' } | null>(null)
let noticeTimer: ReturnType<typeof setTimeout> | null = null

const room = computed(() => roomStore.room)
const self = computed(() => roomStore.self)
const members = computed(() => roomStore.members)
const isOwner = computed(() => !!roomStore.self?.isOwner)
const showDebugUI = import.meta.env.DEV || location.hostname === 'localhost' || location.hostname === '127.0.0.1'

function pushNotice(text: string, type: 'info' | 'ok' | 'error' = 'info') {
  notice.value = { text, type }
  if (noticeTimer) clearTimeout(noticeTimer)
  noticeTimer = setTimeout(() => {
    notice.value = null
  }, 1800)
}

function mapMessage(raw: any): ChatMessage {
  return {
    id: Number(raw.id || Date.now()),
    roomId: raw.roomId || roomId,
    userId: raw.userId || undefined,
    nickname: raw.nickname || '系统',
    senderType: raw.senderType || 'system',
    content: raw.content || '',
    clientMsgId: raw.clientMsgId || raw.client_msg_id || undefined,
    pending: !!raw.pending,
    replyToMessageId: raw.replyToMessageId ? Number(raw.replyToMessageId) : undefined,
    replyToSenderId: raw.replyToSenderId || undefined,
    replyToSenderName: raw.replyToSenderName || undefined,
    replyToPreview: raw.replyToPreview || undefined,
    isBotMessage: raw.isBotMessage === undefined ? raw.senderType === 'bot' : !!raw.isBotMessage,
    botRole: raw.botRole || raw.bot_role || undefined,
    replySource: raw.reply_source || raw.replySource || undefined,
    llmModel: raw.llm_model || raw.llmModel || undefined,
    fallbackReason: raw.fallback_reason || raw.fallbackReason || undefined,
    traceId: raw.trace_id || raw.traceId || undefined,
    forceReply: raw.force_reply === undefined ? raw.forceReply : !!raw.force_reply,
    triggerReason: raw.trigger_reason || raw.triggerReason || undefined,
    hypeScore:
      raw.hype_score === undefined || raw.hype_score === null
        ? raw.hypeScore
        : Number(raw.hype_score),
    createdAt: raw.createdAt || raw.created_at || new Date().toISOString()
  }
}

function mergeIncomingMessage(msg: ChatMessage) {
  const existed = roomStore.messages.findIndex((item) => item.id === msg.id)
  if (existed >= 0) {
    roomStore.messages[existed] = { ...msg, pending: false }
    return
  }
  if (msg.clientMsgId) {
    const pendingIdx = roomStore.messages.findIndex(
      (item) => item.pending && item.clientMsgId && item.clientMsgId === msg.clientMsgId
    )
    if (pendingIdx >= 0) {
      roomStore.messages[pendingIdx] = { ...msg, pending: false }
      return
    }
  }
  roomStore.appendMessage({ ...msg, pending: false })
}

function appendLocalEcho(content: string, clientMsgId: string, reply?: ChatMessage | null) {
  const selfMember = self.value
  if (!selfMember) return
  roomStore.appendMessage({
    id: localPendingSeq--,
    roomId,
    userId: selfMember.userId,
    nickname: selfMember.nickname,
    senderType: 'user',
    content,
    clientMsgId,
    pending: true,
    replyToMessageId: reply?.id,
    replyToSenderId: reply?.userId,
    replyToSenderName: reply?.nickname,
    replyToPreview: reply?.content,
    createdAt: new Date().toISOString()
  })
}

function mapDebugEvent(raw: any): BotDebugEvent {
  return {
    time: raw.time || new Date().toISOString(),
    event: raw.event || 'unknown',
    roomId: raw.room_id || raw.roomId || undefined,
    messageId: raw.message_id === undefined ? raw.messageId : Number(raw.message_id),
    senderId: raw.sender_id || raw.senderId || undefined,
    senderName: raw.sender_name || raw.senderName || undefined,
    content: raw.content || undefined,
    replyToMessageId:
      raw.reply_to_message_id === undefined || raw.reply_to_message_id === null
        ? raw.replyToMessageId
        : Number(raw.reply_to_message_id),
    replyToIsBot: raw.reply_to_is_bot === undefined ? raw.replyToIsBot : !!raw.reply_to_is_bot,
    botRole: raw.bot_role || raw.botRole || undefined,
    triggerReason: raw.trigger_reason || raw.triggerReason || undefined,
    triggerType: raw.trigger_type || raw.triggerType || undefined,
    skipReason: raw.skip_reason || raw.skipReason || undefined,
    botReplySkipped:
      raw.bot_reply_skipped === undefined ? raw.botReplySkipped : !!raw.bot_reply_skipped,
    forceReply: raw.force_reply === undefined ? raw.forceReply : !!raw.force_reply,
    hypeScore: raw.hype_score === undefined ? raw.hypeScore : Number(raw.hype_score),
    absurdityScore:
      raw.absurdity_score === undefined || raw.absurdity_score === null
        ? raw.absurdityScore
        : Number(raw.absurdity_score),
    riskScore:
      raw.risk_score === undefined || raw.risk_score === null ? raw.riskScore : Number(raw.risk_score),
    replyMode: raw.reply_mode || raw.replyMode || undefined,
    modelPool: raw.model_pool || raw.modelPool || undefined,
    candidateModels: raw.candidate_models || raw.candidateModels || undefined,
    triedModels: raw.tried_models || raw.triedModels || undefined,
    selectedModel: raw.selected_model || raw.selectedModel || undefined,
    modelFailures: raw.model_failures || raw.modelFailures || undefined,
    skippedModels: raw.skipped_models || raw.skippedModels || undefined,
    lastErrorType: raw.last_error_type || raw.lastErrorType || undefined,
    circuitOpenUntil: raw.circuit_open_until || raw.circuitOpenUntil || undefined,
    llmAttempted: raw.llm_attempted === undefined ? raw.llmAttempted : !!raw.llm_attempted,
    replySource: raw.reply_source || raw.replySource || undefined,
    fallbackReason: raw.fallback_reason || raw.fallbackReason || undefined,
    usedLLM: raw.used_llm === undefined ? raw.usedLLM : !!raw.used_llm,
    llmEnabled: raw.llm_enabled === undefined ? raw.llmEnabled : !!raw.llm_enabled,
    providerInitialized:
      raw.provider_initialized === undefined ? raw.providerInitialized : !!raw.provider_initialized,
    traceId: raw.trace_id || raw.traceId || undefined,
    model: raw.model || undefined,
    provider: raw.provider || undefined,
    baseUrl: raw.base_url || raw.baseUrl || undefined,
    apiKeyPresent: raw.api_key_present === undefined ? raw.apiKeyPresent : !!raw.api_key_present,
    httpStatus:
      raw.http_status === undefined || raw.http_status === null ? undefined : Number(raw.http_status),
    latencyMs:
      raw.latency_ms === undefined || raw.latency_ms === null ? undefined : Number(raw.latency_ms),
    promptTokens:
      raw.prompt_tokens === undefined || raw.prompt_tokens === null ? undefined : Number(raw.prompt_tokens),
    completionTokens:
      raw.completion_tokens === undefined || raw.completion_tokens === null
        ? undefined
        : Number(raw.completion_tokens),
    totalTokens:
      raw.total_tokens === undefined || raw.total_tokens === null ? undefined : Number(raw.total_tokens),
    requestPromptExcerpt: raw.request_prompt_excerpt || raw.requestPromptExcerpt || undefined,
    responseText: raw.response_text || raw.responseText || undefined,
    errorMessage: raw.error_message || raw.errorMessage || undefined,
    displayableContentFound:
      raw.displayable_content_found === undefined
        ? raw.displayableContentFound
        : !!raw.displayable_content_found,
    reasoningOnlyResponse:
      raw.reasoning_only_response === undefined
        ? raw.reasoningOnlyResponse
        : !!raw.reasoning_only_response
  }
}

function mapAuditMessage(raw: any) {
  if (!raw) return undefined
  return {
    id: Number(raw.id || 0),
    nickname: raw.nickname || '未知',
    senderType: (raw.sender_type || raw.senderType || 'system') as 'user' | 'bot' | 'system',
    content: raw.content || '',
    createdAt: raw.created_at || raw.createdAt || new Date().toISOString()
  }
}

function mapBotAudit(raw: any): BotReplyAudit {
  return {
    id: Number(raw.id || 0),
    traceId: raw.trace_id || raw.traceId || '',
    roomId: raw.room_id || raw.roomId || roomId,
    triggerMessageId:
      raw.trigger_message_id === undefined || raw.trigger_message_id === null
        ? undefined
        : Number(raw.trigger_message_id),
    triggerSenderId: raw.trigger_sender_id || raw.triggerSenderId || undefined,
    triggerSenderName: raw.trigger_sender_name || raw.triggerSenderName || '',
    replyMessageId:
      raw.reply_message_id === undefined || raw.reply_message_id === null
        ? undefined
        : Number(raw.reply_message_id),
    replySource: (raw.reply_source || raw.replySource || 'template') as 'llm' | 'template',
    botRole: raw.bot_role || raw.botRole || 'judge',
    firepowerLevel: raw.firepower_level || raw.firepowerLevel || 'medium',
    triggerReason: raw.trigger_reason || raw.triggerReason || undefined,
    triggerType: raw.trigger_type || raw.triggerType || undefined,
    forceReply: raw.force_reply === undefined ? !!raw.forceReply : !!raw.force_reply,
    hypeScore: raw.hype_score === undefined || raw.hype_score === null ? 0 : Number(raw.hype_score),
    absurdityScore:
      raw.absurdity_score === undefined || raw.absurdity_score === null
        ? undefined
        : Number(raw.absurdity_score),
    riskScore:
      raw.risk_score === undefined || raw.risk_score === null ? undefined : Number(raw.risk_score),
    replyMode: raw.reply_mode || raw.replyMode || undefined,
    llmEnabled: raw.llm_enabled === undefined ? !!raw.llmEnabled : !!raw.llm_enabled,
    providerInitialized:
      raw.provider_initialized === undefined ? !!raw.providerInitialized : !!raw.provider_initialized,
    apiKeyPresent: raw.api_key_present === undefined ? !!raw.apiKeyPresent : !!raw.api_key_present,
    provider: raw.provider || undefined,
    model: raw.model || undefined,
    requestPromptExcerpt: raw.request_prompt_excerpt || raw.requestPromptExcerpt || undefined,
    responseText: raw.response_text || raw.responseText || undefined,
    fallbackReason: raw.fallback_reason || raw.fallbackReason || undefined,
    httpStatus:
      raw.http_status === undefined || raw.http_status === null ? undefined : Number(raw.http_status),
    latencyMs: raw.latency_ms === undefined || raw.latency_ms === null ? undefined : Number(raw.latency_ms),
    promptTokens:
      raw.prompt_tokens === undefined || raw.prompt_tokens === null ? undefined : Number(raw.prompt_tokens),
    completionTokens:
      raw.completion_tokens === undefined || raw.completion_tokens === null
        ? undefined
        : Number(raw.completion_tokens),
    totalTokens:
      raw.total_tokens === undefined || raw.total_tokens === null ? undefined : Number(raw.total_tokens),
    errorMessage: raw.error_message || raw.errorMessage || undefined,
    displayableContentFound:
      raw.displayable_content_found === undefined
        ? raw.displayableContentFound
        : !!raw.displayable_content_found,
    reasoningOnlyResponse:
      raw.reasoning_only_response === undefined
        ? raw.reasoningOnlyResponse
        : !!raw.reasoning_only_response,
    createdAt: raw.created_at || raw.createdAt || new Date().toISOString(),
    updatedAt: raw.updated_at || raw.updatedAt || new Date().toISOString(),
    triggerMessage: mapAuditMessage(raw.trigger_message || raw.triggerMessage),
    replyMessage: mapAuditMessage(raw.reply_message || raw.replyMessage)
  }
}

async function loadBotAudits() {
  if (!showDebugUI) return
  auditLoading.value = true
  auditError.value = ''
  try {
    const res = await api.debugRoomBotAudits(roomId, { page: 1, pageSize: 20 })
    botAudits.value = (res.items || []).map(mapBotAudit)
  } catch (e: any) {
    auditError.value = e.message || '加载审计记录失败'
  } finally {
    auditLoading.value = false
  }
}

async function loadRoom() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.getRoom(roomId)
    roomStore.room = res.room
    roomStore.self = res.self
    roomStore.members = res.members

    const history = await api.roomMessages(roomId)
    roomStore.messages = history.items.map(mapMessage)
  } catch (e: any) {
    error.value = e.message || '加载房间失败'
  } finally {
    loading.value = false
  }
}

function connectWS() {
  const token = userStore.token
  if (!token) return

  socket.value = new RoomSocket(roomId, token, {
    onOpen: () => {
      roomStore.wsConnected = true
    },
    onClose: () => {
      roomStore.wsConnected = false
    },
    onMessage: (evt) => {
      const { type, payload } = evt
      if (type === 'bootstrap') {
        roomStore.room = payload.room as Room
        roomStore.self = payload.member as RoomMember
        roomStore.members = payload.members as RoomMember[]
        roomStore.messages = (payload.messages || []).map(mapMessage)
      }
      if (type === 'chat') {
        mergeIncomingMessage(mapMessage(payload))
      }
      if (type === 'system') {
        if (payload?.userId && payload.userId === self.value?.userId) {
          let pendingIdx = -1
          for (let i = roomStore.messages.length - 1; i >= 0; i -= 1) {
            const item = roomStore.messages[i]
            if (item.pending && item.senderType === 'user' && item.userId === self.value?.userId) {
              pendingIdx = i
              break
            }
          }
          if (pendingIdx >= 0) {
            roomStore.messages.splice(pendingIdx, 1)
          }
        }
        roomStore.appendMessage(
          mapMessage({
            id: Date.now(),
            nickname: '系统提示',
            senderType: 'system',
            content: payload?.content || '',
            createdAt: new Date().toISOString()
          })
        )
      }
      if (type === 'member_update') {
        roomStore.members = payload.members || []
      }
      if (type === 'control_update') {
        if (payload.room) {
          roomStore.room = payload.room as Room
        }
      }
      if (type === 'room_end') {
        roomStore.report = payload.report as RoomReport
        router.push(`/room/${roomId}/report`)
      }
      if (type === 'bot_debug') {
        roomStore.appendBotDebugEvent(mapDebugEvent(payload))
      }
    }
  })

  socket.value.connect()
}

function send() {
  if (!input.value.trim()) return
  if (!roomStore.wsConnected) {
    pushNotice('连接恢复中，请稍后发送', 'error')
    return
  }
  const text = input.value.trim()
  const clientMsgId = `c_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  appendLocalEcho(text, clientMsgId, replyingTo.value)
  if (replyingTo.value) {
    socket.value?.sendChat(text, {
      replyToMessageId: replyingTo.value.id,
      replyToSenderName: replyingTo.value.nickname,
      replyToPreview: replyingTo.value.content
    }, clientMsgId)
  } else {
    socket.value?.sendChat(text, undefined, clientMsgId)
  }
  input.value = ''
  replyingTo.value = null
}

function handleQuote(msg: ChatMessage) {
  if (msg.senderType === 'system') return
  replyingTo.value = msg
}

function cancelReply() {
  replyingTo.value = null
}

function openBotProfile(payload: {
  role: BotRole
  name: string
  bio: string
  identity: string
  duty: string
}) {
  activeBotProfile.value = payload
  showBotProfile.value = true
}

async function doControl(action: string, value = '') {
  try {
    await api.roomControl(roomId, action, value)
    if (action === 'end_room') {
      const { report } = await api.roomReport(roomId)
      roomStore.report = report
      router.push(`/room/${roomId}/report`)
      return
    }
    pushNotice('操作已生效', 'ok')
  } catch (e: any) {
    pushNotice(e.message || '操作失败', 'error')
  }
}

async function runLLMTest() {
  try {
    await api.debugLLMTest(roomId)
    pushNotice('已触发 Bot 测试回复', 'ok')
  } catch (e: any) {
    pushNotice(e.message || 'Bot 测试失败', 'error')
  }
}

async function switchIdentityFromSheet(identity: Identity, confirmTarget: boolean) {
  if (!selectedMember.value || selectedMember.value.userId !== self.value?.userId) {
    pushNotice('当前仅可切换自己的身份', 'info')
    return
  }
  try {
    await api.updateIdentity(roomId, identity, confirmTarget)
    const res = await api.getRoom(roomId)
    roomStore.self = res.self
    roomStore.members = res.members
    pushNotice('身份已更新', 'ok')
    showMemberActionSheet.value = false
  } catch (e: any) {
    pushNotice(e.message || '切换失败', 'error')
  }
}

async function reportMember(targetUserId: string, message: string) {
  try {
    await api.reportAbuse(roomId, message, targetUserId)
    pushNotice('已提交举报，感谢反馈', 'ok')
    showMemberActionSheet.value = false
  } catch (e: any) {
    pushNotice(e.message || '举报失败', 'error')
  }
}

function onMoreAction(action: 'invite' | 'members' | 'host' | 'guide' | 'debug' | 'theme') {
  showMoreMenu.value = false
  if (action === 'invite') {
    showInviteSheet.value = true
    return
  }
  if (action === 'members') {
    showMembersSheet.value = true
    return
  }
  if (action === 'host') {
    showHostToolSheet.value = true
    return
  }
  if (action === 'guide') {
    showGuideSheet.value = true
    return
  }
  if (action === 'debug') {
    showDebugDrawer.value = true
    void loadBotAudits()
    return
  }
  if (action === 'theme') {
    showThemeSheet.value = true
  }
}

function openMemberAction(member: RoomMember) {
  selectedMember.value = member
  showMembersSheet.value = false
  showMemberActionSheet.value = true
}

watch(
  () => roomStore.messages.length,
  async () => {
    await nextTick()
    if (listRef.value) {
      listRef.value.scrollTop = listRef.value.scrollHeight
    }
  }
)

watch(
  () => members.value,
  () => {
    if (!selectedMember.value) return
    const latest = members.value.find((m) => m.userId === selectedMember.value?.userId)
    if (latest) selectedMember.value = latest
  },
  { deep: true }
)

watch(
  () => showDebugDrawer.value,
  (open) => {
    if (open) {
      void loadBotAudits()
    }
  }
)

onMounted(async () => {
  roomStore.botDebugEvents = []
  await loadRoom()
  connectWS()
})

onUnmounted(() => {
  socket.value?.close()
  if (noticeTimer) clearTimeout(noticeTimer)
})
</script>

<template>
  <main class="room-page">
    <RoomHeader :room="room" :ws-connected="roomStore.wsConnected" @open-more="showMoreMenu = true" />
    <RoomMoreMenu
      :open="showMoreMenu"
      :is-owner="isOwner"
      :show-debug="showDebugUI"
      @close="showMoreMenu = false"
      @action="onMoreAction"
    />

    <section v-if="error" class="state-card">
      <div class="title">房间加载失败</div>
      <div class="muted">{{ error }}</div>
      <button class="retry" @click="loadRoom">重试</button>
    </section>

    <section v-else ref="listRef" class="chat-stream">
      <div v-if="roomStore.messages.length === 0" class="empty-state">
        <div class="empty-title">聊天刚开始</div>
        <div class="empty-sub">发第一句试试，比如“都行？”“稳了？”</div>
      </div>

      <MessageBubble
        v-for="msg in roomStore.messages"
        :key="`${msg.id}-${msg.createdAt}`"
        :msg="msg"
        :self-user-id="self?.userId"
        @quote="handleQuote"
        @open-profile="openBotProfile"
      />
    </section>

    <transition name="fade">
      <div v-if="notice" class="toast" :class="notice.type">{{ notice.text }}</div>
    </transition>

    <ChatComposer
      v-model="input"
      :loading="loading"
      :replying-to="replyingTo"
      @send="send"
      @cancel-reply="cancelReply"
    />

    <InviteSheet
      :open="showInviteSheet"
      :room-id="roomId"
      :share-code="room?.shareCode || '--'"
      @close="showInviteSheet = false"
    />

    <MemberListSheet
      :open="showMembersSheet"
      :members="members"
      :self-user-id="self?.userId || ''"
      @close="showMembersSheet = false"
      @select-member="openMemberAction"
    />

    <MemberActionSheet
      :open="showMemberActionSheet"
      :member="selectedMember"
      :self-user-id="self?.userId || ''"
      @close="showMemberActionSheet = false"
      @switch-identity="switchIdentityFromSheet"
      @report="reportMember"
    />

    <HostControlSheet
      :open="showHostToolSheet"
      :can-control="isOwner"
      :show-dev-tools="showDebugUI"
      @close="showHostToolSheet = false"
      @control="doControl"
      @invite="showInviteSheet = true"
      @llm-test="runLLMTest"
    />

    <BotDebugDrawer
      :open="showDebugDrawer"
      :events="roomStore.botDebugEvents"
      :audits="botAudits"
      :loading="auditLoading"
      :error="auditError"
      @refresh="loadBotAudits"
      @close="showDebugDrawer = false"
    />
    <BotProfileCard
      :open="showBotProfile"
      :profile="activeBotProfile"
      @close="showBotProfile = false"
    />
    <ThemePickerSheet :open="showThemeSheet" @close="showThemeSheet = false" />

    <Teleport to="body">
      <div v-if="showGuideSheet" class="guide-wrap">
        <button class="mask" @click="showGuideSheet = false"></button>
        <div class="guide-panel">
          <h3>房间说明</h3>
          <p>1. 聊天优先，Bot 会根据发言内容和气氛插话。</p>
          <p>2. target 会被更频繁关注，immune 会尽量避开重点吐槽。</p>
          <p>3. 房间结束后会自动生成本局战报。</p>
          <button class="guide-close" @click="showGuideSheet = false">知道了</button>
        </div>
      </div>
    </Teleport>
  </main>
</template>

<style scoped>
.room-page {
  height: 100dvh;
  max-width: 760px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  padding: 12px 12px calc(84px + env(safe-area-inset-bottom));
  background: radial-gradient(circle at 14% 8%, var(--app-glow-1) 0%, transparent 36%),
    radial-gradient(circle at 88% 2%, var(--app-glow-2) 0%, transparent 30%);
}

.chat-stream {
  margin-top: 10px;
  flex: 1;
  min-height: 0;
  overflow: auto;
  border-radius: 18px;
  border: 1px solid var(--chat-stream-border);
  padding: 12px 12px 18px;
  background: var(--chat-stream-bg);
  box-shadow: var(--chat-stream-shadow);
}

.empty-state {
  min-height: 140px;
  display: grid;
  place-content: center;
  text-align: center;
}

.empty-title {
  font-size: 15px;
  font-weight: 700;
}

.empty-sub {
  margin-top: 6px;
  color: var(--text-sub);
  font-size: 13px;
}

.state-card {
  margin-top: 10px;
  border-radius: 16px;
  border: 1px solid var(--state-error-border);
  background: var(--state-error-bg);
  color: var(--state-error-text);
  padding: 14px;
}

.state-card .title {
  font-weight: 700;
  margin-bottom: 5px;
}

.retry {
  margin-top: 10px;
  border: 0;
  border-radius: 10px;
  color: var(--accent-contrast);
  font-weight: 700;
  padding: 9px 12px;
  background: linear-gradient(120deg, var(--danger), var(--warn));
}

.toast {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  bottom: calc(76px + env(safe-area-inset-bottom));
  z-index: 25;
  min-width: 190px;
  max-width: min(90vw, 460px);
  text-align: center;
  border-radius: 999px;
  padding: 8px 14px;
  font-size: 13px;
  color: var(--toast-text);
  border: 1px solid var(--toast-border);
  background: var(--toast-bg);
}

.toast.ok {
  border-color: var(--toast-ok-border);
  background: var(--toast-ok-bg);
}

.toast.error {
  border-color: var(--toast-error-border);
  background: var(--toast-error-bg);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.guide-wrap {
  position: fixed;
  inset: 0;
  z-index: 59;
}

.mask {
  position: absolute;
  inset: 0;
  border: 0;
  background: var(--overlay-mask);
}

.guide-panel {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: 0;
  width: min(760px, 100%);
  border-radius: 20px 20px 0 0;
  border: 1px solid var(--sheet-border);
  border-bottom: 0;
  padding: 14px 14px calc(16px + env(safe-area-inset-bottom));
  background: var(--sheet-bg);
}

.guide-panel h3 {
  margin-top: 0;
}

.guide-panel p {
  margin: 7px 0;
  color: var(--guide-text);
  font-size: 14px;
  line-height: 1.45;
}

.guide-close {
  margin-top: 8px;
  width: 100%;
  border: 0;
  border-radius: 10px;
  padding: 10px;
  color: var(--accent-contrast);
  font-weight: 700;
  background: linear-gradient(125deg, var(--btn-grad-1), var(--btn-grad-2));
}
</style>
