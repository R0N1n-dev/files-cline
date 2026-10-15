import { state, toast, fail, upsertOp, activeCwd } from './store'
import { refresh } from './storeNav'
import { backend } from './api'
import { targetPaths } from './storeFiles'

/* ------------------------------------------------------------------ archive */

export async function compressSelection(destArchive) {
  const paths = targetPaths()
  if (!paths.length) return
  try {
    const dest = await backend.CreateArchive(paths, destArchive)
    toast(`Created ${dest.split('/').pop()}`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function extractSelection(destDir) {
  const archives = targetPaths().filter((p) => /\.(zip|tar|tar\.gz|tgz)$/i.test(p))
  if (!archives.length) {
    toast('Select a .zip, .tar, or .tar.gz archive first', 'error')
    return
  }
  try {
    const id = await backend.ExtractArchive(archives, destDir || activeCwd())
    toast('Extracting — you can cancel it from the operations ring', 'info')
    const ops = await backend.ListOperations()
    ;(ops || []).forEach(upsertOp)
    return id
  } catch (err) {
    fail(err)
  }
}

/* --------------------------------------------------------------------- drop */

export async function handleDrop(sources, destDir, move = false) {
  if (!sources || !sources.length) return
  try {
    await backend.HandleDrop(sources, destDir || activeCwd(), move)
    toast(`${move ? 'Moving' : 'Copying'} ${sources.length} dropped item(s)…`, 'info')
  } catch (err) {
    fail(err)
  }
}

/* ---------------------------------------------------------------- clipboard */

const internalClipboard = { paths: [], cut: false }

export async function copySelection(cut = false) {
  const paths = targetPaths()
  if (!paths.length) return
  internalClipboard.paths = paths
  internalClipboard.cut = cut
  state.clipboardCut = cut ? [...paths] : []
  try {
    await backend.CopyToClipboard(paths, cut)
    toast(`${cut ? 'Cut' : 'Copied'} ${paths.length} item(s) — paste into any Linux app`, 'success')
  } catch {
    toast(`${cut ? 'Cut' : 'Copied'} ${paths.length} item(s) (internal clipboard)`, 'info')
  }
}

export async function pasteInto(destDir) {
  const target = destDir || activeCwd()
  let paths = internalClipboard.paths
  let cut = internalClipboard.cut
  let fromSystem = false

  try {
    const payload = await backend.ReadClipboard()
    if (payload && payload.paths && payload.paths.length) {
      paths = payload.paths
      cut = !!payload.cut
      fromSystem = true
    }
  } catch {
    /* system clipboard unavailable — use the internal one */
  }

  if (!paths || !paths.length) {
    toast('Clipboard is empty', 'info')
    return
  }

  try {
    if (cut) {
      await backend.Move(paths, target, state.conflictMode)
      internalClipboard.paths = []
    } else {
      await backend.Copy(paths, target, state.conflictMode)
    }
    const origin = fromSystem ? ' from the system clipboard' : ''
    toast(`${cut ? 'Moving' : 'Copying'} ${paths.length} item(s)${origin}…`, 'info')
  } catch (err) {
    fail(err)
  }
}

/* ------------------------------------------------------ native integrations */

export async function showInNativeFileManager(paths) {
  try {
    await backend.ShowInNativeFileManager(targetPaths(paths))
    toast('Opened in the native file manager', 'success')
  } catch (err) {
    fail(err)
  }
}

export async function openTerminal(dir) {
  try {
    await backend.OpenTerminal(dir || state.cwd)
  } catch (err) {
    fail(err)
  }
}

export async function stageOutboundDrag() {
  const paths = targetPaths()
  if (!paths.length) return
  state.dragShelf = paths
  try {
    await backend.StartOutboundDrag(paths)
    toast('Drag the floating card into another app, or use “Copy for Linux apps”', 'info')
  } catch {
    toast('Native drag source unavailable — copied to the clipboard instead', 'info')
    await copySelection(false)
  }
}

export function clearDragShelf() {
  state.dragShelf = []
}

/* ------------------------------------------------------- dual-pane transfer */

/**
 * Copies (or moves) the active pane's selection into the other pane — the
 * classic dual-pane "F5 / F6" workflow.
 */
export async function transferToOtherPane(move = false) {
  const sources = targetPaths()
  const dest = state.activePane === 'secondary' ? state.cwd : state.secondary.cwd
  if (!sources.length) {
    toast('Nothing selected', 'info')
    return
  }
  if (!dest) {
    toast('Open a folder in the second pane first', 'info')
    return
  }
  try {
    await backend[move ? 'Move' : 'Copy'](sources, dest, state.conflictMode)
    toast(`${move ? 'Moving' : 'Copying'} ${sources.length} item(s) to ${dest}`, 'info')
  } catch (err) {
    fail(err)
  }
}

export async function addBookmark(path) {
  try {
    await backend.AddBookmark(path || activeCwd())
    state.sidebar = await backend.GetSidebar()
    toast('Added to bookmarks', 'success')
  } catch (err) {
    fail(err)
  }
}

export async function removeBookmark(path) {
  try {
    await backend.RemoveBookmark(path)
    state.sidebar = await backend.GetSidebar()
    toast('Bookmark removed', 'info')
  } catch (err) {
    fail(err)
  }
}
