<template>
  <div v-if="state.dragShelf.length" class="drag-shelf">
    <div class="row-inline">
      <Icon name="drag" />
      <strong class="grow">Outbound drag &amp; clipboard</strong>
      <button class="icon-btn" style="width: 24px; height: 24px" @click="clearDragShelf">
        <Icon name="close" />
      </button>
    </div>

    <div
      class="drag-card"
      draggable="true"
      title="Drag this card into another application"
      @dragstart="onDragStart"
    >
      <Icon name="drag" />
      <span class="grow truncate">{{ state.dragShelf.length }} item(s)</span>
      <span class="hint">drag me</span>
    </div>

    <div class="row-inline">
      <button class="btn grow" @click="copyForLinuxApps">
        <Icon name="copy" /> Copy for Linux apps
      </button>
      <button class="btn" title="Open a native GTK drag window" @click="nativeDrag">
        <Icon name="external" />
      </button>
    </div>

    <p class="hint">
      WebKit cannot start OS-level drags, so this shelf hands the selection to the native helper.
      Paste with <kbd>Ctrl</kbd>+<kbd>V</kbd> in Nautilus, Dolphin, Telegram, browsers, or IDEs.
    </p>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'
import { state } from '../store'
import { copySelection, clearDragShelf } from '../storeFiles2'
import { backend } from '../api'

function onDragStart(e) {
  const uris = state.dragShelf.map((p) => 'file://' + p)
  e.dataTransfer.setData('text/uri-list', uris.join('\r\n'))
  e.dataTransfer.setData('text/plain', state.dragShelf.join('\n'))
  e.dataTransfer.effectAllowed = 'copy'
}

async function copyForLinuxApps() {
  const previous = [...state.selection]
  state.selection = [...state.dragShelf]
  await copySelection(false)
  state.selection = previous
}

async function nativeDrag() {
  try {
    await backend.StartOutboundDrag(state.dragShelf)
  } catch {
    /* helper unavailable */
  }
}
</script>
