import { state, toast, fail, upsertOp, selectedEntries } from './store'
import { navigate, refresh } from './storeNav'
import { backend } from './api'

/** Internal clipboard used when the system clipboard is unavailable. */
const internalClipboard = { paths: [], cut: false, active: false }

export function targetPaths(explicit) {
  if (explicit && explicit.length) return explicit
  return state.selection.length ? [...state.selection] : []
}

/* ------------------------------------------------------------ open / activate */

export async function activateEntry(entry) {
  if (!entry) return
  if (entry.isDir) {
    await navigate(entry.path)
    return
  }
  await openPaths([entry.path])
}

export async function openPaths(paths) {
  try {
    await backend.OpenDefault(paths)
  } catch (err) {
    fail(err)
  }
}

/* -------------------------------------------------------------- create items */

export async function createFolder(parent, name) {
  try {
    await backend.CreateFolder(parent || state.cwd, name)
    await refresh()
    toast('Folder created', 'success')
  } catch (err) {
    fail(err)
  }
}

export async function createFile(parent, name, content = '') {
  try {
    await backend.CreateFile(parent || state.cwd, name, content)
    await refresh()
    toast('File created', 'success')
  } catch (err) {
    fail(err)
  }
}

export async function renameEntry(path, newName) {
  try {
    await backend.Rename(path, newName)
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function applyBatchRename(pairs) {
  try {
    await backend.BatchRename(pairs)
    state.batchRenameOpen = false
    await refresh()
    toast(`Renamed ${pairs.length} item(s)`, 'success')
  } catch (err) {
    fail(err)
  }
}

export async function duplicateSelection() {
  const paths = targetPaths()
  if (!paths.length) return
  try {
    const created = await backend.DuplicateItems(paths)
    await refresh()
    toast(`Created ${created.length} duplicate(s)`, 'success')
  } catch (err) {
    fail(err)
  }
}

export async function chmodSelection(mode) {
  const paths = targetPaths()
  if (!paths.length) return
  try {
    await backend.ChmodPaths(paths, mode)
    await refresh()
    toast(`Permissions set to ${mode}`, 'success')
  } catch (err) {
    fail(err)
  }
}

/* -------------------------------------------------------------------- trash */

export async function trashPaths(paths) {
  if (!paths.length) return
  try {
    const txn = await backend.MoveToTrash(paths)
    const count = txn && txn.items ? txn.items.length : paths.length
    const mounts = txn && txn.mounts && txn.mounts.length
      ? ` (home Trash + ${txn.mounts.join(', ')})`
      : ''
    state.trashUndo = { count }
    toast(`${count} item(s) moved to Trash${mounts}`, 'success', {
      label: 'Undo',
      run: () => undoTrash()
    })
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function undoTrash() {
  try {
    const n = await backend.UndoTrash()
    state.trashUndo = null
    toast(`Restored ${n} item(s)`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function restoreSelection() {
  const ids = selectedEntries.value.filter((e) => e.trashId).map((e) => e.trashId)
  if (!ids.length) return
  try {
    const n = await backend.RestoreTrashItems(ids)
    toast(`Restored ${n} item(s)`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function deleteTrashSelection() {
  const ids = selectedEntries.value.filter((e) => e.trashId).map((e) => e.trashId)
  if (!ids.length) return
  try {
    const n = await backend.DeleteTrashItems(ids)
    toast(`Deleted ${n} item(s) permanently`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function emptyTrash() {
  try {
    const n = await backend.EmptyTrash()
    toast(`Emptied Trash (${n} item(s))`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}

export async function deletePermanently(paths) {
  if (!paths.length) return
  try {
    await backend.DeletePermanently(paths)
    toast(`Deleted ${paths.length} item(s) permanently`, 'success')
    await refresh()
  } catch (err) {
    fail(err)
  }
}
