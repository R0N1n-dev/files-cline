<template>
  <header class="headerbar">
    <button class="icon-btn" :disabled="!state.back.length" title="Back (Alt+Left)" @click="goBack">
      <Icon name="back" />
    </button>
    <button class="icon-btn" :disabled="!state.forward.length" title="Forward (Alt+Right)" @click="goForward">
      <Icon name="forward" />
    </button>
    <button class="icon-btn" :disabled="!state.listing.parentPath" title="Up (Backspace)" @click="goUp">
      <Icon name="up" />
    </button>

    <div class="pathbar" @click="startEdit" @dblclick="startEdit">
      <template v-if="!editing">
        <template v-for="(crumb, i) in crumbs" :key="crumb.path">
          <span v-if="i > 0" class="crumb-sep">/</span>
          <button
            class="crumb"
            :class="{ current: i === crumbs.length - 1 }"
            :title="crumb.path"
            @click.stop="navigate(crumb.path)"
          >
            <Icon v-if="i === 0" name="home" style="width: 13px; height: 13px" />
            <span>{{ i === 0 ? '' : crumb.name }}</span>
          </button>
        </template>
      </template>
      <input
        v-else
        ref="pathInput"
        v-model="editValue"
        spellcheck="false"
        @keydown.enter.prevent="commitEdit"
        @keydown.esc.prevent="editing = false"
        @blur="commitEdit"
      />
    </div>

    <div class="searchbox">
      <Icon name="search" style="width: 15px; height: 15px; color: var(--text-faint)" />
      <input
        v-model="query"
        placeholder="Search…"
        spellcheck="false"
        @input="onSearchInput"
        @keydown.esc="clearSearch"
      />
      <button v-if="state.searchMode" class="icon-btn" style="width: 22px; height: 22px" @click="clearSearch">
        <Icon name="close" />
      </button>
    </div>

    <button class="icon-btn" :class="{ active: state.showHidden }" title="Show hidden files (Ctrl+H)" @click="toggleHidden">
      <Icon :name="state.showHidden ? 'eye' : 'eyeOff'" />
    </button>

    <OperationsPopover />

    <div class="segmented">
      <button :class="{ active: state.viewMode === 'grid' }" title="Grid view" @click="setViewMode('grid')">
        <Icon name="grid" />
      </button>
      <button :class="{ active: state.viewMode === 'list' }" title="List view" @click="setViewMode('list')">
        <Icon name="list" />
      </button>
    </div>

    <button class="icon-btn" :class="{ active: state.splitView }" title="Split view (F3)" @click="toggleSplit">
      <Icon name="split" />
    </button>

    <button class="icon-btn" :class="{ active: state.inspectorOpen }" title="Inspector (Ctrl+I)" @click="toggleInspector">
      <Icon name="info" />
    </button>

    <button class="icon-btn" title="Command palette (Ctrl+K)" @click="state.paletteOpen = true">
      <Icon name="command" />
    </button>

    <button class="icon-btn" :title="`Theme: ${state.theme}`" @click="cycleTheme">
      <Icon :name="state.resolvedTheme === 'dark' ? 'moon' : 'sun'" />
    </button>

    <button class="icon-btn" title="New folder (Ctrl+Shift+N)" @click="newFolder">
      <Icon name="folderPlus" />
    </button>
  </header>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import Icon from './Icon.vue'
import OperationsPopover from './OperationsPopover.vue'
import { state, setTheme, currentBreadcrumbs } from '../store'
import { goBack, goForward, goUp, navigate, setViewMode, toggleHidden } from '../storeNav'
import { runSearch, cancelSearch, toggleInspector } from '../storeOps'
import { createFolder, createFile } from '../storeFiles'
import { openPrompt } from '../prompt'

const crumbs = computed(() => currentBreadcrumbs.value)
const editing = ref(false)
const editValue = ref('')
const pathInput = ref(null)
const query = ref('')

let searchTimer = null

function startEdit() {
  editing.value = true
  editValue.value = state.cwd
  nextTick(() => {
    pathInput.value && pathInput.value.focus()
    pathInput.value && pathInput.value.select()
  })
}

function commitEdit() {
  const value = (editValue.value || '').trim()
  editing.value = false
  if (value && value !== state.cwd) navigate(value)
}

function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => runSearch(query.value.trim()), 220)
}

function clearSearch() {
  query.value = ''
  cancelSearch()
}

function cycleTheme() {
  const order = ['system', 'light', 'dark']
  const next = order[(order.indexOf(state.theme) + 1) % order.length]
  setTheme(next)
}

function toggleSplit() {
  state.splitView = !state.splitView
  import('../store').then((m) => m.persist())
}

function newFolder() {
  openPrompt({
    title: 'New folder',
    label: 'Folder name',
    value: 'New Folder',
    confirmLabel: 'Create',
    onSubmit: (name) => createFolder(state.cwd, name)
  })
}

function newFile() {
  openPrompt({
    title: 'New file',
    label: 'File name',
    value: 'untitled.txt',
    confirmLabel: 'Create',
    onSubmit: (name) => createFile(state.cwd, name, '')
  })
}

defineExpose({ newFile })
</script>
