import { computed } from 'vue'
import { state } from '../store'

/* --------------------------------------------------------- size & date fmt */

export function formatSize(bytes) {
  if (bytes === null || bytes === undefined || bytes === 0) return '—'
  const units = ['B', 'kB', 'MB', 'GB', 'TB']
  let value = Math.abs(bytes)
  let unit = 0
  while (value >= 1000 && unit < units.length - 1) {
    value /= 1000
    unit++
  }
  const digits = value < 10 && unit > 0 ? 1 : 0
  return `${value.toFixed(digits)} ${units[unit]}`
}

export function formatSpeed(bytesPerSecond) {
  if (!bytesPerSecond) return ''
  return `${formatSize(bytesPerSecond)}/s`
}

export function formatEta(seconds) {
  if (!seconds || seconds < 0 || !isFinite(seconds)) return ''
  const s = Math.round(seconds)
  if (s < 60) return `${s}s left`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ${s % 60}s left`
  return `${Math.floor(m / 60)}h ${m % 60}m left`
}

export function formatDate(unixMillis) {
  if (!unixMillis) return '—'
  const d = new Date(unixMillis)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  const time = d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  if (sameDay) return `Today, ${time}`
  const yesterday = new Date(now.getTime() - 86400000)
  if (d.toDateString() === yesterday.toDateString()) return `Yesterday, ${time}`
  if (d.getFullYear() === now.getFullYear()) {
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) + ', ' + time
  }
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

export function relativeTime(unixMillis) {
  if (!unixMillis) return ''
  const diff = Date.now() - unixMillis
  const min = Math.floor(diff / 60000)
  if (min < 1) return 'just now'
  if (min < 60) return `${min} min ago`
  const hours = Math.floor(min / 60)
  if (hours < 24) return `${hours} h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} d ago`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} mo ago`
  return `${Math.floor(months / 12)} y ago`
}

/* ----------------------------------------------------------------- icons */

const CATEGORY_ICONS = {
  folder: 'M3 6.5A1.5 1.5 0 0 1 4.5 5h4l1.6 2H19.5A1.5 1.5 0 0 1 21 8.5v9A1.5 1.5 0 0 1 19.5 19h-15A1.5 1.5 0 0 1 3 17.5z',
  image: 'M4 5h16v14H4zM7 9.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM4 17l5-5 4 4 3-3 4 4',
  video: 'M4 6h11v12H4zM15 10l5-3v10l-5-3',
  audio: 'M9 17V6l10-2v11M9 17a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM19 15a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
  archive: 'M4 6h16v4H4zM5 10h14v8H5zM10 13h4',
  code: 'M9 8l-4 4 4 4M15 8l4 4-4 4',
  document: 'M7 3h7l4 4v14H7zM14 3v4h4',
  pdf: 'M7 3h7l4 4v14H7zM9 13h6M9 16h4',
  file: 'M7 3h7l4 4v14H7z'
}

const EXT_ICONS = {
  go: 'code', js: 'code', ts: 'code', vue: 'code', py: 'code', rs: 'code',
  c: 'code', cpp: 'code', h: 'code', java: 'code', sh: 'code', json: 'code',
  yml: 'code', yaml: 'code', toml: 'code', xml: 'code', html: 'code', css: 'code',
  md: 'document', txt: 'document', log: 'document', csv: 'document', ini: 'document',
  zip: 'archive', tar: 'archive', gz: 'archive', tgz: 'archive', xz: 'archive', '7z': 'archive',
  jpg: 'image', jpeg: 'image', png: 'image', gif: 'image', webp: 'image', svg: 'image',
  mp4: 'video', mkv: 'video', webm: 'video', mov: 'video',
  mp3: 'audio', flac: 'audio', wav: 'audio', ogg: 'audio', m4a: 'audio'
}

export function iconPath(entry) {
  if (!entry) return CATEGORY_ICONS.file
  if (entry.isDir) return CATEGORY_ICONS.folder
  const byExt = EXT_ICONS[(entry.extension || '').toLowerCase()]
  return CATEGORY_ICONS[byExt] || CATEGORY_ICONS[entry.category] || CATEGORY_ICONS.file
}

export function iconTint(entry) {
  if (!entry) return 'tint-file'
  if (entry.isDir) return 'tint-folder'
  return 'tint-' + (entry.category || 'file')
}

/* ------------------------------------------------------------------ layout */

export const visibleEntries = computed(() => {
  if (state.searchMode) return state.searchResults
  return state.listing.entries
})

export function selectionLabel() {
  const n = state.selection.length
  if (!n) return ''
  return n === 1 ? state.selection[0].split('/').pop() : `${n} items selected`
}
