import { state, toast, fail, upsertOp, activeCwd, activeSelection } from './store'
import { refresh } from './storeNav'
import { backend } from './api'
import { targetPaths } from './storeFiles'

/* -------------------------------------------------------------- operations */

export async function refreshOperations() {
  try {
    const ops = await backend.ListOperations()
    if (Array.isArray(ops)) state.ops = ops
  } catch {
    /* ignore */
  }
}

export async function cancelOp(id) {
  try {
    await backend.CancelOperation(id)
    toast('Cancelling — partial files are removed', 'info')
  } catch (err) {
    fail(err)
  }
}

export async function pauseOp(id) {
  try {
    await backend.PauseOperation(id)
  } catch (err) {
    fail(err)
  }
}

export async function resumeOp(id) {
  try {
    await backend.ResumeOperation(id)
  } catch (err) {
    fail(err)
  }
}

export async function undoOp(id) {
  try {
    await backend.UndoOperation(id)
    toast('Reverted', 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function clearFinishedOps() {
  try {
    await backend.ClearFinishedOperations()
    await refreshOperations()
  } catch (err) {
    fail(err)
  }
}

export async function resolveConflict(decision) {
  const c = state.conflict
  if (!c) return
  try {
    await backend.ResolveConflict(c.id, c.srcPath, decision)
  } catch (err) {
    fail(err)
  } finally {
    state.conflict = null
  }
}

export function setConflictMode(mode) {
  state.conflictMode = mode
  import('./store').then((m) => m.persist())
  toast(`Conflict policy: ${mode}`, 'info')
}

/* ------------------------------------------------------------------- search */

let searchToken = 0

export async function runSearch(query, root) {
  const token = ++searchToken
  state.searchQuery = query
  if (!query) {
    state.searchMode = false
    state.searchResults = []
    return
  }
  state.searchMode = true
  state.loading = true
  try {
    const results = await backend.Search(root || state.cwd, query, state.showHidden)
    if (token !== searchToken) return
    state.searchResults = Array.isArray(results) ? results : []
  } catch (err) {
    if (token === searchToken) fail(err)
  } finally {
    if (token === searchToken) state.loading = false
  }
}

export async function cancelSearch() {
  searchToken++
  try {
    await backend.CancelSearch()
  } catch {
    /* ignore */
  }
  state.searchMode = false
  state.searchResults = []
  state.searchQuery = ''
}

/* ------------------------------------------------------- inspect & quick look */

export async function inspectPath(path) {
  if (!path) {
    state.inspector = null
    return
  }
  try {
    state.inspector = await backend.InspectFile(path)
  } catch (err) {
    state.inspector = null
    fail(err)
  }
}

export async function toggleInspector() {
  state.inspectorOpen = !state.inspectorOpen
  if (state.inspectorOpen) {
    await inspectPath(activeSelection()[0] || activeCwd())
  }
}

export async function quickLook(path) {
  if (!path) return
  if (state.quickLook && state.quickLook.entry && state.quickLook.entry.path === path) {
    state.quickLook = null
    return
  }
  state.quickLookLoading = true
  state.quickLook = { loading: true }
  try {
    state.quickLook = await backend.InspectFile(path)
  } catch (err) {
    state.quickLook = null
    fail(err)
  } finally {
    state.quickLookLoading = false
  }
}

export function closeQuickLook() {
  state.quickLook = null
}

/* --------------------------------------------------------------- open with */

export async function openWithMenu(path) {
  const target = path || activeSelection()[0] || activeCwd()
  try {
    const info = await backend.GetAppsForPath(target)
    state.openWith = { ...info, path: target }
  } catch (err) {
    fail(err)
  }
}

export async function openWithApp(desktopId) {
  const info = state.openWith
  if (!info) return
  const selection = activeSelection()
  const paths = selection.length ? [...selection] : [info.path]
  try {
    await backend.OpenWith(desktopId, paths)
    state.openWith = null
  } catch (err) {
    fail(err)
  }
}

export async function setDefaultAppForType() {
  const info = state.openWith
  if (!info || !info.defaultApp) return
  try {
    await backend.SetDefaultApp(info.mimeType, info.defaultApp.id)
    toast(`Default app for ${info.mimeType} updated`, 'success')
  } catch (err) {
    fail(err)
  }
}

export async function setAsDefault(desktopId) {
  const info = state.openWith
  if (!info) return
  try {
    await backend.SetDefaultApp(info.mimeType, desktopId)
    toast('Default application updated', 'success')
    await openWithMenu(info.path)
  } catch (err) {
    fail(err)
  }
}

export async function runCustomCommand(command) {
  const info = state.openWith
  const paths = state.selection.length ? [...state.selection] : info ? [info.path] : [state.cwd]
  try {
    await backend.RunCustomCommand(command, paths)
    state.openWith = null
    toast('Command launched', 'success')
  } catch (err) {
    fail(err)
  }
}

/* -------------------------------------------------------------- trash views */

export async function openTrash() {
  const { navigate } = await import('./storeNav')
  await navigate('trash://')
}

export function selectionPaths() {
  return targetPaths()
}
