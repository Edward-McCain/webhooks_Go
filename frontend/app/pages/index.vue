<script setup lang="ts">
type Stats = {
  total_events: number
  delivered: number
  failed: number
  retrying: number
  pending: number
  processing: number
  success_rate: number
  avg_delivery_time_ms: number
}

type EventItem = {
  id: string
  event_type: string
  endpoint_id: string
  status: string
  created_at: string
}

const { get, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!

const loading = ref(true)
const stats = ref<Stats | null>(null)
const events = ref<EventItem[]>([])
const error = ref('')

async function load() {
  if (!apiKey.value) {
    loading.value = false
    error.value = 'Paste your API key in the top bar to load live data.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const [s, e] = await Promise.all([
      get<Stats>('/api/v1/stats'),
      get<{ items: EventItem[] }>('/api/v1/events?limit=8'),
    ])
    stats.value = s
    events.value = e.items || []
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load dashboard'
    toast(error.value, 'error')
  } finally {
    loading.value = false
  }
}

watch(apiKey, () => load(), { immediate: true })

function fmt(n?: number) {
  if (n == null) return '—'
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
      <p class="mt-1 text-sm text-[var(--text-muted)]">Delivery health across all endpoints.</p>
    </div>

    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="i in 6" :key="i" class="panel h-24 animate-pulse bg-[var(--bg-soft)]" />
    </div>

    <template v-else>
      <p v-if="error" class="text-sm text-amber-300">{{ error }}</p>

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <StatCard label="Total events" :value="fmt(stats?.total_events)" />
        <StatCard label="Delivered" :value="fmt(stats?.delivered)" />
        <StatCard label="Failed" :value="fmt(stats?.failed)" />
        <StatCard label="Retrying" :value="fmt(stats?.retrying)" />
        <StatCard label="Success rate" :value="`${fmt(stats?.success_rate)}%`" />
        <StatCard label="Avg delivery" :value="`${fmt(stats?.avg_delivery_time_ms)} ms`" />
      </div>

      <section class="panel">
        <div class="flex items-center justify-between border-b border-[var(--border)] px-4 py-3">
          <h2 class="text-sm font-semibold">Recent events</h2>
          <NuxtLink to="/events" class="text-xs text-emerald-400 hover:text-emerald-300">View all</NuxtLink>
        </div>

        <EmptyState
          v-if="!events.length"
          title="No events yet"
          description="Send a signed webhook to an endpoint public ID to see activity here."
        />

        <div v-else class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>Event</th>
                <th>Type</th>
                <th>Status</th>
                <th>Created</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ev in events" :key="ev.id">
                <td>
                  <NuxtLink :to="`/events/${ev.id}`" class="mono text-xs text-emerald-400 hover:underline">
                    {{ ev.id.slice(0, 8) }}…
                  </NuxtLink>
                </td>
                <td>{{ ev.event_type }}</td>
                <td><StatusBadge :status="ev.status" /></td>
                <td class="mono text-xs">{{ new Date(ev.created_at).toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>
