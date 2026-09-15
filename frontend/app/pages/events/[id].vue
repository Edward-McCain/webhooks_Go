<script setup lang="ts">
const route = useRoute()
const { get, post, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!

const loading = ref(true)
const data = ref<{
  event: {
    id: string
    external_event_id: string
    event_type: string
    status: string
    created_at: string
    payload: unknown
  }
  delivery?: { id: string; status: string }
  attempts?: Array<{
    id: string
    attempt_number: number
    status: string
    response_status?: number
    duration_ms: number
    created_at: string
    error?: string
    response_body?: string
  }>
} | null>(null)

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    data.value = await get(`/api/v1/events/${route.params.id}`)
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load event', 'error')
  } finally {
    loading.value = false
  }
}

async function retry() {
  try {
    await post(`/api/v1/events/${route.params.id}/retry`)
    toast('Retry queued', 'success')
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Retry failed', 'error')
  }
}

const payloadText = computed(() => {
  try {
    return JSON.stringify(data.value?.event?.payload ?? {}, null, 2)
  } catch {
    return String(data.value?.event?.payload ?? '')
  }
})

watch(apiKey, () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <NuxtLink to="/events" class="text-xs text-[var(--text-dim)] hover:text-white">← Events</NuxtLink>
        <h1 class="mt-2 text-2xl font-semibold tracking-tight">Event details</h1>
      </div>
      <button
        v-if="data?.event?.status === 'FAILED'"
        type="button"
        class="btn-primary"
        @click="retry"
      >
        Retry delivery
      </button>
    </div>

    <div v-if="loading" class="panel h-64 animate-pulse bg-[var(--bg-soft)]" />

    <template v-else-if="data?.event">
      <section class="grid gap-3 lg:grid-cols-2">
        <div class="panel space-y-3 p-4">
          <div class="flex items-center justify-between">
            <h2 class="text-sm font-semibold">Overview</h2>
            <StatusBadge :status="data.event.status" />
          </div>
          <dl class="space-y-2 text-sm">
            <div class="flex justify-between gap-3">
              <dt class="text-[var(--text-dim)]">Event ID</dt>
              <dd class="mono text-xs">{{ data.event.id }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-[var(--text-dim)]">External ID</dt>
              <dd class="mono text-xs">{{ data.event.external_event_id }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-[var(--text-dim)]">Type</dt>
              <dd>{{ data.event.event_type }}</dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt class="text-[var(--text-dim)]">Created</dt>
              <dd class="mono text-xs">{{ new Date(data.event.created_at).toLocaleString() }}</dd>
            </div>
          </dl>
        </div>

        <div class="panel p-4">
          <h2 class="mb-3 text-sm font-semibold">Timeline</h2>
          <ol class="space-y-3 text-sm">
            <li class="flex gap-3">
              <span class="mt-1 h-2 w-2 rounded-full bg-sky-400" />
              <div>
                <div>Received</div>
                <div class="text-xs text-[var(--text-dim)]">{{ new Date(data.event.created_at).toLocaleString() }}</div>
              </div>
            </li>
            <li v-for="a in data.attempts || []" :key="a.id" class="flex gap-3">
              <span
                class="mt-1 h-2 w-2 rounded-full"
                :class="a.status === 'SUCCESS' ? 'bg-emerald-400' : 'bg-rose-400'"
              />
              <div>
                <div>Attempt {{ a.attempt_number }} · {{ a.status }}</div>
                <div class="text-xs text-[var(--text-dim)]">
                  {{ new Date(a.created_at).toLocaleString() }}
                  <span v-if="a.response_status"> · HTTP {{ a.response_status }}</span>
                  · {{ a.duration_ms }}ms
                </div>
                <div v-if="a.error" class="mt-1 text-xs text-rose-300">{{ a.error }}</div>
              </div>
            </li>
            <li v-if="data.event.status === 'DELIVERED'" class="flex gap-3">
              <span class="mt-1 h-2 w-2 rounded-full bg-emerald-400" />
              <div>Delivered</div>
            </li>
          </ol>
        </div>
      </section>

      <section class="panel p-4">
        <h2 class="mb-3 text-sm font-semibold">Payload</h2>
        <pre class="mono overflow-x-auto rounded-lg bg-black/40 p-3 text-xs text-[var(--text-muted)]">{{ payloadText }}</pre>
      </section>
    </template>
  </div>
</template>
