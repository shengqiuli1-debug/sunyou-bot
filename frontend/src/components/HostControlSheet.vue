<script setup lang="ts">
import { ref, watch } from 'vue'
import { FIRE_OPTIONS, ROLE_OPTIONS } from '@/constants/copy'

const props = defineProps<{
  open: boolean
  canControl: boolean
  showDevTools?: boolean
}>()

const emit = defineEmits<{
  close: []
  control: [action: string, value?: string]
  invite: []
  llmTest: []
}>()

const endingPending = ref(false)

watch(
  () => props.open,
  (open) => {
    if (open) endingPending.value = false
  }
)

function endRoom() {
  if (!endingPending.value) {
    endingPending.value = true
    return
  }
  emit('control', 'end_room')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open && props.canControl" class="sheet-wrap">
      <button class="mask" @click="emit('close')"></button>
      <div class="sheet">
        <div class="sheet-head">
          <h3>房主工具</h3>
          <button class="close" @click="emit('close')">×</button>
        </div>

        <section class="group">
          <div class="group-title">Bot 设置</div>
          <div class="label">角色切换</div>
          <div class="chips">
            <button class="chip-btn" v-for="r in ROLE_OPTIONS" :key="r.value" @click="emit('control', 'switch_role', r.value)">
              {{ r.label }}
            </button>
          </div>

          <div class="label">火力等级</div>
          <div class="chips">
            <button class="chip-btn" v-for="f in FIRE_OPTIONS" :key="f.value" @click="emit('control', 'switch_fire', f.value)">
              {{ f.label }}
            </button>
          </div>
        </section>

        <section class="group">
          <div class="group-title">临时控制</div>
          <button class="op-btn" @click="emit('control', 'mute_bot', '20')">Bot 闭嘴 20 秒</button>
        </section>

        <section class="group">
          <div class="group-title">房间操作</div>
          <button class="op-btn" @click="emit('invite')">邀请朋友</button>
          <button v-if="props.showDevTools" class="op-btn dev" @click="emit('llmTest')">测试 Bot 回复</button>
          <button class="op-btn danger" @click="endRoom">{{ endingPending ? '再次点击确认结束房间' : '结束房间' }}</button>
        </section>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-wrap {
  position: fixed;
  inset: 0;
  z-index: 58;
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
  max-height: 80dvh;
  overflow: auto;
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
  margin-bottom: 10px;
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

.group {
  border: 1px solid var(--sheet-item-border);
  border-radius: 12px;
  padding: 10px;
  margin-bottom: 10px;
  background: var(--sheet-item-bg);
}

.group-title {
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--text-main);
}

.label {
  margin-bottom: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-sub);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.chip-btn {
  border: 1px solid var(--chip-border);
  border-radius: 999px;
  padding: 7px 10px;
  color: var(--chip-text);
  background: var(--chip-bg);
}

.op-btn {
  width: 100%;
  border: 1px solid var(--sheet-item-border);
  border-radius: 10px;
  padding: 10px;
  color: var(--text-main);
  background: var(--sheet-item-bg);
  margin-bottom: 8px;
}

.op-btn.danger {
  border-color: transparent;
  color: var(--accent-contrast);
  font-weight: 700;
  background: linear-gradient(120deg, var(--danger), var(--warn));
}

.op-btn.dev {
  border-color: var(--ok);
  background: var(--sheet-item-bg);
}
</style>
