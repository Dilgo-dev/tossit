<script lang="ts">
  import { formatSize, getWsUrl, hashEqual } from '../utils'
  import { deriveKey, decryptChunk } from '../crypto'
  import * as proto from '../protocol'

  let { token = '', initialCode = '' }: { token: string; initialCode: string } = $props()

  type Phase = 'idle' | 'connecting' | 'deriving' | 'receiving' | 'verifying' | 'done' | 'error'

  let inputCode = $state(initialCode)
  let phase = $state<Phase>(initialCode ? 'idle' : 'idle')
  let progress = $state(0)
  let speed = $state(0)
  let fileName = $state('')
  let fileSize = $state(0)
  let received = $state(0)
  let errorMsg = $state('')

  let chunks: Uint8Array[] = []
  let startTime = 0

  $effect(() => {
    if (initialCode && initialCode !== inputCode) {
      inputCode = initialCode
    }
  })

  async function receive() {
    if (!inputCode.trim()) return

    const code = inputCode.trim()
    phase = 'connecting'
    progress = 0
    received = 0
    errorMsg = ''
    chunks = []

    let ws: WebSocket

    try {
      ws = new WebSocket(getWsUrl(token))
      ws.binaryType = 'arraybuffer'

      await new Promise<void>((resolve, reject) => {
        ws.onopen = () => resolve()
        ws.onerror = () => reject(new Error('Connection failed'))
      })

      ws.send(proto.encodeBrowserJoin(code))

      phase = 'deriving'
      const key = await deriveKey(code)

      phase = 'receiving'
      startTime = Date.now()

      await new Promise<void>((resolve, reject) => {
        ws.onmessage = async (event) => {
          try {
            const msg = proto.decode(event.data)

            if (msg.type === proto.MsgError) {
              reject(new Error(new TextDecoder().decode(msg.payload)))
              return
            }

            if (msg.type === proto.MsgStored) {
              // store-and-forward: data follows
              return
            }

            if (msg.type !== proto.MsgData) return

            const payload = msg.payload
            if (payload.length === 0) return
            const peerType = payload[0]

            if (peerType === proto.PeerMetadata) {
              const json = new TextDecoder().decode(payload.slice(1))
              const meta = JSON.parse(json)
              fileName = meta.name
              fileSize = meta.size
              return
            }

            if (peerType === proto.PeerChunk) {
              const { seq, ciphertext } = proto.decodeChunk(payload)
              const plaintext = await decryptChunk(key, seq, ciphertext)
              chunks.push(plaintext)
              received += plaintext.length

              if (fileSize > 0) {
                progress = received / fileSize
              }
              const elapsed = (Date.now() - startTime) / 1000
              speed = elapsed > 0 ? received / elapsed : 0
              return
            }

            if (peerType === proto.PeerDone) {
              phase = 'verifying'
              const expectedHash = proto.decodeDone(payload)

              const allData = new Uint8Array(received)
              let offset = 0
              for (const chunk of chunks) {
                allData.set(chunk, offset)
                offset += chunk.length
              }

              const actualHash = new Uint8Array(
                await crypto.subtle.digest('SHA-256', allData)
              )

              if (!hashEqual(expectedHash, actualHash)) {
                reject(new Error('Hash mismatch - transfer corrupted'))
                return
              }

              const blob = new Blob([allData])
              const url = URL.createObjectURL(blob)
              const a = document.createElement('a')
              a.href = url
              a.download = fileName
              a.click()
              URL.revokeObjectURL(url)

              phase = 'done'
              progress = 1
              resolve()
              ws.close()
            }
          } catch (e: any) {
            reject(e)
          }
        }

        ws.onerror = () => reject(new Error('Connection lost'))
        ws.onclose = (e) => {
          if (phase !== 'done') reject(new Error('Connection closed'))
        }
      })
    } catch (e: any) {
      phase = 'error'
      errorMsg = e.message || 'Transfer failed'
    }
  }

  function reset() {
    inputCode = ''
    phase = 'idle'
    progress = 0
    speed = 0
    fileName = ''
    fileSize = 0
    received = 0
    errorMsg = ''
    chunks = []
  }
</script>

<div class="mx-auto max-w-2xl animate-in">
  <div class="mb-8">
    <h1 class="text-3xl font-bold tracking-tight text-white">
      Receive<span class="text-text-muted">_</span>
    </h1>
    <p class="mt-2 text-sm text-text-dim">
      Enter the transfer code to download files.
    </p>
  </div>

  {#if phase === 'idle' || phase === 'error'}
    <div class="animate-in delay-1 space-y-4">
      <div class="flex gap-3">
        <input
          type="text"
          placeholder="e.g. bold-acorn-42"
          bind:value={inputCode}
          onkeydown={(e) => { if (e.key === 'Enter') receive() }}
          class="flex-1 rounded border border-border bg-surface px-4 py-3 font-mono text-sm text-white placeholder:text-text-muted/50 transition-colors focus:border-accent/40 focus:outline-none"
        />
        <button
          class="rounded bg-accent px-6 py-3 font-mono text-sm font-bold text-bg transition-all hover:bg-accent-dim hover:shadow-[0_0_24px_rgba(163,230,53,0.15)] disabled:opacity-30 disabled:cursor-not-allowed"
          onclick={receive}
          disabled={!inputCode.trim()}
        >
          Receive
        </button>
      </div>

      {#if phase === 'error'}
        <div class="rounded border border-error/30 bg-error/5 px-4 py-3 font-mono text-sm text-error">
          {errorMsg}
        </div>
      {/if}
    </div>

  {:else if phase === 'connecting' || phase === 'deriving'}
    <div class="animate-in rounded-lg border border-border bg-surface p-8 text-center">
      <div class="mb-4 font-mono text-xs uppercase tracking-widest text-accent" style="animation: pulse-glow 1.5s ease-in-out infinite">
        {phase === 'connecting' ? 'Connecting to relay...' : 'Deriving encryption key...'}
      </div>
      <p class="font-mono text-xs text-text-muted">{inputCode}</p>
    </div>

  {:else if phase === 'receiving' || phase === 'verifying'}
    <div class="animate-in space-y-6 rounded-lg border border-border bg-surface p-8">
      {#if fileName}
        <div class="flex items-center justify-between">
          <div>
            <div class="font-mono text-sm text-white">{fileName}</div>
            <div class="font-mono text-xs text-text-muted">{formatSize(fileSize)}</div>
          </div>
          <span class="font-mono text-xs text-accent">
            {phase === 'verifying' ? 'Verifying...' : formatSize(speed) + '/s'}
          </span>
        </div>
      {/if}

      <div class="h-1.5 overflow-hidden rounded-full bg-border">
        <div
          class="h-full rounded-full transition-all duration-300 {progress < 1 ? 'progress-shimmer' : 'bg-accent'}"
          style="width: {Math.round(progress * 100)}%"
        ></div>
      </div>

      <div class="flex justify-between font-mono text-xs text-text-muted">
        <span>{formatSize(received)} / {formatSize(fileSize)}</span>
        <span>{Math.round(progress * 100)}%</span>
      </div>
    </div>

  {:else if phase === 'done'}
    <div class="animate-in space-y-6">
      <div class="rounded-lg border border-accent/20 bg-accent/[0.03] p-8 text-center">
        <div class="mb-2 font-mono text-xs uppercase tracking-widest text-accent">Transfer complete</div>
        <p class="text-sm text-text-dim">
          {fileName} ({formatSize(received)}) - verified and saved
        </p>
      </div>

      <button
        class="w-full rounded border border-border py-2.5 font-mono text-xs uppercase tracking-widest text-text-muted transition-all hover:border-accent hover:text-accent"
        onclick={reset}
      >
        Receive another
      </button>
    </div>
  {/if}
</div>
