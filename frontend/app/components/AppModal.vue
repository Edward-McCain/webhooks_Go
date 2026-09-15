<script setup lang="ts">
defineProps<{
  open: boolean
  title: string
}>()

defineEmits<{
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-end justify-center bg-black/60 p-4 sm:items-center">
      <div class="panel w-full max-w-lg p-5 shadow-[var(--shadow)]" role="dialog" aria-modal="true">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 class="text-lg font-semibold">{{ title }}</h2>
          <button type="button" class="btn-ghost" @click="$emit('close')">Close</button>
        </div>
        <slot />
      </div>
    </div>
  </Teleport>
</template>
