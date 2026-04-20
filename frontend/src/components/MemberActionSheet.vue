<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Identity, RoomMember } from '@/types'

const props = defineProps<{
  open: boolean
  member: RoomMember | null
  selfUserId: string
}>()

const emit = defineEmits<{
  close: []
  switchIdentity: [identity: Identity, confirmTarget: boolean]
  report: [targetUserId: string, message: string]
}>()

const reportText = ref('')
const targetPending = ref(false)

const isSelf = computed(() => props.member?.userId === props.selfUserId)

watch(
  () => props.open,
  (open) => {
    if (open) {
      reportText.value = ''
      targetPending.value = false
    }
  }
)

function chooseIdentity(identity: Identity) {
  if (!isSelf.value) return
  if (identity === 'target' && !targetPending.value) {
    targetPending.value = true
    return
  }
  emit('switchIdentity', identity, identity !== 'target' || targetPending.value)
  targetPending.value = false
}

function submitReport() {
  if (!props.member) return
  emit('report', props.member.userId, reportText.value.trim() || '存在不适内容')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open && props.member" class="sheet-wrap">
      <button class="mask" @click="emit('close')"></button>
      <div class="sheet">
        <div class="sheet-head">
          <h3>{{ props.member.nickname }}</h3>
          <button class="close" @click="emit('close')">×</button>
        </div>

        <div class="meta-line">
          <span class="muted">当前身份：{{ props.member.identity }}</span>
          <span class="muted" v-if="props.member.isOwner">房主</span>
        </div>

        <div class="block" v-if="isSelf">
          <div class="label">切换我的身份</div>
          <div class="chips">
            <button class="chip-btn" @click="chooseIdentity('normal')">normal</button>
            <button class="chip-btn" @click="chooseIdentity('target')">target</button>
            <button class="chip-btn" @click="chooseIdentity('immune')">immune</button>
          </div>
          <div v-if="targetPending" class="warn">target 需要二次确认，请再点一次 target。</div>
        </div>

        <div class="block" v-else>
          <div class="label">身份操作</div>
          <div class="muted">当前版本仅支持修改自己的身份，你可对该成员进行举报。</div>
        </div>

        <div class="block">
          <div class="label">举报该成员</div>
          <textarea v-model="reportText" class="report-field" placeholder="可选：补充举报原因（不会公开）"></textarea>
          <button class="report-btn" @click="submitReport">提交举报</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-wrap {
  position: fixed;
  inset: 0;
  z-index: 57;
}

.mask {
  position: absolute;
  inset: 0;
  border: 0;
  background: var(--overlay-mask);
}

.sheet {
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

.sheet-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sheet-head h3 {
  margin: 0;
}

.close {
  border: 0;
  background: var(--sheet-close-bg);
  color: var(--sheet-close-text);
  border-radius: 8px;
  width: 28px;
  height: 28px;
  font-size: 18px;
}

.meta-line {
  margin-top: 6px;
  display: flex;
  gap: 10px;
}

.block {
  margin-top: 12px;
  border: 1px solid var(--sheet-item-border);
  border-radius: 12px;
  padding: 10px;
  background: var(--sheet-item-bg);
}

.label {
  font-size: 13px;
  color: var(--text-sub);
  margin-bottom: 8px;
}

.chips {
  display: flex;
  gap: 8px;
}

.chip-btn {
  border: 1px solid var(--chip-border);
  border-radius: 999px;
  padding: 7px 10px;
  color: var(--chip-text);
  background: var(--chip-bg);
}

.warn {
  margin-top: 8px;
  color: var(--warn);
  font-size: 12px;
}

.muted {
  color: var(--text-sub);
  font-size: 13px;
}

.report-field {
  width: 100%;
  min-height: 66px;
  border-radius: 10px;
  border: 1px solid var(--input-border);
  color: var(--input-text);
  background: var(--input-bg);
  padding: 8px;
  resize: none;
}

.report-btn {
  margin-top: 8px;
  border: 0;
  border-radius: 10px;
  width: 100%;
  padding: 10px;
  color: var(--accent-contrast);
  font-weight: 700;
  background: linear-gradient(120deg, var(--danger), var(--warn));
}
</style>
