<script setup lang="ts">
import { ROLE_OPTIONS, FIRE_OPTIONS } from '@/constants/copy'

const props = defineProps<{ isOwner: boolean }>()
const emit = defineEmits<{
  control: [action: string, value?: string]
}>()
</script>

<template>
  <div v-if="props.isOwner" class="card">
    <h3>房主控制台</h3>
    <div class="stack">
      <div class="row">
        <span class="muted">切换 Bot 角色</span>
        <div class="chips">
          <button class="btn ghost" v-for="r in ROLE_OPTIONS" :key="r.value" @click="emit('control', 'switch_role', r.value)">
            {{ r.label }}
          </button>
        </div>
      </div>
      <div class="row">
        <span class="muted">切换火力等级</span>
        <div class="chips">
          <button class="btn ghost" v-for="f in FIRE_OPTIONS" :key="f.value" @click="emit('control', 'switch_fire', f.value)">
            {{ f.label }}
          </button>
        </div>
      </div>
      <div class="chips">
        <button class="btn ghost" @click="emit('control', 'mute_bot', '20')">Bot 闭嘴 20 秒</button>
        <button class="btn warn" @click="emit('control', 'end_room')">结束房间</button>
      </div>
    </div>
  </div>
</template>
