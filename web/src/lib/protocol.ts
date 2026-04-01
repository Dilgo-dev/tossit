export const MsgRegister = 0x01
export const MsgJoin = 0x02
export const MsgReady = 0x03
export const MsgData = 0x04
export const MsgError = 0x05
export const MsgClose = 0x06
export const MsgBrowserJoin = 0x07
export const MsgStored = 0x08

export const PeerMetadata = 0x10
export const PeerChunk = 0x11
export const PeerDone = 0x12

export function encode(type: number, payload: Uint8Array): ArrayBuffer {
  const buf = new Uint8Array(1 + payload.length)
  buf[0] = type
  buf.set(payload, 1)
  return buf.buffer
}

export function decode(data: ArrayBuffer): { type: number; payload: Uint8Array } {
  const buf = new Uint8Array(data)
  return { type: buf[0], payload: buf.slice(1) }
}

export function encodeRegister(code: string, streamMode: boolean): ArrayBuffer {
  const codeBytes = new TextEncoder().encode(code)
  const payload = new Uint8Array(1 + codeBytes.length)
  payload[0] = streamMode ? 0x01 : 0x00
  payload.set(codeBytes, 1)
  return encode(MsgRegister, payload)
}

export function encodeBrowserJoin(code: string): ArrayBuffer {
  const codeBytes = new TextEncoder().encode(code)
  return encode(MsgBrowserJoin, codeBytes)
}

export interface FileMetadata {
  name: string
  size: number
  mode: number
  is_dir: boolean
  chunk_size: number
  file_count?: number
}

export function encodeMetadata(meta: FileMetadata): Uint8Array {
  const json = new TextEncoder().encode(JSON.stringify(meta))
  const buf = new Uint8Array(1 + json.length)
  buf[0] = PeerMetadata
  buf.set(json, 1)
  return buf
}

export function encodeChunk(seq: number, ciphertext: Uint8Array): Uint8Array {
  const buf = new Uint8Array(1 + 4 + ciphertext.length)
  buf[0] = PeerChunk
  new DataView(buf.buffer).setUint32(1, seq)
  buf.set(ciphertext, 5)
  return buf
}

export function encodeDone(hash: Uint8Array): Uint8Array {
  const buf = new Uint8Array(1 + hash.length)
  buf[0] = PeerDone
  buf.set(hash, 1)
  return buf
}

export function decodeChunk(payload: Uint8Array): { seq: number; ciphertext: Uint8Array } {
  const seq = new DataView(payload.buffer, payload.byteOffset + 1, 4).getUint32(0)
  const ciphertext = payload.slice(5)
  return { seq, ciphertext }
}

export function decodeDone(payload: Uint8Array): Uint8Array {
  return payload.slice(1)
}

const adjectives = [
  'able', 'bold', 'calm', 'cool', 'dark', 'deep', 'fair', 'fast',
  'fine', 'firm', 'flat', 'free', 'full', 'glad', 'gold', 'good',
  'gray', 'half', 'hard', 'high', 'holy', 'huge', 'just', 'keen',
  'kind', 'last', 'late', 'lazy', 'lean', 'left', 'live', 'long',
  'lost', 'loud', 'main', 'mild', 'near', 'neat', 'next', 'nice',
  'okay', 'only', 'open', 'pale', 'past', 'pink', 'pure', 'rare',
  'real', 'rich', 'ripe', 'safe', 'same', 'sick', 'slim', 'slow',
  'soft', 'sole', 'sore', 'sure', 'tall', 'tidy', 'tiny', 'tops',
  'true', 'vast', 'warm', 'weak', 'wide', 'wild', 'wise', 'zero',
  'airy', 'arch', 'avid', 'back', 'bare', 'base', 'best', 'blue',
  'bone', 'both', 'busy', 'cold', 'cozy', 'cute', 'damp', 'dear',
  'down', 'dual', 'dull', 'dusk', 'each', 'easy', 'edgy', 'epic',
  'even', 'evil', 'fond', 'four', 'grim', 'grit', 'hazy', 'iced',
  'iron', 'jade', 'lame', 'lush', 'malt', 'mega', 'mini', 'mint',
  'mist', 'much', 'muon', 'nano', 'neon', 'nine', 'nova', 'oaky',
  'oily', 'onyx', 'oval', 'peak', 'plum', 'posh', 'quad', 'quiz',
  'rosy', 'ruby', 'rust', 'sage', 'sand', 'silk', 'snap', 'snug',
  'star', 'surf', 'tart', 'teal', 'thin', 'tint', 'trim', 'twin',
  'used', 'void', 'wavy', 'wiry', 'woke', 'worn', 'zinc', 'zone',
  'aged', 'alps', 'aqua', 'ashy', 'auto', 'bass', 'bead', 'bent',
  'birk', 'bite', 'boxy', 'brew', 'buff', 'bulk', 'burr', 'cape',
  'cave', 'chic', 'chip', 'city', 'clay', 'clip', 'club', 'coal',
  'cobs', 'coil', 'colt', 'cork', 'crop', 'cube', 'curb', 'curt',
  'deft', 'dome', 'dose', 'dove', 'drip', 'drum', 'dune', 'dusk',
  'dust', 'earl', 'east', 'echo', 'edge', 'fawn', 'felt', 'fern',
  'feta', 'flex', 'flip', 'flop', 'flux', 'foam', 'folk', 'font',
  'fork', 'fowl', 'foxy', 'frog', 'fuel', 'fume', 'fuse', 'gale',
  'gear', 'gene', 'gilt', 'gist', 'glow', 'glue', 'gnat', 'goth',
  'gust', 'hale', 'hawk', 'heat', 'helm', 'hemp', 'herb', 'herd',
  'hike', 'hive', 'home', 'hoop', 'horn', 'hull', 'hymn', 'ibis',
  'itch', 'jive', 'jolt', 'jump', 'kelp', 'keys', 'kite', 'knit',
  'lace', 'lark', 'lava', 'leaf', 'lime', 'link', 'lion', 'loft',
]

const nouns = [
  'acorn', 'anvil', 'arrow', 'badge', 'beach', 'bench', 'berry',
  'birch', 'bloom', 'bluff', 'board', 'booth', 'brace', 'brick',
  'brook', 'brush', 'cabin', 'camel', 'candy', 'cedar', 'chain',
  'chalk', 'charm', 'chief', 'clamp', 'cliff', 'clock', 'cloud',
  'coach', 'coral', 'couch', 'crane', 'creek', 'crest', 'crown',
  'daisy', 'delta', 'depot', 'dingo', 'dodge', 'draft', 'drift',
  'eagle', 'ember', 'epoch', 'fable', 'fence', 'finch', 'flame',
  'flask', 'flint', 'frost', 'gleam', 'globe', 'goose', 'gorge',
  'grain', 'grape', 'grove', 'guild', 'haven', 'hazel', 'heart',
  'heron', 'hiker', 'house', 'ivory', 'jewel', 'juice', 'knoll',
  'lance', 'lemon', 'lilac', 'llama', 'lodge', 'lotus', 'manor',
  'maple', 'marsh', 'medal', 'melon', 'model', 'moose', 'mound',
  'north', 'ocean', 'olive', 'orbit', 'otter', 'oxide', 'pansy',
  'patch', 'peach', 'pearl', 'penny', 'perch', 'piano', 'pilot',
  'pixel', 'plaza', 'plume', 'point', 'poppy', 'pouch', 'prism',
  'pulse', 'quail', 'quilt', 'raven', 'ridge', 'river', 'robin',
  'rover', 'saint', 'scale', 'scout', 'shade', 'shell', 'shore',
  'spark', 'spire', 'spoke', 'spoon', 'spray', 'staff', 'stage',
  'stamp', 'steam', 'steel', 'stone', 'storm', 'stove', 'stump',
  'suite', 'swamp', 'swift', 'table', 'thorn', 'tiger', 'toast',
  'topaz', 'torch', 'tower', 'track', 'trail', 'trout', 'tulip',
  'vault', 'vigor', 'viola', 'viper', 'watch', 'whale', 'wheel',
  'yacht', 'aspen', 'atlas', 'bass', 'blaze', 'bonus', 'cadet',
  'cargo', 'chess', 'cider', 'cloak', 'cobra', 'comet', 'crisp',
  'cross', 'depth', 'diver', 'easel', 'elbow', 'fairy', 'feast',
  'fiber', 'flora', 'forge', 'gavel', 'ghost', 'glaze', 'green',
  'grind', 'hatch', 'hedge', 'hoist', 'honor', 'horns', 'hyena',
  'ingot', 'inlet', 'jolly', 'kayak', 'label', 'latch', 'ledge',
  'mango', 'maxim', 'mocha', 'motto', 'nexus', 'oasis', 'onion',
  'opera', 'panda', 'plaid', 'quake', 'quest', 'racer', 'ranch',
  'relay', 'salad', 'serif', 'siren', 'solar', 'spine', 'squid',
  'stern', 'surge', 'talon', 'theta', 'twist', 'union', 'vapor',
  'verse', 'waltz', 'wedge', 'whisk', 'wrist', 'zebra', 'zonal',
  'amber', 'bison', 'bower', 'cairn', 'coral', 'crown', 'dwell',
  'flame', 'flute', 'frost', 'glyph', 'gnome', 'grape', 'index',
  'ivory', 'knack', 'lunar', 'marsh', 'nerve', 'octet', 'plank',
]

export function generateCode(): string {
  const adj = adjectives[Math.floor(Math.random() * adjectives.length)]
  const noun = nouns[Math.floor(Math.random() * nouns.length)]
  const num = Math.floor(Math.random() * 100).toString().padStart(2, '0')
  return `${adj}-${noun}-${num}`
}
