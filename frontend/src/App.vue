<template>
  <div class="app-shell">
    <router-view />
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100dvh;
  color: var(--text-main);
  background: radial-gradient(circle at 15% 10%, var(--app-glow-1) 0%, transparent 36%),
    radial-gradient(circle at 85% 0%, var(--app-glow-2) 0%, transparent 32%),
    var(--bg-base);
}
</style>
