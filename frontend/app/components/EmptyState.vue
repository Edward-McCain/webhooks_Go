<script setup lang="ts">
defineProps<{
  title: string
  description?: string
  actionLabel?: string
}>()

defineEmits<{
  action: []
}>()
</script>

<template>
  <div class="panel flex flex-col items-start gap-3 px-6 py-10">
    <h3 class="text-base font-semibold">{{ title }}</h3>
    <p v-if="description" class="max-w-lg text-sm text-[var(--text-muted)]">{{ description }}</p>
    <button v-if="actionLabel" type="button" class="btn-primary" @click="$emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>
