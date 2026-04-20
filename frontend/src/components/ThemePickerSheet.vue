<script setup lang="ts">
import { useThemeStore } from '@/stores/theme'
import type { ThemeId } from '@/constants/themes'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  close: []
}>()

const themeStore = useThemeStore()

function selectTheme(id: ThemeId) {
  themeStore.setTheme(id)
}
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="sheet-wrap">
      <button class="mask" @click="emit('close')" aria-label="关闭主题面板"></button>
      <div class="sheet">
        <div class="sheet-head">
          <h3>主题外观</h3>
          <button class="close" @click="emit('close')">×</button>
        </div>

        <p class="sheet-sub">切换房间氛围，主题会自动记忆。</p>

        <button
          v-for="theme in themeStore.options"
          :key="theme.id"
          class="theme-row"
          :class="{ active: themeStore.themeId === theme.id }"
          @click="selectTheme(theme.id)"
        >
          <div class="theme-main">
            <div class="theme-title">
              {{ theme.name }}
              <span class="theme-tone">{{ theme.tone }}</span>
            </div>
            <div class="theme-desc">{{ theme.description }}</div>
          </div>
          <div class="swatches" aria-hidden="true">
            <span v-for="(c, idx) in theme.swatches" :key="`${theme.id}-${idx}`" class="dot" :style="{ background: c }"></span>
          </div>
        </button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-wrap {
  position: fixed;
  inset: 0;
  z-index: 61;
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
  max-height: 84dvh;
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
}

.sheet-head h3 {
  margin: 0;
}

.sheet-sub {
  margin: 8px 0 10px;
  color: var(--text-sub);
  font-size: 13px;
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

.theme-row {
  width: 100%;
  border: 1px solid var(--sheet-item-border);
  border-radius: 14px;
  padding: 10px;
  margin-bottom: 9px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  background: var(--sheet-item-bg);
  color: var(--text-main);
}

.theme-row.active {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--chip-border);
}

.theme-main {
  text-align: left;
  min-width: 0;
}

.theme-title {
  font-size: 14px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}

.theme-tone {
  font-weight: 500;
  font-size: 12px;
  color: var(--text-sub);
}

.theme-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-sub);
}

.swatches {
  display: flex;
  gap: 5px;
}

.dot {
  width: 14px;
  height: 14px;
  border-radius: 999px;
  border: 1px solid var(--line);
}
</style>
