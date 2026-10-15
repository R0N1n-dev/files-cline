<template>
  <div class="shell" @keydown="onGlobalKey">
    <HeaderBar />

    <div class="shell-body">
      <SidebarPlaces v-if="state.sidebarVisible" />

      <VirtualFileView ref="primaryView" @context-menu="onContextMenu">
        <template #toolbar>
          <span v-if="state.searchMode" class="pill">
            Search results: {{ state.searchResults.length }}
          </span>
          <span v-else-if="state.listing.totalCount" class="pill">
            {{ state.listing.totalCount }} items
            <template v-if="state.listing.totalSize">· {{ formatSize(state.listing.totalSize) }}</template>
          </span>
          <span v-if="state.selection.length" class="pill warn">
            {{ state.selection.length }} selected · {{ formatSize(totalSelectedSize) }}
          </span>
          <span v-if="state.demoMode" class="pill danger">demo data (backend offline)</span>
          <span v-if="state.loading" class="pill">working…</span>
        </template>
      </VirtualFileView>

      <VirtualFileView
        v-if="state.splitView"
        secondary
        :entries="state.listing.entries"
        :cwd="state.cwd"
        @context-menu="onContextMenu"
      />

      <InspectorDrawer />
    </div>

    <ContextMenu :menu="state.contextMenu" @close="state.contextMenu = null" />
    <OpenWithModal />
    <BatchRenameModal />
    <ConflictDialog />
    <CommandPalette />
    <PromptModal />
    <QuickLook />
    <DragShelf />
    <ToastHost />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import HeaderBar from './components/HeaderBar.vue'
import SidebarPlaces from './components/SidebarPlaces.vue'
import VirtualFileView from './components/VirtualFileView.vue'
import InspectorDrawer from './components/InspectorDrawer.vue'
import ContextMenu from './components/ContextMenu.vue'
import OpenWithModal from './components/OpenWithModal.vue'
import BatchRenameModal from './components/BatchRenameModal.vue'
import ConflictDialog from './components/ConflictDialog.vue'
import CommandPalette from './components/CommandPalette.vue'
import PromptModal from './components/PromptModal.vue'
import QuickLook from './components/QuickLook.vue'
import DragShelf from './components/DragShelf.vue'
import ToastHost from './components/ToastHost.vue'
import { state, totalSelectedSize, persist } from './store'
import { goBack, goForward, toggleHidden } from './storeNav'
import { toggleInspector, quickLook } from './storeOps'
import { openPrompt } from './prompt'
import { targetPaths, trashPaths, duplicateSelection, openPaths, createFolder } from './storeFiles'
import { copySelection, pasteInto } from './storeFiles2'
import { formatSize } from './utils/format'

const primaryView = ref(null)

function onContextMenu(payload) {
  state.contextMenu = payload
}

function closeFloating() {
  state.contextMenu = null
}

function onRenameRequest(e) {
  primaryView.value && primaryView.value.startRename(e.detail)
}

function onNewFolderRequest() {
  openPrompt({
    title: 'New folder',
    label: 'Folder name',
    value: 'New Folder',
    confirmLabel: 'Create',
    onSubmit: (name) => createFolder(state.cwd, name)
  })
}

function onGlobalKey(e) {
  const tag = (e.target && e.target.tagName) || ''
  const typing = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
  const mod = e.ctrlKey || e.metaKey

  if (e.key === 'Escape') {
    closeFloating()
    if (!typing) state.quickLook = null
    return
  }
  if (mod && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    state.paletteOpen = true
    return
  }
  if (typing) return

  if (mod && e.shiftKey && e.key.toLowerCase() === 'n') {
    e.preventDefault()
    onNewFolderRequest()
    return
  }
  if (mod && e.key.toLowerCase() === 'h') { e.preventDefault(); toggleHidden(); return }
  if (mod && e.key.toLowerCase() === 'i') { e.preventDefault(); toggleInspector(); return }
  if (mod && e.key.toLowerCase() === 'd') { e.preventDefault(); duplicateSelection(); return }
  if (mod && e.key.toLowerCase() === 'c') { e.preventDefault(); copySelection(false); return }
  if (mod && e.key.toLowerCase() === 'x') { e.preventDefault(); copySelection(true); return }
  if (mod && e.key.toLowerCase() === 'v') { e.preventDefault(); pasteInto(state.cwd); return }
  if (e.altKey && e.key === 'ArrowLeft') { e.preventDefault(); goBack(); return }
  if (e.altKey && e.key === 'ArrowRight') { e.preventDefault(); goForward(); return }
  if (e.key === 'F3') {
    e.preventDefault()
    state.splitView = !state.splitView
    persist()
    return
  }
  if (e.key === 'Delete' && state.selection.length) { e.preventDefault(); trashPaths(targetPaths()); return }
  if (e.key === 'F2' && state.selection.length) {
    e.preventDefault()
    onRenameRequest({ detail: state.selection[0] })
    return
  }
  if (e.key === ' ' && state.selection.length) { e.preventDefault(); quickLook(state.selection[0]); return }
  if (e.key === 'Enter' && state.selection.length === 1) {
    e.preventDefault()
    const entry = state.listing.entries.find((x) => x.path === state.selection[0])
    if (entry && !entry.isDir) openPaths([entry.path])
  }
}

function onWindowContextMenu(e) {
  if (!e.target.closest('.popover')) state.contextMenu = null
}

onMounted(() => {
  window.addEventListener('click', closeFloating)
  window.addEventListener('vela:rename', onRenameRequest)
  window.addEventListener('vela:new-folder', onNewFolderRequest)
  window.addEventListener('contextmenu', onWindowContextMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('click', closeFloating)
  window.removeEventListener('vela:rename', onRenameRequest)
  window.removeEventListener('vela:new-folder', onNewFolderRequest)
  window.removeEventListener('contextmenu', onWindowContextMenu)
})
</script>
