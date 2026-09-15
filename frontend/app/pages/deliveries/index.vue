<script setup lang="ts">
type Delivery = {
  id: string
  event_id: string
  endpoint_id: string
  status: string
  attempt_count: number
  response_status?: number
  created_at: string
}

const { get, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!
const items = ref<Delivery[]>([])
const loading = ref(true)
const status = ref('')

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const q = status.value ? `?limit=100&status=${status.value}` : '?limit=100'
    const res = await get<{ items: Delivery[] }>(`/api/v1/deliveries${q}`)
    items.value = res.items || []
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load deliveries', 'error')
  } finally {
    loading.value = false
  }
}

watch([apiKey, status], () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Deliveries</h1>
        <p class="mt-1 text-sm text-[var(--text-muted)]">Outbound delivery attempts and outcomes.</p>
      </div>
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
      <EmptyState v-else-if="!items.length" title="No deliveries" description="Deliveries are created when events are ingested." />
      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>Delivery</th>
              <th>Event</th>
              <th>Status</th>
              <th>Attempts</th>
              <th>Response</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in items" :key="d.id">
              <td class="mono text-xs">{{ d.id.slice(0, 8) }}…</td>
              <td>
                <NuxtLink :to="`/events/${d.event_id}`" class="mono text-xs text-emerald-400 hover:underline">
                  {{ d.event_id.slice(0, 8) }}…
                </NuxtLink>
              </td>
              <td><StatusBadge :status="d.status" /></td>
              <td>{{ d.attempt_count }}</td>
              <td>{{ d.response_status || '—' }}</td>
              <td class="mono text-xs">{{ new Date(d.created_at).toLocaleString() }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
