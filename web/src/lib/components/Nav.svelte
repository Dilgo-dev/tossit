<script lang="ts">
  let { currentRoute = '' }: { currentRoute: string } = $props()

  const links = [
    { hash: '', label: 'Send', icon: '>' },
    { hash: 'receive', label: 'Receive', icon: '<' },
    { hash: 'transfers', label: 'Transfers', icon: '#' },
    { hash: 'admin', label: 'Admin', icon: '*' },
  ]
</script>

<nav class="fixed top-0 left-0 right-0 z-50 border-b border-border bg-bg/80 backdrop-blur-xl">
  <div class="mx-auto flex h-14 max-w-5xl items-center justify-between px-5">
    <a href="#/" class="flex items-center gap-2 font-mono text-sm font-bold tracking-tight text-accent">
      tossit<span class="text-text-muted">_</span><span class="text-text-muted/40 text-xs font-normal">relay</span>
    </a>

    <div class="flex items-center gap-1">
      {#each links as link}
        {@const active = currentRoute === link.hash}
        <a
          href="#{link.hash ? '/' + link.hash : '/'}"
          class="group flex items-center gap-1.5 rounded px-3 py-1.5 font-mono text-[11px] uppercase tracking-widest transition-all duration-200
            {active
              ? 'bg-accent-muted/40 text-accent'
              : 'text-text-muted hover:text-text-dim'}"
        >
          <span class="text-[10px] opacity-50 transition-opacity group-hover:opacity-100
            {active ? 'opacity-100' : ''}">{link.icon}</span>
          <span class="hidden sm:inline">{link.label}</span>
        </a>
      {/each}
    </div>
  </div>
</nav>
