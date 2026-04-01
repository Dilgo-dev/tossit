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

  async function fetchData() {
    try {
      const [healthRes, configRes] = await Promise.all([
        fetch(getApiUrl('/health', token)),
        fetch(getApiUrl('/api/config', token)).catch(() => null),
      ])

      if (!healthRes.ok) throw new Error(`Health: ${healthRes.status}`)
      health = await healthRes.json()

      if (configRes?.ok) {
        config = await configRes.json()
      }

      error = ''
    } catch (e: any) {
      error = e.message || 'Failed to fetch'
    } finally {
      loading = false
    }
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
  {:else if error}
    <div class="rounded border border-error/30 bg-error/5 px-4 py-3 font-mono text-sm text-error">
      {error}
    </div>
  {:else}
    <div class="space-y-4">
      <!-- Health status -->
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

      <!-- Config -->
      {#if config}
        <div class="animate-in delay-1 rounded-lg border border-border bg-surface overflow-hidden">
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

      <!-- Connection info -->
      <div class="animate-in delay-2 rounded-lg border border-border bg-surface p-5">
        <div class="mb-3 font-mono text-[10px] uppercase tracking-widest text-text-muted">Relay endpoint</div>
        <code class="block rounded bg-bg px-4 py-2.5 font-mono text-xs text-accent/70">
          {location.origin}/ws
        </code>
      </div>
    </div>
  {/if}
</div>
