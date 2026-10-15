<template>
  <div v-if="state.paletteOpen" class="overlay" @mousedown.self="state.paletteOpen = false">
    <div class="modal palette">
      <input
        ref="input"
        v-model="query"
        placeholder="Type a command or a folder path…"
        spellcheck="false"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.prevent="runActive"
        @keydown.esc.prevent="state.paletteOpen = false"
      />
      <div class="palette-list">
        <button
          v-for="(cmd, i) in filtered"
          :key="cmd.id"
          class="palette-item"
          :class="{ active: i === cursor }"
          @mouseenter="cursor = i"
          @click="run(cmd)"
        >
          <Icon :name="cmd.icon" />
          <span class="grow truncate">{{ cmd.label }}</span>
          <span class="hint">{{ cmd.hint }}</span>
        </button>
        <p v-if="!filtered.length" class="hint" style="padding: 12px">
          No matching command.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { state, persist, entriesOf } from '../store'
import { navigate, navigateSecondary, setSort, setViewMode, toggleHidden, toggleSplitView } from '../storeNav'
import { toggleInspector, openTrash, runSearch } from '../storeOps'
import { targetPaths, trashPaths, duplicateSelection, emptyTrash, undoTrash, targetDir, createFolder, createFile } from '../storeFiles'
import {
  copySelection, pasteInto, compressSelection, extractSelection, showInNativeFileManager,
  openTerminal, stageOutboundDrag, transferToOtherPane
} from '../storeFiles2'
import { openPrompt } from '../prompt'

const query = ref('')
const cursor = ref(0)
const input = ref(null)

const commands = computed(() => [
  { id: 'go', label: query.value.startsWith('/') || query.value.startsWith('~') ? `Open ${query.value}` : 'Go to folder…', icon: 'up', hint: state.splitView && state.activePane === 'secondary' ? 'other pane' : 'path', run: () => (state.splitView && state.activePane === 'secondary' ? navigateSecondary(query.value) : navigate(query.value)) },
  { id: 'search', label: `Search “${query.value}” here`, icon: 'search', hint: 'recursive', run: () => runSearch(query.value, state.cwd) },
  { id: 'new-folder', label: 'New folder…', icon: 'folderPlus', hint: 'Ctrl+Shift+N', run: () => openPrompt({ title: 'New folder', label: 'Folder name', value: 'New Folder', confirmLabel: 'Create', onSubmit: (n) => createFolder(targetDir(), n) }) },
  { id: 'new-file', label: 'New file…', icon: 'filePlus', run: () => openPrompt({ title: 'New file', label: 'File name', value: 'untitled.txt', confirmLabel: 'Create', onSubmit: (n) => createFile(targetDir(), n, '') }) },
  { id: 'terminal', label: 'Open in Terminal', icon: 'terminal', hint: 'Ctrl+T', run: () => openTerminal(targetDir()) },
  { id: 'native', label: 'Open in Native File Manager', icon: 'external', run: () => showInNativeFileManager([targetDir()]) },
  { id: 'trash', label: 'Move selection to Trash', icon: 'trash', hint: 'Delete', run: () => trashPaths(targetPaths()) },
  { id: 'undo-trash', label: 'Undo last Trash', icon: 'undo', run: () => undoTrash() },
  { id: 'empty-trash', label: 'Empty Trash', icon: 'trash', run: () => emptyTrash() },
  { id: 'copy', label: 'Copy selection', icon: 'copy', hint: 'Ctrl+C', run: () => copySelection(false) },
  { id: 'cut', label: 'Cut selection', icon: 'cut', hint: 'Ctrl+X', run: () => copySelection(true) },
  { id: 'paste', label: 'Paste into this folder', icon: 'paste', hint: 'Ctrl+V', run: () => pasteInto(targetDir()) },
  { id: 'copy-pane', label: 'Copy selection to the other pane', icon: 'copy', hint: 'F5', run: () => transferToOtherPane(false) },
  { id: 'move-pane', label: 'Move selection to the other pane', icon: 'cut', hint: 'F6', run: () => transferToOtherPane(true) },
  { id: 'duplicate', label: 'Duplicate selection', icon: 'copy', hint: 'Ctrl+D', run: () => duplicateSelection() },
  { id: 'compress', label: 'Compress selection…', icon: 'archive', run: () => compressSelection(`${targetDir()}/archive.zip`) },
  { id: 'extract', label: 'Extract archive here', icon: 'extract', run: () => extractSelection(targetDir()) },
  { id: 'drag', label: 'Drag selection out to another app', icon: 'drag', run: () => stageOutboundDrag() },
  { id: 'grid', label: 'Switch to Grid view', icon: 'grid', run: () => setViewMode('grid') },
  { id: 'list', label: 'Switch to List view', icon: 'list', run: () => setViewMode('list') },
  { id: 'hidden', label: state.showHidden ? 'Hide hidden files' : 'Show hidden files', icon: 'hidden', hint: 'Ctrl+H', run: () => toggleHidden() },
  { id: 'sort-name', label: 'Sort by name', icon: 'sort', run: () => setSort('name') },
  { id: 'sort-size', label: 'Sort by size', icon: 'sort', run: () => setSort('size') },
  { id: 'sort-date', label: 'Sort by modified', icon: 'clock', run: () => setSort('modified') },
  { id: 'split', label: 'Toggle split view', icon: 'split', hint: 'F3', run: () => toggleSplitView() },
  { id: 'inspector', label: 'Toggle inspector', icon: 'info', hint: 'Ctrl+I', run: () => toggleInspector() },
  { id: 'select-all', label: 'Select all in this pane', icon: 'check', hint: 'Ctrl+A', run: () => { state.selection = entriesOf().map((e) => e.path) } },
  { id: 'trash-view', label: 'Open Trash', icon: 'trash', run: () => openTrash() }
])

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return commands.value
  return commands.value.filter((c) => c.label.toLowerCase().includes(q))
})

watch(() => state.paletteOpen, (open) => {
  if (!open) return
  query.value = ''
  cursor.value = 0
  nextTick(() => input.value && input.value.focus())
})

watch(query, () => { cursor.value = 0 })

function move(delta) {
  const len = filtered.value.length
  if (!len) return
  cursor.value = (cursor.value + delta + len) % len
}

async function runActive() {
  const cmd = filtered.value[cursor.value]
  if (cmd) await run(cmd)
}

async function run(cmd) {
  state.paletteOpen = false
  await cmd.run()
}
</script>
