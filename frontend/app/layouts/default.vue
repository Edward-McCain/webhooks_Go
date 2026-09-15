<script setup lang="ts">
const route = useRoute()
const config = useRuntimeConfig()
const { apiKey } = useApi()
const sidebarOpen = ref(false)
const showKeyField = ref(false)

const hasDefaultKey = computed(() => Boolean(config.public.defaultApiKey))
const keyReady = computed(() => Boolean(apiKey.value))

const nav = [
  { to: '/', label: 'Dashboard', exact: true },
  { to: '/endpoints', label: 'Endpoints' },
  { to: '/events', label: 'Events' },
  { to: '/deliveries', label: 'Deliveries' },
  { to: '/api-keys', label: 'API Keys' },
  { to: '/analytics', label: 'Analytics' },
  { to: '/settings', label: 'Settings' },
]

function isActive(item: (typeof nav)[number]) {
  if (item.exact) return route.path === item.to
  return route.path.startsWith(item.to)
}
</script>

<template>
  <div class="flex min-h-screen">
    <aside
      class="fixed inset-y-0 left-0 z-40 w-64 border-r border-[var(--border)] bg-[var(--bg-elevated)]/95 backdrop-blur transition-transform lg:static lg:translate-x-0"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <div class="flex h-16 items-center gap-3 border-b border-[var(--border)] px-5">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/15 text-emerald-400">
          <span class="mono text-sm font-semibold">HF</span>
        </div>
        <div>
          <div class="text-sm font-semibold tracking-tight">HookForge</div>
          <div class="text-[11px] text-[var(--text-dim)]">Webhook delivery</div>
        </div>
      </div>

      <nav class="space-y-1 p-3">
        <NuxtLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="block rounded-lg px-3 py-2 text-sm transition"
          :class="isActive(item)
            ? 'bg-white/[0.06] text-white'
            : 'text-[var(--text-muted)] hover:bg-white/[0.03] hover:text-white'"
          @click="sidebarOpen = false"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>
    </aside>

    <div v-if="sidebarOpen" class="fixed inset-0 z-30 bg-black/50 lg:hidden" @click="sidebarOpen = false" />

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-[var(--border)] bg-[var(--bg)]/80 px-4 backdrop-blur lg:px-6">
        <button class="btn-ghost lg:hidden" type="button" aria-label="Open menu" @click="sidebarOpen = true">
          Menu
        </button>

        <div class="hidden flex-1 md:block">
          <input class="input max-w-md" type="search" placeholder="Search events, endpoints..." />
        </div>

        <div class="ml-auto flex items-center gap-2">
          <span class="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1 text-[11px] font-medium text-emerald-300">
            development
          </span>
          <span
            v-if="hasDefaultKey && keyReady && !showKeyField"
            class="rounded-full border border-zinc-700 bg-zinc-900 px-2.5 py-1 text-[11px] text-zinc-400"
          >
            API connected
          </span>
          <button
            v-if="hasDefaultKey && !showKeyField"
            type="button"
            class="btn-ghost text-xs"
            @click="showKeyField = true"
          >
            Change key
          </button>
          <input
            v-if="!hasDefaultKey || showKeyField || !keyReady"
            v-model="apiKey"
            class="input max-w-[220px] mono text-xs"
            type="password"
            placeholder="API key"
            aria-label="API key"
          />
        </div>
      </header>

      <main class="flex-1 px-4 py-6 lg:px-8">
        <slot />
      </main>
    </div>
  </div>
</template>
