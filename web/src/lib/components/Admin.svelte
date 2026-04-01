<script lang="ts">
  import { getApiUrl } from '../utils'

  let { token = '' }: { token: string } = $props()

  interface Health {
    status: string
    version: string
    uptime: string
  }

  interface Config {
    port: string
    storage: string
    expire: string
    max_size: string
    rate_limit: number
  }

  let health = $state<Health | null>(null)
  let config = $state<Config | null>(null)
  let error = $state('')
  let loading = $state(true)
  let needsAuth = $state(false)
  let adminPass = $state('')
  let authError = $state('')

  async function fetchData() {
    try {
      const healthRes = await fetch(getApiUrl('/health', token))
      if (!healthRes.ok) throw new Error(`Health: ${healthRes.status}`)
      health = await healthRes.json()

      const configRes = await fetch(getApiUrl('/api/config', token))
      if (configRes.status === 401) {
        needsAuth = true
        loading = false
        return
      }
      if (configRes.ok) {
        config = await configRes.json()
        needsAuth = false
      }

      error = ''
    } catch (e: any) {
      error = e.message || 'Failed to fetch'
    } finally {
      loading = false
    }
  }

  async function loginAdmin() {
    authError = ''
    const res = await fetch('/api/login/admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'Accept': 'application/json' },
      body: 'password=' + encodeURIComponent(adminPass),
    })
    if (!res.ok) {
      authError = 'Invalid password'
      return
    }
    needsAuth = false
    adminPass = ''
    fetchData()
  }

  $effect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  })
</script>

<div class="mx-auto max-w-2xl animate-in">
  <div class="mb-8">
    <h1 class="text-3xl font-bold tracking-tight text-white">
      Admin<span class="text-text-muted">_</span>
    </h1>
    <p class="mt-2 text-sm text-text-dim">
      Relay health and configuration.
    </p>
  </div>

  {#if loading}
    <div class="animate-in rounded-lg border border-border bg-surface p-8 text-center">
      <div class="font-mono text-xs uppercase tracking-widest text-text-muted" style="animation: pulse-glow 1.5s ease-in-out infinite">
        Loading...
      </div>
    </div>
  {:else if needsAuth}
    <div class="animate-in rounded-lg border border-border bg-surface p-8">
      <div class="mb-4 font-mono text-xs uppercase tracking-widest text-text-muted">Admin authentication required</div>
      <div class="flex gap-3">
        <input
          type="password"
          placeholder="admin password"
          bind:value={adminPass}
          onkeydown={(e) => { if (e.key === 'Enter') loginAdmin() }}
          class="flex-1 rounded border border-border bg-bg px-4 py-2.5 font-mono text-sm text-white placeholder:text-text-muted/50 transition-colors focus:border-accent/40 focus:outline-none"
        />
        <button
          class="rounded bg-accent px-5 py-2.5 font-mono text-sm font-bold text-bg transition-all hover:bg-accent-dim"
          onclick={loginAdmin}
        >Enter</button>
      </div>
      {#if authError}
        <p class="mt-3 font-mono text-xs text-error">{authError}</p>
      {/if}
    </div>

    <!-- Still show health if available -->
    {#if health}
      <div class="mt-4 animate-in rounded-lg border border-border bg-surface overflow-hidden">
        <div class="border-b border-border px-5 py-3">
          <span class="font-mono text-[10px] uppercase tracking-widest text-text-muted">Health</span>
        </div>
        <div class="divide-y divide-border">
          <div class="flex items-center justify-between px-5 py-3">
            <span class="text-sm text-text-dim">Status</span>
            <span class="flex items-center gap-2 font-mono text-sm">
              <span class="h-2 w-2 rounded-full {health.status === 'ok' ? 'bg-success' : 'bg-error'}"></span>
              <span class="{health.status === 'ok' ? 'text-success' : 'text-error'}">{health.status}</span>
            </span>
          </div>
          <div class="flex items-center justify-between px-5 py-3">
            <span class="text-sm text-text-dim">Version</span>
            <span class="font-mono text-sm text-white">{health.version}</span>
          </div>
        </div>
      </div>
    {/if}
  {:else if error}
    <div class="rounded border border-error/30 bg-error/5 px-4 py-3 font-mono text-sm text-error">
      {error}
    </div>
  {:else}
    <div class="space-y-4">
      {#if health}
        <div class="animate-in rounded-lg border border-border bg-surface overflow-hidden">
          <div class="border-b border-border px-5 py-3">
            <span class="font-mono text-[10px] uppercase tracking-widest text-text-muted">Health</span>
          </div>
          <div class="divide-y divide-border">
            <div class="flex items-center justify-between px-5 py-3">
              <span class="text-sm text-text-dim">Status</span>
              <span class="flex items-center gap-2 font-mono text-sm">
                <span class="h-2 w-2 rounded-full {health.status === 'ok' ? 'bg-success' : 'bg-error'}"></span>
                <span class="{health.status === 'ok' ? 'text-success' : 'text-error'}">{health.status}</span>
              </span>
            </div>
            <div class="flex items-center justify-between px-5 py-3">
              <span class="text-sm text-text-dim">Version</span>
              <span class="font-mono text-sm text-white">{health.version}</span>
            </div>
            <div class="flex items-center justify-between px-5 py-3">
              <span class="text-sm text-text-dim">Uptime</span>
              <span class="font-mono text-sm text-text">{health.uptime}</span>
            </div>
          </div>
        </div>
      {/if}

      {#if config}
        <div class="animate-in rounded-lg border border-border bg-surface overflow-hidden">
          <div class="border-b border-border px-5 py-3">
            <span class="font-mono text-[10px] uppercase tracking-widest text-text-muted">Configuration</span>
          </div>
          <div class="divide-y divide-border">
            {#each [
              { label: 'Port', value: config.port },
              { label: 'Storage', value: config.storage },
              { label: 'Expiry', value: config.expire },
              { label: 'Max Size', value: config.max_size },
              { label: 'Rate Limit', value: config.rate_limit > 0 ? `${config.rate_limit}/min` : 'off' },
            ] as row}
              <div class="flex items-center justify-between px-5 py-3">
                <span class="text-sm text-text-dim">{row.label}</span>
                <span class="font-mono text-sm text-text">{row.value}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <div class="animate-in rounded-lg border border-border bg-surface p-5">
        <div class="mb-3 font-mono text-[10px] uppercase tracking-widest text-text-muted">Relay endpoint</div>
        <code class="block rounded bg-bg px-4 py-2.5 font-mono text-xs text-accent/70">
          {location.origin}/ws
        </code>
      </div>
    </div>
  {/if}
</div>
