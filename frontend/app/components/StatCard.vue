<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  hint?: string
}>()
</script>

<template>
  <div class="panel p-4">
    <div class="text-xs uppercase tracking-[0.08em] text-[var(--text-dim)]">{{ label }}</div>
    <div class="mt-2 text-2xl font-semibold tracking-tight">{{ value }}</div>
    <div v-if="hint" class="mt-1 text-xs text-[var(--text-muted)]">{{ hint }}</div>
  </div>
</template>
