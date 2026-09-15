<script setup lang="ts">
type EventItem = {
  id: string
  event_type: string
  endpoint_id: string
  status: string
  external_event_id: string
  created_at: string
}

const { get, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!

const items = ref<EventItem[]>([])
const loading = ref(true)
const status = ref('')
const q = ref('')

const filtered = computed(() => {
  const query = q.value.trim().toLowerCase()
  return items.value.filter((ev) => {
    if (status.value && ev.status !== status.value) return false
    if (!query) return true
    return (
      ev.id.toLowerCase().includes(query) ||
      ev.event_type.toLowerCase().includes(query) ||
      ev.external_event_id.toLowerCase().includes(query)
    )
  })
})

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const query = status.value ? `?limit=100&status=${status.value}` : '?limit=100'
    const res = await get<{ items: EventItem[] }>(`/api/v1/events${query}`)
    items.value = res.items || []
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load events', 'error')
  } finally {
    loading.value = false
  }
}

let timer: ReturnType<typeof setTimeout> | undefined
watch(q, () => {
  clearTimeout(timer)
  timer = setTimeout(() => {}, 200)
})

watch([apiKey, status], () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Events</h1>
      <p class="mt-1 text-sm text-[var(--text-muted)]">Ingested webhook payloads and their delivery state.</p>
    </div>

    <div class="flex flex-wrap gap-2">
      <input v-model="q" class="input max-w-sm" type="search" placeholder="Search by ID or type" />
      <select v-model="status" class="input max-w-[180px]">
        <option value="">All statuses</option>
        <option>PENDING</option>
        <option>PROCESSING</option>
        <option>DELIVERED</option>
        <option>RETRYING</option>
        <option>FAILED</option>
      </select>
    </div>

    <div class="panel">
      <div v-if="loading" class="space-y-2 p-4">
        <div v-for="i in 6" :key="i" class="h-10 animate-pulse rounded bg-[var(--bg-soft)]" />
      </div>

      <EmptyState
        v-else-if="!filtered.length"
        title="No events"
        description="Events appear here after webhook ingestion."
      />

      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Endpoint</th>
              <th>Status</th>
              <th>Created</th>
              <th />
            </tr>
          </thead>
          <tbody>
            <tr v-for="ev in filtered" :key="ev.id">
              <td class="mono text-xs">{{ ev.external_event_id }}</td>
              <td>{{ ev.event_type }}</td>
              <td class="mono text-xs">{{ ev.endpoint_id.slice(0, 8) }}…</td>
              <td><StatusBadge :status="ev.status" /></td>
              <td class="mono text-xs">{{ new Date(ev.created_at).toLocaleString() }}</td>
              <td>
                <NuxtLink :to="`/events/${ev.id}`" class="text-xs text-emerald-400 hover:underline">Open</NuxtLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
