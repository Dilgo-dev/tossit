<script lang="ts">
  import { formatSize, getApiUrl } from '../utils'

  let { token = '' }: { token: string } = $props()

  interface Metrics {
    active_sessions: number
    stored_transfers: number
    transfers_completed: number
    bytes_relayed: number
    errors_total: number
    storage_used_bytes: number
  }

  let metrics = $state<Metrics | null>(null)
  let error = $state('')
  let loading = $state(true)

  async function fetchMetrics() {
    try {
      const res = await fetch(getApiUrl('/metrics', token))
      if (!res.ok) throw new Error(`${res.status}`)
      metrics = await res.json()
      error = ''
    } catch (e: any) {
      error = e.message || 'Failed to fetch'
    } finally {
      loading = false
    }
  }

  $effect(() => {
    fetchMetrics()
    const interval = setInterval(fetchMetrics, 5000)
    return () => clearInterval(interval)
  })

  const cards = $derived(
    metrics
      ? [
          { label: 'Active Sessions', value: String(metrics.active_sessions), accent: true },
          { label: 'Stored Transfers', value: String(metrics.stored_transfers), accent: false },
          { label: 'Completed', value: String(metrics.transfers_completed), accent: false },
          { label: 'Bytes Relayed', value: formatSize(metrics.bytes_relayed), accent: false },
          { label: 'Storage Used', value: formatSize(metrics.storage_used_bytes), accent: false },
          { label: 'Errors', value: String(metrics.errors_total), accent: metrics.errors_total > 0 },
        ]
      : []
  )
</script>

<div class="mx-auto max-w-2xl animate-in">
  <div class="mb-8">
    <h1 class="text-3xl font-bold tracking-tight text-white">
      Transfers<span class="text-text-muted">_</span>
    </h1>
    <p class="mt-2 text-sm text-text-dim">
      Live relay statistics. Auto-refreshes every 5 seconds.
    </p>
  </div>

  {#if loading}
    <div class="animate-in rounded-lg border border-border bg-surface p-8 text-center">
      <div class="font-mono text-xs uppercase tracking-widest text-text-muted" style="animation: pulse-glow 1.5s ease-in-out infinite">
        Loading metrics...
      </div>
    </div>
  {:else if error}
    <div class="rounded border border-error/30 bg-error/5 px-4 py-3 font-mono text-sm text-error">
      Failed to fetch metrics: {error}
    </div>
  {:else}
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
      {#each cards as card, i}
        <div class="animate-in rounded-lg border border-border bg-surface p-5 transition-colors hover:border-border-hover"
             style="animation-delay: {i * 0.05}s">
          <div class="font-mono text-[10px] uppercase tracking-widest text-text-muted">
            {card.label}
          </div>
          <div class="mt-2 font-mono text-2xl font-bold {card.accent ? 'text-accent' : 'text-white'}">
            {card.value}
          </div>
        </div>
      {/each}
    </div>

    {#if metrics}
      <!-- Activity bar -->
      <div class="mt-6 animate-in delay-3 rounded-lg border border-border bg-surface p-5">
        <div class="mb-3 flex items-center justify-between">
          <span class="font-mono text-[10px] uppercase tracking-widest text-text-muted">Storage utilization</span>
          <span class="font-mono text-xs text-text-dim">{formatSize(metrics.storage_used_bytes)}</span>
        </div>
        <div class="h-1.5 overflow-hidden rounded-full bg-border">
          <div
            class="h-full rounded-full bg-accent transition-all duration-500"
            style="width: {Math.min(100, metrics.storage_used_bytes / (1024 * 1024 * 1024 * 2) * 100)}%"
          ></div>
        </div>
      </div>
    {/if}
  {/if}
</div>
