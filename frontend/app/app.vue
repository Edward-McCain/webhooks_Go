<script setup lang="ts">
const toasts = useState<Array<{ id: number; message: string; type: 'success' | 'error' | 'info' }>>('toasts', () => [])
const { apiKey } = useApi()

onMounted(() => {
  const saved = localStorage.getItem('hookforge_api_key')
  if (saved) apiKey.value = saved
})

watch(apiKey, (v) => {
  if (import.meta.client) localStorage.setItem('hookforge_api_key', v || '')
})

function pushToast(message: string, type: 'success' | 'error' | 'info' = 'info') {
  const id = Date.now()
  toasts.value.push({ id, message, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }, 3200)
}

provide('toast', pushToast)
</script>

<template>
  <div class="min-h-screen text-[var(--text)]">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>

    <div class="pointer-events-none fixed bottom-4 right-4 z-50 flex w-[320px] flex-col gap-2">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="pointer-events-auto rounded-lg border px-3 py-2 text-sm shadow-[var(--shadow)]"
        :class="{
          'border-emerald-500/30 bg-emerald-500/10 text-emerald-200': t.type === 'success',
          'border-rose-500/30 bg-rose-500/10 text-rose-200': t.type === 'error',
          'border-[var(--border)] bg-[var(--bg-elevated)] text-[var(--text)]': t.type === 'info',
        }"
      >
        {{ t.message }}
      </div>
    </div>
  </div>
</template>
