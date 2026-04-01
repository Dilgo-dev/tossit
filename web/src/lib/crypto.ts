import { argon2id } from 'hash-wasm'

const KDF_SALT = 'tossit-v1-key-derivation'
export const CHUNK_SIZE = 64 * 1024 // 64KB, matches Go side

export async function deriveKey(code: string): Promise<CryptoKey> {
  const salt = new TextEncoder().encode(KDF_SALT)
  const keyBytes = await argon2id({
    password: code,
    salt,
    iterations: 1,
    memorySize: 65536,
    parallelism: 4,
    hashLength: 32,
    outputType: 'binary',
  })

  return crypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: 'AES-GCM' },
    false,
    ['encrypt', 'decrypt'],
  )
}

function makeNonce(seq: number): Uint8Array {
  const nonce = new Uint8Array(12)
  new DataView(nonce.buffer).setUint32(8, seq)
  return nonce
}

export async function encryptChunk(
  key: CryptoKey,
  seq: number,
  plaintext: Uint8Array,
): Promise<Uint8Array> {
  const nonce = makeNonce(seq)
  const ciphertext = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: nonce },
    key,
    plaintext,
  )
  return new Uint8Array(ciphertext)
}

export async function decryptChunk(
  key: CryptoKey,
  seq: number,
  ciphertext: Uint8Array,
): Promise<Uint8Array> {
  const nonce = makeNonce(seq)
  const plaintext = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: nonce },
    key,
    ciphertext,
  )
  return new Uint8Array(plaintext)
}

export async function sha256(data: Uint8Array): Promise<Uint8Array> {
  const hash = await crypto.subtle.digest('SHA-256', data)
  return new Uint8Array(hash)
}
