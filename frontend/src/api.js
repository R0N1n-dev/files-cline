/**
 * Backend bridge.
 *
 * Prefers the Wails v2 Go bindings (window.go.main.App) and transparently falls
 * back to the headless HTTP JSON API (used for browser dev and the demo mode).
 */

const wailsApp = () => (window.go && window.go.main && window.go.main.App) || null

const listeners = new Map()
let eventSource = null
let httpEventsBound = false

export const usingWails = () => !!wailsApp()

function bindHttpEvents() {
  if (httpEventsBound || typeof EventSource === 'undefined') return
  httpEventsBound = true
  try {
    eventSource = new EventSource('/api/events')
    eventSource.onmessage = (msg) => {
      try {
        const { event, data } = JSON.parse(msg.data)
        dispatch(event, data)
      } catch {
        /* ignore malformed frames */
      }
    }
  } catch {
    eventSource = null
  }
}

function dispatch(name, payload) {
  const set = listeners.get(name)
  if (set) set.forEach((cb) => cb(payload))
  const wildcard = listeners.get('*')
  if (wildcard) wildcard.forEach((cb) => cb({ event: name, data: payload }))
}

/** Subscribe to a backend event. Returns an unsubscribe function. */
export function onEvent(name, cb) {
  if (!listeners.has(name)) listeners.set(name, new Set())
  listeners.get(name).add(cb)
  return () => listeners.get(name)?.delete(cb)
}

/** Invoke a Go backend method by name. */
export async function call(method, ...args) {
  const app = wailsApp()
  if (app && typeof app[method] === 'function') {
    return app[method](...args)
  }
  const res = await fetch('/api/' + method, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(args)
  })
  const data = await res.json()
  if (data && data.error) throw new Error(data.error)
  return data ? data.result : null
}

export const backend = {
  ListDirectory: (path, showHidden, sortBy, sortDesc, filter) =>
    call('ListDirectory', path, showHidden, sortBy, sortDesc, filter),
  GetSidebar: () => call('GetSidebar'),
  GetConfig: () => call('GetConfig'),
  SaveConfig: (cfg) => call('SaveConfig', cfg),
  GetHomeDir: () => call('GetHomeDir'),
  GetMounts: () => call('GetMounts'),
  GetDiskUsage: (path) => call('GetDiskUsage', path),
  GetThumbnail: (path, size) => call('GetThumbnail', path, size),
  QueueThumbnails: (paths, size) => call('QueueThumbnails', paths, size),
  InspectFile: (path) => call('InspectFile', path),
  CreateFolder: (parent, name) => call('CreateFolder', parent, name),
  CreateFile: (parent, name, content) => call('CreateFile', parent, name, content),
  Rename: (path, newName) => call('Rename', path, newName),
  BatchRename: (pairs) => call('BatchRename', pairs),
  DuplicateItems: (paths) => call('DuplicateItems', paths),
  ChmodPaths: (paths, mode) => call('ChmodPaths', paths, mode),
  DeletePermanently: (paths) => call('DeletePermanently', paths),
  Copy: (sources, dest, mode) => call('Copy', sources, dest, mode),
  Move: (sources, dest, mode) => call('Move', sources, dest, mode),
  ExtractArchive: (archives, dest) => call('ExtractArchive', archives, dest),
  CreateArchive: (paths, dest) => call('CreateArchive', paths, dest),
  ListOperations: () => call('ListOperations'),
  CancelOperation: (id) => call('CancelOperation', id),
  PauseOperation: (id) => call('PauseOperation', id),
  ResumeOperation: (id) => call('ResumeOperation', id),
  ClearFinishedOperations: () => call('ClearFinishedOperations'),
  UndoOperation: (id) => call('UndoOperation', id),
  ResolveConflict: (opId, srcPath, decision) => call('ResolveConflict', opId, srcPath, decision),
  HandleDrop: (sources, dest, move) => call('HandleDrop', sources, dest, move),
  MoveToTrash: (paths) => call('MoveToTrash', paths),
  UndoTrash: () => call('UndoTrash'),
  RestoreTrashItems: (ids) => call('RestoreTrashItems', ids),
  DeleteTrashItems: (ids) => call('DeleteTrashItems', ids),
  EmptyTrash: () => call('EmptyTrash'),
  GetTrashInfo: () => call('GetTrashInfo'),
  GetAppsForPath: (path) => call('GetAppsForPath', path),
  OpenWith: (desktopId, paths) => call('OpenWith', desktopId, paths),
  OpenDefault: (paths) => call('OpenDefault', paths),
  SetDefaultApp: (mime, desktopId) => call('SetDefaultApp', mime, desktopId),
  RunCustomCommand: (cmd, paths) => call('RunCustomCommand', cmd, paths),
  ShowInNativeFileManager: (paths) => call('ShowInNativeFileManager', paths),
  OpenTerminal: (dir) => call('OpenTerminal', dir),
  CopyToClipboard: (paths, cut) => call('CopyToClipboard', paths, cut),
  ReadClipboard: () => call('ReadClipboard'),
  StartOutboundDrag: (paths) => call('StartOutboundDrag', paths),
  Search: (root, query, showHidden) => call('Search', root, query, showHidden),
  CancelSearch: () => call('CancelSearch'),
  AddBookmark: (path) => call('AddBookmark', path),
  RemoveBookmark: (path) => call('RemoveBookmark', path)
}

export function initEvents() {
  const runtime = window.runtime
  if (runtime && typeof runtime.EventsOn === 'function') {
    const names = [
      'op:progress', 'op:state', 'op:done', 'op:undo', 'op:conflict',
      'thumbs:ready', 'trash:changed', 'fs:changed', 'drop:received', 'archive:progress'
    ]
    names.forEach((n) => runtime.EventsOn(n, (data) => dispatch(n, data)))
  } else {
    bindHttpEvents()
  }
}
