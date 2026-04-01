<script lang="ts">
  import Nav from './lib/components/Nav.svelte'
  import Send from './lib/components/Send.svelte'
  import Receive from './lib/components/Receive.svelte'
  import Transfers from './lib/components/Transfers.svelte'
  import Admin from './lib/components/Admin.svelte'

  // Read auth token from URL params once
  const params = new URLSearchParams(window.location.search)
  const token = params.get('token') || ''

  let hash = $state(window.location.hash)

  $effect(() => {
    const onHash = () => { hash = window.location.hash }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  })

  // Parse route from hash
  let route = $derived((() => {
    const h = hash.replace(/^#\/?/, '')
    if (h.startsWith('receive/')) return 'receive'
    if (h.startsWith('receive')) return 'receive'
    if (h.startsWith('transfers')) return 'transfers'
    if (h.startsWith('admin')) return 'admin'
    if (h.startsWith('send')) return ''
    return ''
  })())

  // Extract code from /#/receive/<code>
  let receiveCode = $derived((() => {
    const h = hash.replace(/^#\/?/, '')
    if (h.startsWith('receive/')) {
      return h.slice('receive/'.length)
    }
    return ''
  })())
</script>

<div class="grain relative min-h-screen">
  <Nav currentRoute={route} />

  <main class="px-5 pt-24 pb-16">
    {#if route === 'receive'}
      {#key receiveCode}
        <Receive {token} initialCode={receiveCode} />
      {/key}
    {:else if route === 'transfers'}
      <Transfers {token} />
    {:else if route === 'admin'}
      <Admin {token} />
    {:else}
      <Send {token} />
    {/if}
  </main>

  <!-- Footer -->
  <footer class="border-t border-border py-6 text-center">
    <p class="font-mono text-[10px] uppercase tracking-widest text-text-muted/50">
      tossit relay - end-to-end encrypted file transfer
    </p>
  </footer>
</div>
