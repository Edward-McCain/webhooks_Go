<script setup lang="ts">
defineProps<{
  status: string
}>()

const styles: Record<string, string> = {
  DELIVERED: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  FAILED: 'border-rose-500/30 bg-rose-500/10 text-rose-300',
  RETRYING: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
  PENDING: 'border-sky-500/30 bg-sky-500/10 text-sky-300',
  PROCESSING: 'border-violet-500/30 bg-violet-500/10 text-violet-300',
  SUCCESS: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  TIMEOUT: 'border-amber-500/30 bg-amber-500/10 text-amber-300',
  active: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300',
  inactive: 'border-zinc-500/30 bg-zinc-500/10 text-zinc-300',
}
</script>

<template>
  <span
    class="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide"
    :class="styles[status] || 'border-[var(--border)] bg-[var(--bg-soft)] text-[var(--text-muted)]'"
  >
    {{ status }}
  </span>
</template>
