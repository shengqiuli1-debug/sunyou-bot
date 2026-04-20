<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  open: boolean
  roomId: string
  shareCode: string
}>()

const emit = defineEmits<{
  close: []
}>()

const origin = window.location.origin
const copied = ref('')

const links = computed(() => ({
  normal: `${origin}/join/${props.roomId}?identity=normal`,
  target: `${origin}/join/${props.roomId}?identity=target`,
  immune: `${origin}/join/${props.roomId}?identity=immune`
}))

async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = `${label}已复制`
    setTimeout(() => {
      copied.value = ''
    }, 1400)
  } catch {
    copied.value = '复制失败，请手动复制'
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="sheet-wrap">
      <button class="mask" @click="emit('close')"></button>
      <div class="sheet">
        <div class="sheet-head">
          <h3>邀请朋友</h3>
          <button class="close" @click="emit('close')">×</button>
        </div>

        <div class="section">
          <div class="label">房间分享码</div>
          <div class="code-row">
            <strong>{{ props.shareCode }}</strong>
            <button class="mini" @click="copyText(props.shareCode, '分享码')">复制码</button>
          </div>
        </div>

        <div class="section">
          <div class="label">邀请链接</div>
          <div class="stack">
            <button class="link-item" @click="copyText(links.normal, '普通入口链接')">普通入口链接</button>
            <button class="link-item" @click="copyText(links.target, '上靶入口链接')">上靶入口链接（需二次确认）</button>
            <button class="link-item" @click="copyText(links.immune, '免疫入口链接')">免疫入口链接</button>
          </div>
        </div>

        <div class="hint">上靶适合熟人局；不想被重点关注可选 normal 或 immune。</div>
        <div v-if="copied" class="ok">{{ copied }}</div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-wrap {
  position: fixed;
  inset: 0;
  z-index: 55;
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
  align-items: center;
  justify-content: space-between;
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

.section {
  margin-top: 12px;
  border: 1px solid var(--sheet-item-border);
  border-radius: 12px;
  padding: 10px;
  background: var(--sheet-item-bg);
}

.label {
  color: var(--text-sub);
  font-size: 12px;
  margin-bottom: 8px;
}

.code-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mini {
  border: 1px solid var(--sheet-item-border);
  background: var(--btn-ghost-bg);
  color: var(--btn-ghost-text);
  border-radius: 8px;
  padding: 6px 9px;
}

.stack {
  display: grid;
  gap: 8px;
}

.link-item {
  border: 1px solid var(--sheet-item-border);
  color: var(--text-main);
  border-radius: 10px;
  padding: 9px 10px;
  text-align: left;
  background: var(--sheet-item-bg);
}

.hint {
  margin-top: 10px;
  color: var(--text-sub);
  font-size: 13px;
}

.ok {
  margin-top: 8px;
  color: var(--ok);
  font-size: 13px;
}
</style>
