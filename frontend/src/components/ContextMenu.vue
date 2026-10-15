<template>
  <div v-if="menu" class="popover" :style="menuStyle" @mousedown.stop @contextmenu.prevent>
    <template v-for="(item, i) in items" :key="i">
      <div v-if="item.sep" class="menu-sep" />
      <div v-else-if="item.label" class="menu-label">{{ item.label }}</div>
      <div v-else-if="item.submenu" style="position: relative">
        <button
          class="menu-item"
          :class="{ highlighted: submenuOpen === item.id }"
          @mouseenter="submenuOpen = item.id"
        >
          <Icon :name="item.icon" />
          <span>{{ item.text }}</span>
          <span class="shortcut"><Icon name="chevronRight" style="width: 12px; height: 12px" /></span>
        </button>
        <div
          v-if="submenuOpen === item.id"
          class="popover"
          style="left: 100%; top: 0; margin-left: 4px; max-height: 60vh; overflow: auto"
          @mouseenter="submenuOpen = item.id"
        >
          <button v-for="sub in item.submenu" :key="sub.text" class="menu-item" @click="run(sub)">
            <Icon :name="sub.icon || 'open'" />
            <span class="grow truncate">{{ sub.text }}</span>
            <span v-if="sub.hint" class="shortcut">{{ sub.hint }}</span>
          </button>
        </div>
      </div>
      <button
        v-else
        class="menu-item"
        :class="{ danger: item.danger }"
        :disabled="item.disabled"
        @click="run(item)"
      >
        <Icon :name="item.icon" />
        <span>{{ item.text }}</span>
        <span v-if="item.shortcut" class="shortcut">{{ item.shortcut }}</span>
      </button>
    </template>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { openPrompt } from '../prompt'
import {
  openPaths, trashPaths, deletePermanently, restoreSelection, deleteTrashSelection,
  emptyTrash, duplicateSelection, chmodSelection, targetPaths,
  createFolder, createFile
} from '../storeFiles'
import {
  stageOutboundDrag, compressSelection, extractSelection, copySelection, pasteInto,
  showInNativeFileManager, openTerminal
} from '../storeFiles2'
import { openWithMenu, quickLook, toggleInspector } from '../storeOps'
import { toggleHidden as toggleHiddenFiles, setSort } from '../storeNav'

const props = defineProps({
  menu: { type: Object, default: null }
})
const emit = defineEmits(['close'])

const submenuOpen = ref(null)

watch(() => props.menu, () => { submenuOpen.value = null })

const menuStyle = computed(() => {
  const m = props.menu || { x: 0, y: 0 }
  return {
    left: Math.min(m.x, Math.max(12, window.innerWidth - 280)) + 'px',
    top: Math.min(m.y, Math.max(12, window.innerHeight - 440)) + 'px'
  }
})

const isTrashView = computed(() => state.listing.isTrashView)
const isArchive = computed(() => {
  const e = props.menu && props.menu.entry
  return !!e && /\.(zip|tar|tar\.gz|tgz)$/i.test(e.name)
})

function sortMark(key) {
  return state.sortBy === key ? (state.sortDesc ? ' ↓' : ' ↑') : ''
}

function openWithSubmenu() {
  const subs = [{ text: 'Other Application…', icon: 'search', action: () => openWithMenu(props.menu.entry.path) }]
  const def = state.openWith && state.openWith.defaultApp
  if (def) subs.unshift({ text: def.name, hint: 'default', action: () => openPaths(targetPaths()) })
  return subs
}

function renameAction() {
  window.dispatchEvent(new CustomEvent('vela:rename', { detail: props.menu.entry.path }))
}

function compressAction() {
  const first = (props.menu.entry.name || 'archive').replace(/\.[^.]+$/, '')
  openPrompt({
    title: 'Compress',
    label: 'Archive name (.zip, .tar, .tar.gz)',
    value: `${first}.zip`,
    confirmLabel: 'Create',
    onSubmit: (name) => compressSelection(`${state.cwd}/${name}`)
  })
}

function newFolder() {
  openPrompt({
    title: 'New folder', label: 'Folder name', value: 'New Folder', confirmLabel: 'Create',
    onSubmit: (name) => createFolder(state.cwd, name)
  })
}

function newFile() {
  openPrompt({
    title: 'New file', label: 'File name', value: 'untitled.txt', confirmLabel: 'Create',
    onSubmit: (name) => createFile(state.cwd, name, '')
  })
}

function selectAll() {
  state.selection = state.listing.entries.map((e) => e.path)
}

function toggleHidden() {
  toggleHiddenFiles()
}

async function run(item) {
  if (!item || !item.action) return
  emit('close')
  await item.action()
}

function buildItems() {
  const entry = props.menu ? props.menu.entry : null
  const multi = state.selection.length > 1

  if (!entry) {
    return [
      { label: 'This folder' },
      { text: 'New Folder…', icon: 'folderPlus', shortcut: 'Ctrl+Shift+N', action: newFolder },
      { text: 'New File…', icon: 'filePlus', action: newFile },
      { text: 'Paste', icon: 'paste', shortcut: 'Ctrl+V', action: () => pasteInto(state.cwd) },
      { sep: true },
      { text: 'Select All', icon: 'check', shortcut: 'Ctrl+A', action: selectAll },
      {
        text: state.showHidden ? 'Hide Hidden Files' : 'Show Hidden Files',
        icon: 'hidden', shortcut: 'Ctrl+H', action: toggleHidden
      },
      { label: 'Sort by' },
      { text: `Name${sortMark('name')}`, icon: 'sort', action: () => setSort('name') },
      { text: `Size${sortMark('size')}`, icon: 'sort', action: () => setSort('size') },
      { text: `Modified${sortMark('modified')}`, icon: 'clock', action: () => setSort('modified') },
      { text: `Type${sortMark('type')}`, icon: 'sort', action: () => setSort('type') },
      { sep: true },
      { text: 'Open in Terminal', icon: 'terminal', action: () => openTerminal(state.cwd) },
      {
        text: 'Open in Native File Manager', icon: 'external',
        action: () => showInNativeFileManager([state.cwd])
      },
      { sep: true },
      { text: 'Empty Trash', icon: 'trash', danger: true, action: () => emptyTrash() }
    ]
  }

  if (isTrashView.value) {
    return [
      { label: entry.name },
      { text: 'Restore', icon: 'restore', action: () => restoreSelection() },
      { sep: true },
      { text: 'Delete Permanently', icon: 'trash', danger: true, action: () => deleteTrashSelection() },
      { text: 'Empty Trash', icon: 'trash', danger: true, action: () => emptyTrash() },
      { sep: true },
      { text: 'Properties', icon: 'info', action: () => toggleInspector() }
    ]
  }

  const list = [
    { label: entry.name },
    { text: 'Open', icon: 'open', shortcut: 'Enter', action: () => openPaths(targetPaths()) },
    { text: 'Open With…', icon: 'external', id: 'openwith', submenu: openWithSubmenu() },
    { text: 'Quick Look', icon: 'eye', shortcut: 'Space', action: () => quickLook(entry.path) },
    { sep: true },
    { text: 'Cut', icon: 'cut', shortcut: 'Ctrl+X', action: () => copySelection(true) },
    { text: 'Copy', icon: 'copy', shortcut: 'Ctrl+C', action: () => copySelection(false) },
    { text: 'Copy for Linux apps', icon: 'copy', hint: 'uri-list', action: () => copySelection(false) },
    { text: 'Duplicate', icon: 'copy', shortcut: 'Ctrl+D', action: () => duplicateSelection() },
    { text: 'Rename', icon: 'rename', shortcut: 'F2', action: renameAction }
  ]

  if (multi) {
    list.push({ text: 'Batch Rename…', icon: 'rename', action: () => { state.batchRenameOpen = true } })
  }
  if (isArchive.value) {
    list.push({ text: 'Extract Here', icon: 'extract', action: () => extractSelection(state.cwd) })
  }

  list.push(
    { sep: true },
    { text: 'Move to Trash', icon: 'trash', shortcut: 'Delete', danger: true, action: () => trashPaths(targetPaths()) },
    { text: 'Compress…', icon: 'archive', action: compressAction },
    { sep: true },
    { text: 'Drag Out to App', icon: 'drag', action: () => stageOutboundDrag() },
    {
      text: 'Open in Native File Manager', icon: 'external',
      action: () => showInNativeFileManager(targetPaths())
    },
    {
      text: 'Permissions', icon: 'shield', id: 'chmod',
      submenu: [
        { text: '644 (rw-r--r--)', action: () => chmodSelection('644') },
        { text: '600 (rw-------)', action: () => chmodSelection('600') },
        { text: '755 (rwxr-xr-x)', action: () => chmodSelection('755') },
        { text: '700 (rwx------)', action: () => chmodSelection('700') }
      ]
    },
    { text: 'Delete Permanently', icon: 'trash', danger: true, action: () => deletePermanently(targetPaths()) },
    { sep: true },
    { text: 'Properties', icon: 'info', shortcut: 'Ctrl+I', action: () => toggleInspector() }
  )

  return list
}

const items = computed(() => buildItems())
</script>
