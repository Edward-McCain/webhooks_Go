<script setup lang="ts">
type Endpoint = {
  id: string
  public_id: string
  name: string
  target_url: string
  active: boolean
  rate_limit: number
  created_at: string
}

const { get, post, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!

const items = ref<Endpoint[]>([])
const loading = ref(true)
const open = ref(false)
const createdSecret = ref('')
const form = reactive({ name: '', target_url: '', rate_limit: 100 })

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const res = await get<{ items: Endpoint[] }>('/api/v1/endpoints')
    items.value = res.items || []
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load endpoints', 'error')
  } finally {
    loading.value = false
  }
}

async function createEndpoint() {
  try {
    const res = await post<{ endpoint: Endpoint; secret: string }>('/api/v1/endpoints', form)
    createdSecret.value = res.secret
    toast('Endpoint created', 'success')
    form.name = ''
    form.target_url = ''
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Create failed', 'error')
  }
}

watch(apiKey, () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">Endpoints</h1>
        <p class="mt-1 text-sm text-[var(--text-muted)]">Destinations that receive delivered webhooks.</p>
      </div>
      <button type="button" class="btn-primary" @click="open = true; createdSecret = ''">New endpoint</button>
    </div>

    <div v-if="loading" class="grid gap-3 md:grid-cols-2">
      <div v-for="i in 4" :key="i" class="panel h-36 animate-pulse bg-[var(--bg-soft)]" />
    </div>

    <EmptyState
      v-else-if="!items.length"
      title="No endpoints"
      description="Create an endpoint to start accepting and delivering webhooks."
      action-label="Create endpoint"
      @action="open = true"
    />

    <div v-else class="grid gap-3 md:grid-cols-2">
      <article v-for="ep in items" :key="ep.id" class="panel p-4">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="font-semibold">{{ ep.name }}</h2>
            <p class="mono mt-1 break-all text-xs text-[var(--text-muted)]">{{ ep.target_url }}</p>
          </div>
          <StatusBadge :status="ep.active ? 'active' : 'inactive'" />
        </div>
        <div class="mt-4 grid grid-cols-2 gap-2 text-xs text-[var(--text-dim)]">
          <div>
            <div class="uppercase tracking-[0.08em]">Public ID</div>
            <div class="mono mt-1 text-[var(--text-muted)]">{{ ep.public_id }}</div>
          </div>
          <div>
            <div class="uppercase tracking-[0.08em]">Rate limit</div>
            <div class="mt-1 text-[var(--text-muted)]">{{ ep.rate_limit }}/min</div>
          </div>
        </div>
      </article>
    </div>

    <AppModal :open="open" title="Create endpoint" @close="open = false">
      <form class="space-y-3" @submit.prevent="createEndpoint">
        <div>
          <label class="label">Name</label>
          <input v-model="form.name" class="input" required />
        </div>
        <div>
          <label class="label">Target URL</label>
          <input v-model="form.target_url" class="input" placeholder="https://example.com/hooks" required />
        </div>
        <div>
          <label class="label">Rate limit</label>
          <input v-model.number="form.rate_limit" class="input" type="number" min="1" />
        </div>

        <div v-if="createdSecret" class="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
          <div class="font-medium">Save this secret now. It will not be shown again.</div>
          <code class="mono mt-2 block break-all text-xs">{{ createdSecret }}</code>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button type="button" class="btn-ghost" @click="open = false">Cancel</button>
          <button type="submit" class="btn-primary">Create</button>
        </div>
      </form>
    </AppModal>
  </div>
</template>
