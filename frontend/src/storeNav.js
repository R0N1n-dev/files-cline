import { state, fail, persist, applyConfig, applyTheme, toast } from './store'
import { backend, onEvent } from './api'
import { demoListing, demoSidebar, demoOperations, demoConfig } from './demo'

// Lets the shell resolve which folder an OS-level drop landed on.
let dropTargetResolver = null
export function setDropTargetResolver(fn) {
  dropTargetResolver = fn
}
export function resolveDropTarget(x, y) {
  return dropTargetResolver ? dropTargetResolver(x, y) : null
}

/* -------------------------------------------------------------- navigation */

export async function navigate(path, options = {}) {
  if (!path) return
  state.loading = true
  state.error = ''
  state.searchMode = false
  try {
    const listing = state.demoMode
      ? demoListing(path)
      : await backend.ListDirectory(path, state.showHidden, state.sortBy, state.sortDesc, state.filter)
    if (!listing) throw new Error('Cannot open ' + path)

    if (!options.replace && state.cwd && state.cwd !== listing.path) {
      state.back.push(state.cwd)
      state.forward = []
    }
    state.cwd = listing.path
    state.listing = listing
    state.selection = []
    state.lastAnchor = -1
    state.quickLook = null
    if (state.inspectorOpen && onInspectorNavigate) onInspectorNavigate(listing.path)
    queueVisibleThumbnails()
    if (!state.demoMode) persist()
  } catch (err) {
    fail(err)
  } finally {
    state.loading = false
  }
}

export async function goBack() {
  const prev = state.back.pop()
  if (!prev) return
  state.forward.push(state.cwd)
  await navigate(prev, { replace: true })
}

export async function goForward() {
  const next = state.forward.pop()
  if (!next) return
  state.back.push(state.cwd)
  await navigate(next, { replace: true })
}

export async function goUp() {
  const parent = state.listing.parentPath
  if (parent) await navigate(parent)
}

export async function refresh() {
  if (!state.cwd) return
  const keep = [...state.selection]
  try {
    const listing = state.demoMode
      ? demoListing(state.cwd)
      : await backend.ListDirectory(state.cwd, state.showHidden, state.sortBy, state.sortDesc, state.filter)
    state.listing = listing
    state.selection = keep.filter((p) => listing.entries.some((e) => e.path === p))
    queueVisibleThumbnails()
  } catch (err) {
    fail(err)
  }
}

export function setSort(sortBy) {
  if (state.sortBy === sortBy) state.sortDesc = !state.sortDesc
  else {
    state.sortBy = sortBy
    state.sortDesc = false
  }
  persist()
  refresh()
}

export function setFilter(value) {
  state.filter = value
  refresh()
}

export function toggleHidden() {
  state.showHidden = !state.showHidden
  persist()
  refresh()
}

export function setViewMode(mode) {
  state.viewMode = mode
  persist()
}

export function setIconSize(size) {
  state.iconSize = Math.min(320, Math.max(64, size))
  persist()
}

/* ---------------------------------------------------------------- selection */

export function setSelection(paths) {
  state.selection = [...new Set(paths)]
}

export function toggleSelection(path, index = -1) {
  if (state.selection.includes(path)) {
    state.selection = state.selection.filter((p) => p !== path)
  } else {
    state.selection = [...state.selection, path]
  }
  if (index >= 0) state.lastAnchor = index
}

export function rangeSelect(index) {
  const entries = state.listing.entries
  const anchor = state.lastAnchor >= 0 ? state.lastAnchor : 0
  const [from, to] = anchor <= index ? [anchor, index] : [index, anchor]
  const range = entries.slice(from, to + 1).map((e) => e.path)
  state.selection = [...new Set([...state.selection, ...range])]
}

export function selectAll() {
  state.selection = state.listing.entries.map((e) => e.path)
}

export function clearSelection() {
  state.selection = []
  state.lastAnchor = -1
}

export function selectOnly(path, index = 0) {
  state.selection = [path]
  state.lastAnchor = index
}

/* --------------------------------------------------------------- thumbnails */

export function thumbFor(path) {
  return state.thumbs[path] || null
}

export function queueVisibleThumbnails() {
  if (state.demoMode) return
  const images = state.listing.entries
    .filter((e) => e.category === 'image' && !e.isDir)
    .slice(0, 240)
    .map((e) => e.path)
  if (images.length) backend.QueueThumbnails(images, Math.max(160, state.iconSize * 2)).catch(() => {})
}

/* ------------------------------------------------------------------- startup */

let onInspectorNavigate = null
export function bindInspectorNavigation(fn) {
  onInspectorNavigate = fn
}

export async function init() {
  try {
    const cfg = await backend.GetConfig()
    applyConfig(cfg)
    state.sidebar = await backend.GetSidebar()
    await navigate(cfg && cfg.lastPath ? cfg.lastPath : await backend.GetHomeDir())
    const ops = await backend.ListOperations()
    state.ops = Array.isArray(ops) ? ops : []
    bindEvents()
  } catch (err) {
    state.demoMode = true
    applyConfig(demoConfig())
    state.sidebar = demoSidebar()
    await navigate('/demo')
    state.ops = demoOperations()
  }
  applyTheme()
  state.ready = true
}

function bindEvents() {
  onEvent('op:progress', (op) => import('./store').then((m) => m.upsertOp(op)))
  onEvent('op:state', (op) => import('./store').then((m) => m.upsertOp(op)))
  onEvent('op:done', (op) => import('./store').then((m) => {
    m.upsertOp(op)
    if (op.state === 'completed') toast(op.message || 'Operation completed', 'success')
    if (op.state === 'error') toast(op.error || 'Operation failed', 'error')
    if (op.kind === 'copy' || op.kind === 'move' || op.kind === 'extract') refresh()
  }))
  onEvent('op:undo', (op) => import('./store').then((m) => m.upsertOp(op)))
  onEvent('op:conflict', (c) => { state.conflict = c })
  onEvent('thumbs:ready', (r) => {
    if (r && r.uri) state.thumbs[r.path] = r.uri
  })
  onEvent('trash:changed', () => {
    backend.GetSidebar().then((s) => { state.sidebar = s }).catch(() => {})
  })
  onEvent('fs:changed', () => refresh())
  onEvent('drop:received', (payload) => {
    state.dropActive = false
    if (payload && payload.paths && payload.paths.length) {
      const hovered = resolveDropTarget(payload.x, payload.y)
      import('./storeFiles2').then((m) => m.handleDrop(payload.paths, hovered || state.cwd, false))
    }
  })
}
