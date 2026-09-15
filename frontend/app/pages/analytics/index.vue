<script setup lang="ts">
type Stats = {
  total_events: number
  delivered: number
  failed: number
  retrying: number
  success_rate: number
  avg_delivery_time_ms: number
}

const { get, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!
const period = ref('7d')
const stats = ref<Stats | null>(null)
const loading = ref(true)

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    stats.value = await get<Stats>('/api/v1/stats')
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load analytics', 'error')
  } finally {
    loading.value = false
  }
}

watch([apiKey, period], () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Analytics</h1>
        <p class="mt-1 text-sm text-[var(--text-muted)]">Delivery performance over time.</p>
      </div>
      <div class="flex gap-2">
        <button
          v-for="p in ['24h', '7d', '30d', '90d']"
          :key="p"
          type="button"
          class="btn"
          :class="period === p ? 'btn-primary' : 'btn-ghost'"
          @click="period = p"
        >
          {{ p }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="grid gap-3 md:grid-cols-3">
      <div v-for="i in 3" :key="i" class="panel h-28 animate-pulse bg-[var(--bg-soft)]" />
    </div>

    <template v-else>
      <div class="grid gap-3 md:grid-cols-3">
        <StatCard label="Events" :value="stats?.total_events ?? '—'" :hint="`Window ${period}`" />
        <StatCard label="Success rate" :value="`${(stats?.success_rate ?? 0).toFixed(1)}%`" />
        <StatCard label="Avg latency" :value="`${(stats?.avg_delivery_time_ms ?? 0).toFixed(0)} ms`" />
      </div>

      <div class="grid gap-3 lg:grid-cols-2">
        <div class="panel p-4">
          <h2 class="mb-4 text-sm font-semibold">Delivery mix</h2>
          <div class="space-y-3 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-[var(--text-muted)]">Delivered</span>
              <span>{{ stats?.delivered ?? 0 }}</span>
            </div>
            <div class="h-2 overflow-hidden rounded-full bg-[var(--bg-soft)]">
              <div
                class="h-full bg-emerald-500"
                :style="{ width: `${Math.min(100, stats?.success_rate || 0)}%` }"
              />
            </div>
            <div class="flex items-center justify-between">
              <span class="text-[var(--text-muted)]">Failed</span>
              <span>{{ stats?.failed ?? 0 }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-[var(--text-muted)]">Retrying</span>
              <span>{{ stats?.retrying ?? 0 }}</span>
            </div>
          </div>
        </div>

        <div class="panel p-4">
          <h2 class="mb-4 text-sm font-semibold">Notes</h2>
          <p class="text-sm leading-relaxed text-[var(--text-muted)]">
            Live counters come from PostgreSQL aggregates. Period filters are ready for time-series
            endpoints; Grafana remains the source of high-resolution latency and retry charts.
          </p>
        </div>
      </div>
    </template>
  </div>
</template>
