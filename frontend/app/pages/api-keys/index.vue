<script setup lang="ts">
type APIKey = {
  id: string
  name: string
  key_prefix: string
  revoked_at?: string | null
  last_used_at?: string | null
  created_at: string
}

const { get, post, del, apiKey } = useApi()
const toast = inject<(m: string, t?: 'success' | 'error' | 'info') => void>('toast')!

const items = ref<APIKey[]>([])
const loading = ref(true)
const open = ref(false)
const name = ref('')
const createdSecret = ref('')

async function load() {
  if (!apiKey.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const res = await get<{ items: APIKey[] }>('/api/v1/api-keys')
    items.value = res.items || []
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Failed to load API keys', 'error')
  } finally {
    loading.value = false
  }
}

async function createKey() {
  try {
    const res = await post<{ api_key: APIKey; secret: string }>('/api/v1/api-keys', { name: name.value })
    createdSecret.value = res.secret
    toast('API key created', 'success')
    name.value = ''
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Create failed', 'error')
  }
}

async function revoke(id: string) {
  try {
    await del(`/api/v1/api-keys/${id}`)
    toast('API key revoked', 'success')
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Revoke failed', 'error')
  }
}

watch(apiKey, () => load(), { immediate: true })
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">API Keys</h1>
        <p class="mt-1 text-sm text-[var(--text-muted)]">Authenticate management API requests.</p>
      </div>
      <button type="button" class="btn-primary" @click="open = true; createdSecret = ''">Create key</button>
    </div>

    <div class="panel">
      <div v-if="loading" class="space-y-2 p-4">
        <div v-for="i in 4" :key="i" class="h-10 animate-pulse rounded bg-[var(--bg-soft)]" />
      </div>
      <EmptyState v-else-if="!items.length" title="No API keys" description="Create a key to access the management API." />
      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Prefix</th>
              <th>Status</th>
              <th>Last used</th>
              <th>Created</th>
              <th />
            </tr>
          </thead>
          <tbody>
            <tr v-for="k in items" :key="k.id">
              <td>{{ k.name }}</td>
              <td class="mono text-xs">{{ k.key_prefix }}…</td>
              <td>
                <StatusBadge :status="k.revoked_at ? 'inactive' : 'active'" />
              </td>
              <td class="mono text-xs">{{ k.last_used_at ? new Date(k.last_used_at).toLocaleString() : '—' }}</td>
              <td class="mono text-xs">{{ new Date(k.created_at).toLocaleString() }}</td>
              <td>
                <button
                  v-if="!k.revoked_at"
                  type="button"
                  class="btn-danger"
                  @click="revoke(k.id)"
                >
                  Revoke
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <AppModal :open="open" title="Create API key" @close="open = false">
      <form class="space-y-3" @submit.prevent="createKey">
        <div>
          <label class="label">Name</label>
          <input v-model="name" class="input" required />
        </div>
        <div v-if="createdSecret" class="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
          <div class="font-medium">Save this key now. It will not be shown again.</div>
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
