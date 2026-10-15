<template>
  <div
    class="pane"
    :class="{ 'pane-secondary': isSecondary, 'pane-active': isActivePane }"
    @mousedown="setActivePane(pane)"
  >
    <div class="view-toolbar">
      <slot name="toolbar" />
      <div class="spacer" />
      <span v-if="paneMeta.diskTotal" class="pill" :title="diskTitle">{{ diskLabel }}</span>
    </div>

    <div
      ref="scrollEl"
      class="view-scroll"
      tabindex="0"
      @scroll="onScroll"
      @mousedown="onMouseDown"
      @dblclick="onBlankDoubleClick"
      @contextmenu.prevent="onBlankContextMenu"
      @dragover.prevent="onDragOver"
      @dragleave="onDragLeave"
      @drop.prevent="onDrop"
      @keydown="onKeyDown"
    >
      <div class="view-spacer" :style="{ height: totalHeight + 'px' }">
        <div v-if="viewMode === 'grid'" class="grid-layer" :style="gridLayerStyle">
          <div
            v-for="item in renderedItems"
            :key="item.entry.path"
            class="tile"
            :class="tileClass(item.entry)"
            :style="item.style"
            draggable="true"
            @dragstart.stop="onItemDragStart($event, item.entry)"
            @mousedown.stop="onItemMouseDown($event, item.entry, item.index)"
            @dblclick.stop="activate(item.entry)"
            @contextmenu.stop.prevent="onItemContextMenu($event, item.entry, item.index)"
            @dragover.stop.prevent="onItemDragOver(item.entry)"
            @dragleave.stop="onItemDragLeave(item.entry)"
            @drop.stop.prevent="onItemDrop($event, item.entry)"
          >
            <div class="tile-thumb" :style="{ height: thumbHeight + 'px' }">
              <FileIcon :entry="item.entry" />
              <span v-if="badgeFor(item.entry)" class="tile-badge">{{ badgeFor(item.entry) }}</span>
            </div>
            <input
              v-if="renaming === item.entry.path"
              ref="renameInputs"
              class="rename-input"
              :value="item.entry.name"
              @keydown.enter.prevent="commitRename(item.entry, $event.target.value)"
              @keydown.esc.prevent="cancelRename"
              @blur="commitRename(item.entry, $event.target.value)"
              @mousedown.stop
            />
            <div v-else class="tile-label" :title="item.entry.name">{{ item.entry.name }}</div>
            <div class="tile-meta">{{ relativeTime(item.entry.modTimeUnix) }}</div>
          </div>
        </div>

        <div v-else class="list-layer">
          <div class="list-header">
            <span @click="sortBy('name')">Name</span>
            <span @click="sortBy('size')">Size</span>
            <span @click="sortBy('modified')">Modified</span>
            <span @click="sortBy('type')">Type</span>
          </div>
          <div
            v-for="item in renderedItems"
            :key="item.entry.path"
            class="row"
            :class="{ selected: isSelected(item.entry.path) }"
            :style="item.style"
            @mousedown.stop="onItemMouseDown($event, item.entry, item.index)"
            @dblclick.stop="activate(item.entry)"
            @contextmenu.stop.prevent="onItemContextMenu($event, item.entry, item.index)"
            @dragover.stop.prevent="onItemDragOver(item.entry)"
            @dragleave.stop="onItemDragLeave(item.entry)"
            @drop.stop.prevent="onItemDrop($event, item.entry)"
          >
            <div class="row-name">
              <FileIcon :entry="item.entry" :allow-thumbnail="false" />
              <input
                v-if="renaming === item.entry.path"
                ref="renameInputs"
                class="rename-input"
                :value="item.entry.name"
                @keydown.enter.prevent="commitRename(item.entry, $event.target.value)"
                @keydown.esc.prevent="cancelRename"
                @blur="commitRename(item.entry, $event.target.value)"
                @mousedown.stop
              />
              <span v-else class="label" :title="item.entry.name">{{ item.entry.name }}</span>
            </div>
            <div class="row-cell num">{{ item.entry.isDir ? '—' : formatSize(item.entry.size) }}</div>
            <div class="row-cell">{{ formatDate(item.entry.modTimeUnix) }}</div>
            <div class="row-cell">{{ item.entry.isDir ? 'Folder' : (item.entry.mimeType || item.entry.extension) }}</div>
          </div>
        </div>

        <div v-if="marquee" class="marquee" :style="marqueeStyle" />
      </div>

      <div v-if="!items.length && !loading" class="empty-state">
        <strong>Nothing here</strong>
        <span>Drop files to add them to this folder.</span>
      </div>

      <div v-if="dropActive" class="drop-overlay">
        <strong>Drop to add files here</strong>
        <small>Hold Shift while dropping to move instead of copy</small>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import FileIcon from './FileIcon.vue'
import { state, isSelected } from '../store'
import {
  setSelection, toggleSelection, rangeSelect, selectOnly, clearSelection,
  navigate, navigateSecondary, goUpSecondary, setActivePane, setSort,
  queueVisibleThumbnails
} from '../storeNav'
import { formatSize, formatDate, relativeTime } from '../utils/format'
import { activateEntry, renameEntry } from '../storeFiles'
import { handleDrop } from '../storeFiles2'
import { quickLook } from '../storeOps'

const props = defineProps({
  /** Which pane this view drives: 'primary' or 'secondary' (split view). */
  pane: { type: String, default: 'primary' }
})

const emit = defineEmits(['context-menu', 'selection-change'])

const isSecondary = computed(() => props.pane === 'secondary')
const isActivePane = computed(() =>
  isSecondary.value ? state.activePane === 'secondary' : state.activePane === 'primary'
)

const scrollEl = ref(null)
const scrollTop = ref(0)
const viewportH = ref(600)
const viewportW = ref(800)
const renaming = ref(null)
const renameInputs = ref([])
const dropTargetPath = ref('')
const marquee = ref(null)

/** Directory metadata for this pane (primary uses the main listing). */
const paneMeta = computed(() => (isSecondary.value ? state.secondary : state.listing))

const items = computed(() => {
  if (isSecondary.value) return state.secondary.entries
  if (state.searchMode) return state.searchResults
  return state.listing.entries
})

const viewMode = computed(() => (isSecondary.value ? 'grid' : state.viewMode))
const iconSize = computed(() => state.iconSize)
const loading = computed(() => state.loading && !isSecondary.value)
const dropActive = computed(() => state.dropActive && !isSecondary.value)

const GAP = 4
const PADDING = 12
const LABEL_H = 62

const columns = computed(() => {
  const usable = Math.max(120, viewportW.value - PADDING * 2)
  return Math.max(1, Math.floor((usable + GAP) / (iconSize.value + GAP)))
})

const thumbHeight = computed(() => Math.round(iconSize.value * 0.78))
const rowHeight = computed(() => thumbHeight.value + LABEL_H)
const rowCount = computed(() => Math.ceil(items.value.length / columns.value))
const totalHeight = computed(() => (viewMode.value === 'grid'
  ? rowCount.value * (rowHeight.value + GAP) + PADDING * 2
  : items.value.length * 34 + 30 + PADDING * 2))

const gridLayerStyle = computed(() => ({
  gridTemplateColumns: `repeat(${columns.value}, minmax(0, 1fr))`,
  gridAutoRows: rowHeight.value + 'px'
}))

const OVERSCAN = 4

const renderedItems = computed(() => {
  const list = items.value
  if (!list.length) return []
  const vh = viewportH.value
  const out = []

  if (viewMode.value === 'grid') {
    const rh = rowHeight.value + GAP
    const startRow = Math.max(0, Math.floor((scrollTop.value - PADDING) / rh) - OVERSCAN)
    const endRow = Math.min(rowCount.value - 1, Math.ceil((scrollTop.value + vh - PADDING) / rh) + OVERSCAN)
    for (let row = startRow; row <= endRow; row++) {
      const top = PADDING + row * rh
      const startIndex = row * columns.value
      const endIndex = Math.min(list.length - 1, startIndex + columns.value - 1)
      for (let i = startIndex; i <= endIndex; i++) {
        out.push({
          entry: list[i],
          index: i,
          row,
          col: i - startIndex,
          style: { gridRow: row + 1, gridColumn: (i - startIndex) + 1 }
        })
      }
    }
    return out
  }

  const rh = 34
  const headerH = 30 + PADDING
  const startIndex = Math.max(0, Math.floor((scrollTop.value - headerH) / rh) - OVERSCAN)
  const endIndex = Math.min(list.length - 1, Math.ceil((scrollTop.value + vh - headerH) / rh) + OVERSCAN)
  for (let i = startIndex; i <= endIndex; i++) {
    out.push({
      entry: list[i],
      index: i,
      style: { position: 'absolute', top: headerH + i * rh + 'px', left: 0, right: 0 }
    })
  }
  return out
})

function onScroll(e) {
  scrollTop.value = e.target.scrollTop
}

function measure() {
  const el = scrollEl.value
  if (!el) return
  viewportH.value = el.clientHeight
  viewportW.value = el.clientWidth
}

let resizeObserver = null
onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(measure)
    resizeObserver.observe(scrollEl.value)
  } else {
    window.addEventListener('resize', measure)
  }
})

onBeforeUnmount(() => {
  if (resizeObserver) resizeObserver.disconnect()
  else window.removeEventListener('resize', measure)
  stopAutoScroll()
})

const diskLabel = computed(() => {
  const free = paneMeta.value.diskFree
  const total = paneMeta.value.diskTotal
  if (!total) return ''
  const pct = Math.round(paneMeta.value.diskUsedPercent || 0)
  return `${formatSize(free)} free · ${pct}% used`
})

const diskTitle = computed(() => {
  const total = paneMeta.value.diskTotal
  if (!total) return ''
  return `${formatSize(paneMeta.value.diskFree)} free of ${formatSize(total)} on ${paneMeta.value.mountPoint || '/'}`
})

function badgeFor(entry) {
  if (!entry) return ''
  if (entry.isDir) return entry.itemCount ? String(entry.itemCount) : ''
  return entry.size ? formatSize(entry.size) : ''
}

function tileClass(entry) {
  return {
    selected: isSelected(entry.path, props.pane),
    cut: state.clipboardCut && state.clipboardCut.includes(entry.path),
    'drop-target': dropTargetPath.value === entry.path
  }
}

function sortBy(key) {
  setSort(key)
}

/* ----------------------------------------------------- rubber-band selection */

let marqueeState = null
let autoScrollTimer = null

function contentPoint(e) {
  const rect = scrollEl.value.getBoundingClientRect()
  return {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top + scrollEl.value.scrollTop
  }
}

function itemRect(index) {
  if (viewMode.value === 'grid') {
    const rh = rowHeight.value + GAP
    const row = Math.floor(index / columns.value)
    const col = index % columns.value
    const usable = viewportW.value - PADDING * 2
    const cellW = (usable - GAP * (columns.value - 1)) / columns.value
    return {
      x: PADDING + col * (cellW + GAP),
      y: PADDING + row * rh,
      w: cellW,
      h: rowHeight.value
    }
  }
  const headerH = 30 + PADDING
  return { x: 0, y: headerH + index * 34, w: viewportW.value, h: 34 }
}

function rectsIntersect(a, b) {
  return !(b.x > a.x + a.w || b.x + b.w < a.x || b.y > a.y + a.h || b.y + b.h < a.y)
}

function onMouseDown(e) {
  if (e.button !== 0) return
  const onBlank = e.target === e.currentTarget ||
    e.target.classList.contains('view-spacer') ||
    e.target.classList.contains('grid-layer') ||
    e.target.classList.contains('list-layer')
  if (!onBlank) return

  scrollEl.value.focus({ preventScroll: true })
  setActivePane(props.pane)
  const additive = e.ctrlKey || e.metaKey
  if (!additive) clearSelection(props.pane)

  marqueeState = {
    start: contentPoint(e),
    additive,
    toggle: additive,
    base: additive ? [...readPaneSelection()] : []
  }
  marquee.value = { x: marqueeState.start.x, y: marqueeState.start.y, w: 0, h: 0 }

  window.addEventListener('mousemove', onMarqueeMove)
  window.addEventListener('mouseup', onMarqueeUp)
}

function onMarqueeMove(e) {
  if (!marqueeState) return
  const point = contentPoint(e)
  const box = {
    x: Math.min(marqueeState.start.x, point.x),
    y: Math.min(marqueeState.start.y, point.y),
    w: Math.abs(point.x - marqueeState.start.x),
    h: Math.abs(point.y - marqueeState.start.y)
  }
  marquee.value = box

  const hit = []
  const list = items.value
  for (let i = 0; i < list.length; i++) {
    if (rectsIntersect(box, itemRect(i))) hit.push(list[i].path)
  }
  if (marqueeState.toggle) {
    const set = new Set(marqueeState.base)
    hit.forEach((p) => (set.has(p) ? set.delete(p) : set.add(p)))
    setSelection([...set], props.pane)
  } else {
    setSelection(hit, props.pane)
  }
  emit('selection-change', readPaneSelection())

  handleAutoScroll(e)
}

function handleAutoScroll(e) {
  const rect = scrollEl.value.getBoundingClientRect()
  const edge = 34
  const top = e.clientY - rect.top
  const bottom = rect.bottom - e.clientY
  let delta = 0
  if (top < edge) delta = -(edge - top) / 2
  else if (bottom < edge) delta = (edge - bottom) / 2

  if (delta === 0) {
    stopAutoScroll()
    return
  }
  if (autoScrollTimer) return
  autoScrollTimer = setInterval(() => {
    if (!marqueeState) return stopAutoScroll()
    scrollEl.value.scrollTop += delta
  }, 16)
}

function stopAutoScroll() {
  if (autoScrollTimer) {
    clearInterval(autoScrollTimer)
    autoScrollTimer = null
  }
}

function onMarqueeUp() {
  window.removeEventListener('mousemove', onMarqueeMove)
  window.removeEventListener('mouseup', onMarqueeUp)
  stopAutoScroll()
  marquee.value = null
  marqueeState = null
}

const marqueeStyle = computed(() => {
  if (!marquee.value) return {}
  return {
    left: marquee.value.x + 'px',
    top: marquee.value.y + 'px',
    width: marquee.value.w + 'px',
    height: marquee.value.h + 'px'
  }
})

/* ------------------------------------------------------------- interaction */

function onItemMouseDown(e, entry, index) {
  if (e.button !== 0) return
  setActivePane(props.pane)
  if (e.ctrlKey || e.metaKey) {
    toggleSelection(entry.path, index, props.pane)
  } else if (e.shiftKey) {
    rangeSelect(index, props.pane)
  } else if (!isSelected(entry.path, props.pane)) {
    selectOnly(entry.path, index, props.pane)
  } else {
    setPaneAnchor(index)
  }
  emit('selection-change', readPaneSelection())
}

async function activate(entry) {
  if (renaming.value === entry.path) return
  if (entry.isDir) {
    if (isSecondary.value) await navigateSecondary(entry.path)
    else await navigate(entry.path)
    return
  }
  await activateEntry(entry)
}

function onBlankDoubleClick(e) {
  if (e.target === e.currentTarget) goUpParent()
}

async function goUpParent() {
  const parent = paneMeta.value.parentPath
  if (!parent) return
  if (isSecondary.value) await goUpSecondary()
  else await navigate(parent)
}

function onItemContextMenu(e, entry, index) {
  setActivePane(props.pane)
  if (!isSelected(entry.path, props.pane)) selectOnly(entry.path, index, props.pane)
  emit('context-menu', { x: e.clientX, y: e.clientY, entry })
}

function onBlankContextMenu(e) {
  setActivePane(props.pane)
  emit('context-menu', { x: e.clientX, y: e.clientY, entry: null })
}

/** Current selection of this pane (used by the marquee and keyboard logic). */
function readPaneSelection() {
  return isSecondary.value ? state.secondary.selection : state.selection
}

function setPaneAnchor(index) {
  if (isSecondary.value) state.secondary.lastAnchor = index
  else state.lastAnchor = index
}

function readPaneAnchor() {
  return isSecondary.value ? state.secondary.lastAnchor : state.lastAnchor
}

/* --------------------------------------------------------------- hit testing */

/**
 * hitTest resolves the folder entry under a viewport coordinate. Used for
 * OS-level drops reported by the Go side (which arrive with window x/y, not a
 * DOM target), so dropping onto a folder tile moves/copies into that folder.
 */
function hitTest(clientX, clientY) {
  const el = scrollEl.value
  if (!el) return null
  const rect = el.getBoundingClientRect()
  if (clientX < rect.left || clientX > rect.right || clientY < rect.top || clientY > rect.bottom) {
    return null
  }
  const x = clientX - rect.left
  const y = clientY - rect.top + el.scrollTop

  for (let i = 0; i < items.value.length; i++) {
    const r = itemRect(i)
    if (x >= r.x && x <= r.x + r.w && y >= r.y && y <= r.y + r.h) {
      const entry = items.value[i]
      return entry && entry.isDir ? entry.path : null
    }
  }
  return null
}

defineExpose({ startRename, scrollEl, hitTest })

/* ---------------------------------------------------------- drag and drop */

function onDragOver(e) {
  if (e.dataTransfer && Array.from(e.dataTransfer.types || []).includes('Files')) {
    state.dropActive = true
  }
}

function onDragLeave(e) {
  if (!e.currentTarget.contains(e.relatedTarget)) {
    state.dropActive = false
    dropTargetPath.value = ''
  }
}

function readInternalDrag(e) {
  try {
    const raw = e.dataTransfer.getData('application/x-vela-paths')
    if (raw) return JSON.parse(raw)
  } catch {
    /* not an internal drag */
  }
  return null
}

async function onDrop(e, folder) {
  state.dropActive = false
  const internal = readInternalDrag(e)
  const destDir = folder && folder.isDir ? folder.path : paneMeta.value.cwd
  const move = e.shiftKey

  if (internal && internal.paths) {
    const sources = internal.paths.filter((p) => p !== destDir)
    if (!sources.length) return
    const { call } = await import('../api')
    await call(move ? 'Move' : 'Copy', sources, destDir, state.conflictMode)
    return
  }

  const files = e.dataTransfer && e.dataTransfer.files ? Array.from(e.dataTransfer.files) : []
  const paths = files.map((f) => f.path || f.webkitRelativePath).filter(Boolean)
  if (paths.length) {
    await handleDrop(paths, destDir, move)
    return
  }
  if (files.length) {
    // WebKit does not expose absolute paths to the webview: the Go-side
    // OnFileDrop handler receives them instead (see drop:received event).
    state.dropActive = true
  }
}

function onItemDragOver(entry) {
  if (entry && entry.isDir) dropTargetPath.value = entry.path
}

function onItemDragLeave(entry) {
  if (dropTargetPath.value === entry.path) dropTargetPath.value = ''
}

function onItemDrop(e, entry) {
  dropTargetPath.value = ''
  return onDrop(e, entry)
}

function onItemDragStart(e, entry) {
  if (!isSelected(entry.path)) selectOnly(entry.path)
  const payload = JSON.stringify({ paths: state.selection, cut: false })
  e.dataTransfer.setData('application/x-vela-paths', payload)
  e.dataTransfer.effectAllowed = 'copyMove'
  if (typeof e.dataTransfer.setDragImage === 'function' && e.target) {
    e.dataTransfer.setDragImage(e.target, 24, 24)
  }
}

/* ------------------------------------------------------------- inline rename */

function startRename(path) {
  renaming.value = path
  nextTick(() => {
    const inputs = renameInputs.value
    if (!inputs || !inputs.length) return
    const input = inputs[0]
    input.focus()
    const dot = input.value.lastIndexOf('.')
    input.setSelectionRange(0, dot > 0 ? dot : input.value.length)
  })
}

async function commitRename(entry, value) {
  const name = (value || '').trim()
  renaming.value = null
  if (!name || name === entry.name) return
  await renameEntry(entry.path, name)
}

function cancelRename() {
  renaming.value = null
}

/* ---------------------------------------------------------------- keyboard */

function moveFocus(delta) {
  const list = items.value
  if (!list.length) return
  const selection = readPaneSelection()
  const current = selection.length ? list.findIndex((e) => e.path === selection[0]) : -1
  const next = Math.min(list.length - 1, Math.max(0, (current < 0 ? 0 : current) + delta))
  selectOnly(list[next].path, next, props.pane)
  scrollIntoView(next)
}

function scrollIntoView(index) {
  const rect = itemRect(index)
  const el = scrollEl.value
  if (!el) return
  if (rect.y < el.scrollTop) el.scrollTop = rect.y - 8
  else if (rect.y + rect.h > el.scrollTop + el.clientHeight) {
    el.scrollTop = rect.y + rect.h - el.clientHeight + 8
  }
}

/**
 * Handles a key inside this pane. Returns true when the event was consumed, so
 * the shell's global shortcut handler never double-fires the same action
 * (e.g. Space opening and instantly closing Quick Look).
 */
async function handlePaneKey(e) {
  const cols = columns.value
  const step = viewMode.value === 'grid' ? cols : 1
  const selection = readPaneSelection()

  switch (e.key) {
    case 'ArrowDown': moveFocus(step); return true
    case 'ArrowUp': moveFocus(-step); return true
    case 'ArrowRight': moveFocus(1); return true
    case 'ArrowLeft': moveFocus(-1); return true
    case 'Home':
      if (items.value.length) { selectOnly(items.value[0].path, 0, props.pane); scrollIntoView(0) }
      return true
    case 'End':
      if (items.value.length) {
        const last = items.value.length - 1
        selectOnly(items.value[last].path, last, props.pane)
        scrollIntoView(last)
      }
      return true
    case 'Enter': {
      const entry = items.value.find((x) => x.path === selection[0])
      if (entry) await activate(entry)
      return true
    }
    case 'F2':
      if (selection.length) startRename(selection[0])
      return true
    case ' ':
      if (selection.length) await quickLook(selection[0])
      return true
    case 'Backspace':
      await goUpParent()
      return true
    case 'Escape':
      clearSelection(props.pane)
      return true
    case 'a':
    case 'A':
      if (e.ctrlKey || e.metaKey) {
        setSelection(items.value.map((x) => x.path), props.pane)
        return true
      }
      return false
    default:
      return false
  }
}

async function onKeyDown(e) {
  if (await handlePaneKey(e)) {
    e.preventDefault()
    e.stopPropagation()
  }
}

watch(() => paneMeta.value.cwd, () => {
  if (scrollEl.value) scrollEl.value.scrollTop = 0
  scrollTop.value = 0
  renaming.value = null
})

watch(() => items.value.length, () => queueVisibleThumbnails(items.value))
</script>
