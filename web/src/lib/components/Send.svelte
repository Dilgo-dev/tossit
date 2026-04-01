<script lang="ts">
  import { formatSize, getWsUrl } from '../utils'
  import { deriveKey, encryptChunk, sha256, CHUNK_SIZE } from '../crypto'
  import * as proto from '../protocol'

  let { token = '' }: { token: string } = $props()

  type Phase = 'idle' | 'encrypting' | 'uploading' | 'done' | 'error'

  let files = $state<File[]>([])
  let phase = $state<Phase>('idle')
  let progress = $state(0)
  let speed = $state(0)
  let code = $state('')
  let errorMsg = $state('')
  let dragOver = $state(false)
  let copied = $state(false)

  let totalSize = $derived(files.reduce((s, f) => s + f.size, 0))
  let browserUrl = $derived(
    code ? `${location.origin}/d/${code}${token ? '?token=' + encodeURIComponent(token) : ''}` : ''
  )

  function onDrop(e: DragEvent) {
    e.preventDefault()
    dragOver = false
    if (e.dataTransfer?.files) {
      files = [...files, ...Array.from(e.dataTransfer.files)]
    }
  }

  function onFileInput(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files) {
      files = [...files, ...Array.from(input.files)]
    }
  }

  function removeFile(index: number) {
    files = files.filter((_, i) => i !== index)
  }

  async function send() {
    if (files.length === 0) return

    phase = 'encrypting'
    progress = 0
    errorMsg = ''
    code = proto.generateCode()

    let ws: WebSocket

    try {
      ws = new WebSocket(getWsUrl(token))
      ws.binaryType = 'arraybuffer'

      await new Promise<void>((resolve, reject) => {
        ws.onopen = () => resolve()
        ws.onerror = () => reject(new Error('Connection failed'))
      })

      ws.send(proto.encodeRegister(code, false))

      const key = await deriveKey(code)
      phase = 'uploading'

      const file = files.length === 1 ? files[0] : null
      const name = file ? file.name : 'archive'
      const size = file ? file.size : totalSize

      const meta = proto.encodeMetadata({
        name,
        size,
        mode: 0o644,
        is_dir: false,
        chunk_size: CHUNK_SIZE,
      })
      ws.send(proto.encode(proto.MsgData, meta))

      let seq = 0
      let sent = 0
      const startTime = Date.now()
      const hasher = new Uint8Array(0)
      const allChunks: Uint8Array[] = []

      for (const f of files) {
        const buffer = await f.arrayBuffer()
        const data = new Uint8Array(buffer)

        for (let offset = 0; offset < data.length; offset += CHUNK_SIZE) {
          const chunk = data.slice(offset, Math.min(offset + CHUNK_SIZE, data.length))
          allChunks.push(chunk)

          const ciphertext = await encryptChunk(key, seq, chunk)
          const encoded = proto.encodeChunk(seq, ciphertext)
          ws.send(proto.encode(proto.MsgData, encoded))

          seq++
          sent += chunk.length
          progress = sent / size

          const elapsed = (Date.now() - startTime) / 1000
          speed = elapsed > 0 ? sent / elapsed : 0
        }
      }

      const allData = new Uint8Array(sent)
      let offset = 0
      for (const chunk of allChunks) {
        allData.set(chunk, offset)
        offset += chunk.length
      }
      const hash = await sha256(allData)
      const done = proto.encodeDone(hash)
      ws.send(proto.encode(proto.MsgData, done))

      await new Promise<void>((resolve, reject) => {
        ws.onmessage = (event) => {
          const msg = proto.decode(event.data)
          if (msg.type === proto.MsgStored) resolve()
          else if (msg.type === proto.MsgError) reject(new Error(new TextDecoder().decode(msg.payload)))
        }
        ws.onerror = () => reject(new Error('Connection lost'))
      })

      phase = 'done'
      progress = 1
    } catch (e: any) {
      phase = 'error'
      errorMsg = e.message || 'Transfer failed'
    }
  }

  function copyCode() {
    navigator.clipboard.writeText(code)
    copied = true
    setTimeout(() => (copied = false), 2000)
  }

  function copyUrl() {
    navigator.clipboard.writeText(browserUrl)
    copied = true
    setTimeout(() => (copied = false), 2000)
  }

  function reset() {
    files = []
    phase = 'idle'
    progress = 0
    speed = 0
    code = ''
    errorMsg = ''
  }
</script>

<div class="mx-auto max-w-2xl animate-in">
  <div class="mb-8">
    <h1 class="text-3xl font-bold tracking-tight text-white">
      Send<span class="text-text-muted">_</span>
    </h1>
    <p class="mt-2 text-sm text-text-dim">
      Files are end-to-end encrypted. The relay never sees your data.
    </p>
  </div>

  {#if phase === 'idle' || phase === 'error'}
    <!-- Drop zone -->
    <div
      class="animate-in delay-1 relative cursor-pointer rounded-lg border-2 border-dashed border-border transition-all duration-300 hover:border-border-hover {dragOver ? 'drop-active' : ''}"
      role="button"
      tabindex="0"
      ondragover={(e) => { e.preventDefault(); dragOver = true }}
      ondragleave={() => (dragOver = false)}
      ondrop={onDrop}
      onclick={() => document.getElementById('file-input')?.click()}
      onkeydown={(e) => { if (e.key === 'Enter') document.getElementById('file-input')?.click() }}
    >
      <div class="flex flex-col items-center justify-center py-16 px-6">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-surface text-2xl text-accent">
          {#if dragOver}
            +
          {:else}
            >
          {/if}
        </div>
        <p class="text-sm font-medium text-text-dim">
          {#if dragOver}
            Drop files here
          {:else}
            Drag files here or click to browse
          {/if}
        </p>
        <p class="mt-1 font-mono text-xs text-text-muted">
          Any file type, up to relay limit
        </p>
      </div>
      <input
        id="file-input"
        type="file"
        multiple
        class="hidden"
        onchange={onFileInput}
      />
    </div>

    <!-- File list -->
    {#if files.length > 0}
      <div class="mt-4 animate-in delay-2 space-y-2">
        {#each files as file, i}
          <div class="flex items-center justify-between rounded border border-border bg-surface px-4 py-3">
            <div class="flex items-center gap-3 overflow-hidden">
              <span class="font-mono text-xs text-accent">{String(i + 1).padStart(2, '0')}</span>
              <span class="truncate text-sm text-text">{file.name}</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="whitespace-nowrap font-mono text-xs text-text-muted">{formatSize(file.size)}</span>
              <button
                class="text-text-muted transition-colors hover:text-error"
                onclick={() => removeFile(i)}
              >x</button>
            </div>
          </div>
        {/each}

        <div class="flex items-center justify-between pt-2">
          <span class="font-mono text-xs text-text-muted">
            {files.length} file{files.length > 1 ? 's' : ''} / {formatSize(totalSize)}
          </span>
          <button
            class="rounded bg-accent px-5 py-2.5 font-mono text-sm font-bold text-bg transition-all hover:bg-accent-dim hover:shadow-[0_0_24px_rgba(163,230,53,0.15)]"
            onclick={send}
          >
            Send
          </button>
        </div>
      </div>
    {/if}

    {#if phase === 'error'}
      <div class="mt-4 rounded border border-error/30 bg-error/5 px-4 py-3 font-mono text-sm text-error">
        {errorMsg}
      </div>
    {/if}

  {:else if phase === 'encrypting'}
    <div class="animate-in rounded-lg border border-border bg-surface p-8 text-center">
      <div class="mb-4 font-mono text-xs uppercase tracking-widest text-accent pulse-glow" style="animation: pulse-glow 1.5s ease-in-out infinite">
        Deriving encryption key...
      </div>
      <p class="text-sm text-text-muted">Using Argon2id key derivation</p>
    </div>

  {:else if phase === 'uploading'}
    <div class="animate-in space-y-6 rounded-lg border border-border bg-surface p-8">
      <div class="flex items-center justify-between">
        <span class="font-mono text-xs uppercase tracking-widest text-accent">Uploading</span>
        <span class="font-mono text-xs text-text-muted">{formatSize(speed)}/s</span>
      </div>

      <div class="h-1.5 overflow-hidden rounded-full bg-border">
        <div
          class="h-full rounded-full transition-all duration-300 {progress < 1 ? 'progress-shimmer' : 'bg-accent'}"
          style="width: {Math.round(progress * 100)}%"
        ></div>
      </div>

      <div class="flex justify-between font-mono text-xs text-text-muted">
        <span>{formatSize(progress * totalSize)} / {formatSize(totalSize)}</span>
        <span>{Math.round(progress * 100)}%</span>
      </div>
    </div>

  {:else if phase === 'done'}
    <div class="animate-in space-y-6">
      <div class="rounded-lg border border-accent/20 bg-accent/[0.03] p-8">
        <div class="mb-6 text-center">
          <div class="mb-2 font-mono text-xs uppercase tracking-widest text-accent">Transfer ready</div>
          <p class="text-sm text-text-dim">Share this code with the receiver</p>
        </div>

        <!-- Code display -->
        <div class="group relative mb-4">
          <button
            class="w-full rounded border border-border bg-bg px-6 py-4 text-center font-mono text-2xl font-bold tracking-wide text-white transition-all hover:border-accent/40"
            onclick={copyCode}
          >
            {code}
          </button>
          <span class="absolute right-3 top-1/2 -translate-y-1/2 font-mono text-[10px] uppercase text-text-muted opacity-0 transition-opacity group-hover:opacity-100">
            {copied ? 'copied' : 'click to copy'}
          </span>
        </div>

        <!-- CLI command -->
        <div class="rounded border border-border bg-bg px-4 py-3 font-mono text-xs text-text-dim">
          <span class="text-text-muted">$</span> tossit receive {code}
        </div>
      </div>

      <!-- Browser link -->
      <div class="rounded border border-border bg-surface px-4 py-3">
        <div class="mb-1 font-mono text-[10px] uppercase tracking-widest text-text-muted">Browser link</div>
        <button
          class="w-full truncate text-left font-mono text-xs text-accent/70 transition-colors hover:text-accent"
          onclick={copyUrl}
        >
          {browserUrl}
        </button>
      </div>

      <button
        class="w-full rounded border border-border py-2.5 font-mono text-xs uppercase tracking-widest text-text-muted transition-all hover:border-accent hover:text-accent"
        onclick={reset}
      >
        Send another
      </button>
    </div>
  {/if}
</div>
