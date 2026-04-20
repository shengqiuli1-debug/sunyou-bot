<script setup lang="ts">
import { computed } from 'vue'
import type { BotDebugEvent, BotReplyAudit } from '@/types'

const props = defineProps<{
  open: boolean
  events: BotDebugEvent[]
  audits: BotReplyAudit[]
  loading: boolean
  error: string
}>()

const emit = defineEmits<{
  close: []
  refresh: []
}>()

const recentEvents = computed(() => props.events.slice(-20).reverse())
const recentAudits = computed(() => props.audits.slice(0, 20))

function fmtTime(v?: string) {
  if (!v) return '--:--:--'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  return d.toLocaleTimeString()
}

function sourceLabel(v?: string) {
  return v === 'llm' ? 'LLM' : '模板'
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="drawer-wrap">
      <button class="mask" @click="emit('close')" aria-label="关闭调试面板"></button>
      <aside class="drawer">
        <div class="drawer-head">
          <h3>Bot 调试面板</h3>
          <div class="head-ops">
            <button class="refresh" @click="emit('refresh')">刷新审计</button>
            <button class="close" @click="emit('close')">×</button>
          </div>
        </div>

        <section class="audit-section">
          <div class="section-title">落库审计链路（最近 20 条）</div>
          <div v-if="props.loading" class="empty">正在加载审计记录...</div>
          <div v-else-if="props.error" class="empty">{{ props.error }}</div>
          <div v-else-if="recentAudits.length > 0" class="event-list">
            <article class="event-card audit" v-for="item in recentAudits" :key="item.id">
              <div class="line top">
                <span class="time">{{ fmtTime(item.createdAt) }}</span>
                <code class="evt">#{{ item.id }} | {{ sourceLabel(item.replySource) }}</code>
              </div>
              <div class="line"><b>触发：</b>{{ item.triggerMessage?.content || '-' }}</div>
              <div class="line"><b>回复：</b>{{ item.replyMessage?.content || item.responseText || '-' }}</div>
              <div class="line">trigger_reason: {{ item.triggerReason || '-' }}</div>
              <div class="line">trigger_type: {{ item.triggerType || '-' }}</div>
              <div class="line">force_reply: {{ item.forceReply ? 'true' : 'false' }}</div>
              <div class="line">llm_enabled: {{ item.llmEnabled ? 'true' : 'false' }}</div>
              <div class="line">provider_initialized: {{ item.providerInitialized ? 'true' : 'false' }}</div>
              <div class="line">api_key_present: {{ item.apiKeyPresent ? 'true' : 'false' }}</div>
              <div class="line">displayable_content_found: {{ item.displayableContentFound ? 'true' : 'false' }}</div>
              <div class="line">reasoning_only_response: {{ item.reasoningOnlyResponse ? 'true' : 'false' }}</div>
              <div class="line">absurdity_score: {{ item.absurdityScore ?? '-' }}</div>
              <div class="line">risk_score: {{ item.riskScore ?? '-' }}</div>
              <div class="line">reply_mode: {{ item.replyMode || '-' }}</div>
              <div class="line">provider/model: {{ item.provider || '-' }} / {{ item.model || '-' }}</div>
              <div class="line">fallback_reason: {{ item.fallbackReason || '-' }}</div>
              <div class="line">latency: {{ item.latencyMs ?? '-' }} ms</div>
              <div class="line">trace_id: {{ item.traceId || '-' }}</div>
            </article>
          </div>
          <div v-else class="empty">暂无审计记录，先触发一次 Bot 回复。</div>
        </section>

        <section class="audit-section">
          <div class="section-title">实时事件（最近 20 条）</div>
        <div class="event-list" v-if="recentEvents.length > 0">
          <article class="event-card" v-for="(evt, idx) in recentEvents" :key="`${evt.time}-${evt.event}-${idx}`">
            <div class="line top">
              <span class="time">{{ fmtTime(evt.time) }}</span>
              <code class="evt">{{ evt.event }}</code>
            </div>
            <div class="line">trigger_reason: {{ evt.triggerReason || '-' }}</div>
            <div class="line">trigger_type: {{ evt.triggerType || '-' }}</div>
            <div class="line">skip_reason: {{ evt.skipReason || '-' }}</div>
            <div class="line">force_reply: {{ evt.forceReply ? 'true' : 'false' }}</div>
            <div class="line">hype_score: {{ evt.hypeScore ?? '-' }}</div>
            <div class="line">absurdity_score: {{ evt.absurdityScore ?? '-' }}</div>
            <div class="line">risk_score: {{ evt.riskScore ?? '-' }}</div>
            <div class="line">reply_mode: {{ evt.replyMode || '-' }}</div>
            <div class="line">model_pool: {{ evt.modelPool || '-' }}</div>
            <div class="line">candidate_models: {{ evt.candidateModels || '-' }}</div>
            <div class="line">skipped_models: {{ evt.skippedModels || '-' }}</div>
            <div class="line">tried_models: {{ evt.triedModels || '-' }}</div>
            <div class="line">selected_model: {{ evt.selectedModel || '-' }}</div>
            <div class="line">model_failures: {{ evt.modelFailures || '-' }}</div>
            <div class="line">last_error_type: {{ evt.lastErrorType || '-' }}</div>
            <div class="line">circuit_open_until: {{ evt.circuitOpenUntil || '-' }}</div>
            <div class="line">reply_source: {{ evt.replySource || '-' }}</div>
            <div class="line">fallback_reason: {{ evt.fallbackReason || '-' }}</div>
            <div class="line">bot_reply_skipped: {{ evt.botReplySkipped ? 'true' : 'false' }}</div>
            <div class="line">llm_attempted: {{ evt.llmAttempted ? 'true' : 'false' }}</div>
            <div class="line">used_llm: {{ evt.usedLLM ? 'true' : 'false' }}</div>
            <div class="line">llm_enabled: {{ evt.llmEnabled ? 'true' : 'false' }}</div>
            <div class="line">provider_initialized: {{ evt.providerInitialized ? 'true' : 'false' }}</div>
            <div class="line">api_key_present: {{ evt.apiKeyPresent ? 'true' : 'false' }}</div>
            <div class="line">displayable_content_found: {{ evt.displayableContentFound ? 'true' : 'false' }}</div>
            <div class="line">reasoning_only_response: {{ evt.reasoningOnlyResponse ? 'true' : 'false' }}</div>
            <div class="line">trace_id: {{ evt.traceId || '-' }}</div>
          </article>
        </div>
        <div class="empty" v-else>暂无调试事件，先发几条消息试试。</div>
        </section>
      </aside>
    </div>
  </Teleport>
</template>

<style scoped>
.drawer-wrap {
  position: fixed;
  inset: 0;
  z-index: 70;
}

.mask {
  position: absolute;
  inset: 0;
  border: 0;
  background: var(--overlay-mask);
}

.drawer {
  position: absolute;
  top: 0;
  right: 0;
  height: 100dvh;
  width: min(420px, 92vw);
  border-left: 1px solid var(--drawer-border);
  background: var(--drawer-bg);
  padding: 12px;
  display: flex;
  flex-direction: column;
  overflow: auto;
}

.drawer-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.head-ops {
  display: flex;
  align-items: center;
  gap: 6px;
}

.refresh {
  border: 1px solid var(--drawer-chip-border);
  border-radius: 9px;
  background: var(--drawer-chip-bg);
  color: var(--drawer-text);
  font-size: 12px;
  padding: 6px 8px;
}

.drawer-head h3 {
  margin: 0;
  font-size: 15px;
}

.close {
  border: 0;
  border-radius: 8px;
  width: 28px;
  height: 28px;
  font-size: 18px;
  color: var(--sheet-close-text);
  background: var(--sheet-close-bg);
}

.event-list {
  overflow: auto;
}

.event-card {
  border: 1px solid var(--drawer-card-border);
  border-radius: 10px;
  padding: 8px;
  margin-bottom: 8px;
  background: var(--drawer-card-bg);
}

.event-card.audit {
  border-color: var(--drawer-audit-border);
  background: var(--drawer-audit-bg);
}

.audit-section {
  margin-bottom: 10px;
}

.section-title {
  font-size: 12px;
  color: var(--drawer-section-title);
  margin-bottom: 6px;
}

.line {
  font-size: 12px;
  color: var(--drawer-text);
  margin-top: 3px;
  word-break: break-all;
}

.line.top {
  margin-top: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.time {
  color: var(--drawer-muted);
}

.evt {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 999px;
  border: 1px solid var(--drawer-chip-border);
  background: var(--drawer-chip-bg);
}

.empty {
  margin-top: 12px;
  font-size: 13px;
  color: var(--drawer-muted);
}
</style>
