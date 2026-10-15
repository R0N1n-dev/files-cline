import { reactive, computed } from 'vue'
import { backend } from './api'

/** Global reactive application state (composition-API store, no Pinia needed). */
export const state = reactive({
  ready: false,
  demoMode: false,
  cwd: '',
  listing: {
    entries: [], breadcrumbs: [], totalCount: 0, totalSize: 0,
    diskFree: 0, diskTotal: 0, diskUsedPercent: 0, isTrashView: false, parentPath: ''
  },
  loading: false,
  error: '',
  selection: [],
  lastAnchor: -1,
  viewMode: 'grid',
  iconSize: 112,
  sortBy: 'name',
  sortDesc: false,
  showHidden: false,
  filter: '',
  theme: 'system',
  resolvedTheme: 'light',
  sidebar: { places: [], volumes: [], bookmarks: [], userName: '', hostName: '' },
  sidebarVisible: true,
  back: [],
  forward: [],
  ops: [],
  toast: null,
  trashUndo: null,
  inspectorOpen: false,
  inspector: null,
  quickLook: null,
  quickLookLoading: false,
  openWith: null,
  rename: null,
  batchRenameOpen: false,
  conflict: null,
  conflictMode: 'rename',
  paletteOpen: false,
  contextMenu: null,
  dropActive: false,
  dragShelf: [],
  splitView: false,
  secondary: { cwd: '', entries: [], breadcrumbs: [], totalCount: 0 },
  secondaryActive: false,
  searchMode: false,
  searchResults: [],
  searchQuery: '',
  thumbs: {},
  gridColumns: 6
})

export const selectedEntries = computed(() =>
  state.listing.entries.filter((e) => state.selection.includes(e.path))
)

export const totalSelectedSize = computed(() =>
  selectedEntries.value.reduce((sum, e) => sum + (e.size || 0), 0)
)

export const activeOps = computed(() =>
  state.ops.filter((o) => o.state === 'running' || o.state === 'paused' || o.state === 'cancelling')
)

export const opProgress = computed(() => {
  if (!activeOps.value.length) return 0
  return activeOps.value.reduce((s, o) => s + (o.progress || 0), 0) / activeOps.value.length
})

export const currentBreadcrumbs = computed(() =>
  state.listing.breadcrumbs && state.listing.breadcrumbs.length
    ? state.listing.breadcrumbs
    : [{ name: '/', path: '/' }]
)

export function isSelected(path) {
  return state.selection.includes(path)
}

export function fail(err) {
  const message = err && err.message ? err.message : String(err)
  state.error = message
  toast(message, 'error')
  return null
}

export function toast(message, kind = 'info', action = null) {
  state.toast = { message, kind, action, id: Date.now() }
  const id = state.toast.id
  setTimeout(() => {
    if (state.toast && state.toast.id === id) state.toast = null
  }, action ? 9000 : 4200)
}

export function dismissToast() {
  state.toast = null
}

export function upsertOp(op) {
  if (!op || !op.id) return
  const idx = state.ops.findIndex((o) => o.id === op.id)
  if (idx >= 0) state.ops.splice(idx, 1, { ...state.ops[idx], ...op })
  else state.ops.unshift(op)
}

/* ------------------------------------------------------------------- theme */

export function applyTheme() {
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
  state.resolvedTheme = state.theme === 'system' ? (prefersDark ? 'dark' : 'light') : state.theme
  document.documentElement.dataset.theme = state.resolvedTheme
}

export function setTheme(theme) {
  state.theme = theme
  applyTheme()
  persist()
}

export function toggleTheme() {
  setTheme(state.resolvedTheme === 'dark' ? 'light' : 'dark')
}

if (window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (state.theme === 'system') applyTheme()
  })
}

/* -------------------------------------------------------------- persistence */

export function applyConfig(cfg) {
  if (!cfg) return
  state.viewMode = cfg.viewMode || 'grid'
  state.iconSize = cfg.iconSize || 112
  state.sortBy = cfg.sortBy || 'name'
  state.sortDesc = !!cfg.sortDesc
  state.showHidden = !!cfg.showHidden
  state.theme = cfg.theme || 'system'
  state.splitView = !!cfg.splitView
  state.conflictMode = cfg.conflictMode || 'rename'
}

export function persist() {
  backend.SaveConfig({
    theme: state.theme,
    viewMode: state.viewMode,
    iconSize: state.iconSize,
    sortBy: state.sortBy,
    sortDesc: state.sortDesc,
    showHidden: state.showHidden,
    splitView: state.splitView,
    conflictMode: state.conflictMode,
    lastPath: state.cwd
  }).catch(() => {})
}
