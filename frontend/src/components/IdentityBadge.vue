<script setup lang="ts">
import type { Identity } from '@/types'

defineProps<{ identity: Identity }>()

const labels: Record<Identity, string> = {
  normal: 'normal',
  target: 'target',
  immune: 'immune'
}
</script>

<template>
  <span class="badge" :class="identity">{{ labels[identity] }}</span>
</template>

<style scoped>
.badge {
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  border: 1px solid var(--line);
  color: var(--text-sub);
}
.badge.target {
  border-color: transparent;
  color: var(--identity-target-text);
  background: linear-gradient(120deg, var(--identity-target-grad-1), var(--identity-target-grad-2));
  font-weight: 700;
}
.badge.immune {
  border-color: transparent;
  color: var(--identity-immune-text);
  background: linear-gradient(120deg, var(--identity-immune-grad-1), var(--identity-immune-grad-2));
  font-weight: 700;
}
</style>
