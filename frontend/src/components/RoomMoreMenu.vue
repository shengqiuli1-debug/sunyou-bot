<script setup lang="ts">
const props = defineProps<{
  open: boolean
  isOwner: boolean
  showDebug?: boolean
}>()

const emit = defineEmits<{
  close: []
  action: [action: 'invite' | 'members' | 'host' | 'guide' | 'debug' | 'theme']
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="menu-wrap">
      <button class="mask" @click="emit('close')" aria-label="关闭"></button>
      <div class="menu-panel">
        <button class="menu-item" @click="emit('action', 'invite')">邀请朋友</button>
        <button class="menu-item" @click="emit('action', 'members')">查看成员</button>
        <button class="menu-item" @click="emit('action', 'theme')">主题切换</button>
        <button class="menu-item" v-if="props.isOwner" @click="emit('action', 'host')">房主工具</button>
        <button class="menu-item" v-if="props.showDebug" @click="emit('action', 'debug')">调试面板</button>
        <button class="menu-item" @click="emit('action', 'guide')">房间说明</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.menu-wrap {
  position: fixed;
  inset: 0;
  z-index: 42;
}

.mask {
  position: absolute;
  inset: 0;
  border: 0;
  background: var(--overlay-mask);
}

.menu-panel {
  position: absolute;
  top: 76px;
  right: max(12px, calc((100vw - 760px) / 2 + 12px));
  width: 178px;
  border-radius: 14px;
  padding: 6px;
  border: 1px solid var(--menu-border);
  background: var(--menu-bg);
  box-shadow: var(--menu-shadow);
}

.menu-item {
  width: 100%;
  text-align: left;
  border: 0;
  background: transparent;
  color: var(--menu-item-text);
  padding: 11px 10px;
  border-radius: 9px;
}

.menu-item:active {
  background: var(--menu-item-active-bg);
}
</style>
