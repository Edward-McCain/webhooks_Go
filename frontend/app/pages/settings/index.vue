<script setup lang="ts">
const form = reactive({
  worker_count: 10,
  request_timeout: '10s',
  max_payload_size: '1048576',
  retry_count: 5,
  rate_limit: 100,
})
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
      <p class="mt-1 text-sm text-[var(--text-muted)]">Runtime defaults for local development.</p>
    </div>

    <div class="grid gap-3 lg:grid-cols-2">
      <section class="panel space-y-4 p-4">
        <h2 class="text-sm font-semibold">General</h2>
        <div>
          <label class="label">Worker count</label>
          <input v-model.number="form.worker_count" class="input" type="number" min="1" />
        </div>
        <div>
          <label class="label">Request timeout</label>
          <input v-model="form.request_timeout" class="input" />
        </div>
        <div>
          <label class="label">Max payload size</label>
          <input v-model="form.max_payload_size" class="input" />
        </div>
      </section>

      <section class="panel space-y-4 p-4">
        <h2 class="text-sm font-semibold">Retry policy & rate limits</h2>
        <div>
          <label class="label">Max retries</label>
          <input v-model.number="form.retry_count" class="input" type="number" min="0" />
        </div>
        <div>
          <label class="label">Default rate limit / minute</label>
          <input v-model.number="form.rate_limit" class="input" type="number" min="1" />
        </div>
        <p class="text-xs text-[var(--text-dim)]">
          These values mirror environment configuration. Persist changes through deployment config,
          not browser state.
        </p>
      </section>
    </div>
  </div>
</template>
