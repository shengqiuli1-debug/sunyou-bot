<script setup lang="ts">
import IdentityBadge from '@/components/IdentityBadge.vue'
import type { RoomMember } from '@/types'

const props = defineProps<{
  open: boolean
  members: RoomMember[]
  selfUserId: string
}>()

const emit = defineEmits<{
  close: []
  selectMember: [member: RoomMember]
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="props.open" class="sheet-wrap">
      <button class="mask" @click="emit('close')"></button>
      <div class="sheet">
        <div class="sheet-head">
          <h3>成员列表</h3>
          <button class="close" @click="emit('close')">×</button>
        </div>

        <button
          v-for="member in props.members"
          :key="member.userId"
          class="member-row"
          @click="emit('selectMember', member)"
        >
          <div class="left">
            <div class="name">{{ member.nickname }}</div>
            <div class="meta">
              <span v-if="member.isOwner">房主</span>
              <span v-if="member.userId === props.selfUserId">你</span>
            </div>
          </div>
          <IdentityBadge :identity="member.identity" />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.sheet-wrap {
  position: fixed;
  inset: 0;
  z-index: 56;
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
  max-height: 75dvh;
  overflow: auto;
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

.member-row {
  width: 100%;
  border: 1px solid var(--sheet-item-border);
  border-radius: 12px;
  padding: 11px;
  margin-bottom: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--sheet-item-bg);
  color: var(--text-main);
}

.left {
  text-align: left;
}

.name {
  color: var(--text-main);
}

.meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-sub);
  display: flex;
  gap: 8px;
}
</style>
