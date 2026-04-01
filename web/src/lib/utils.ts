export function formatSize(bytes: number): string {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return bytes + ' B'
}

export function formatUptime(uptime: string): string {
  return uptime
}

export function getWsUrl(token?: string): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  let url = `${proto}//${location.host}/ws`
  if (token) url += `?token=${encodeURIComponent(token)}`
  return url
}

export function getApiUrl(path: string, token?: string): string {
  let url = path
  if (token) url += `?token=${encodeURIComponent(token)}`
  return url
}

export function hashEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false
  let diff = 0
  for (let i = 0; i < a.length; i++) diff |= a[i] ^ b[i]
  return diff === 0
}
