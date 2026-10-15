<template>
  <aside class="sidebar">
    <div v-if="state.sidebar.userName" class="sidebar-section" style="padding-top: 2px">
      {{ state.sidebar.userName }}@{{ state.sidebar.hostName }}
    </div>

    <button
      v-for="place in places"
      :key="place.id"
      class="place"
      :class="{ active: isActive(place.path) }"
      :title="place.path"
      @click="go(place)"
      @contextmenu.prevent="place.kind === 'trash' ? null : addBookmark(place.path)"
    >
      <Icon :name="place.icon" />
      <span class="label">{{ place.name }}</span>
      <span v-if="place.badge" class="count">{{ place.badge }}</span>
    </button>

    <template v-if="volumes.length">
      <div class="sidebar-section">Drives &amp; Mounts</div>
      <button
        v-for="volume in volumes"
        :key="volume.mountPoint"
        class="volume"
        :class="{ active: isActive(volume.mountPoint) }"
        :title="`${volume.device} · ${volume.fsType}`"
        @click="navigate(volume.mountPoint)"
      >
        <span class="volume-head">
          <Icon :name="volume.isRemovable ? 'usb' : 'drive'" />
          <span class="label">{{ volume.name }}</span>
          <span class="free">{{ formatSize(volume.freeBytes) }} free</span>
        </span>
        <span class="meter" :class="meterClass(volume)">
          <span :style="{ width: Math.min(100, volume.usedPercent) + '%' }" />
        </span>
        <span v-if="volume.trashCount" class="hint">
          {{ volume.trashCount }} item(s) in {{ volume.mountPoint }}/.Trash
        </span>
      </button>
    </template>

    <template v-if="bookmarks.length">
      <div class="sidebar-section">Bookmarks</div>
      <button
        v-for="mark in bookmarks"
        :key="mark.path"
        class="place"
        :class="{ active: isActive(mark.path) }"
        :title="mark.path"
        @click="navigate(mark.path)"
        @contextmenu.prevent="removeBookmark(mark.path)"
      >
        <Icon name="bookmark" />
        <span class="label">{{ mark.name }}</span>
      </button>
    </template>

    <div class="spacer" style="flex: 1" />

    <button class="place" title="Bookmark the current folder" @click="addBookmark(state.cwd)">
      <Icon name="plus" />
      <span class="label">Bookmark this folder</span>
    </button>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { navigate } from '../storeNav'
import { addBookmark, removeBookmark } from '../storeFiles2'
import { formatSize } from '../utils/format'

const places = computed(() => state.sidebar.places || [])
const volumes = computed(() => state.sidebar.volumes || [])
const bookmarks = computed(() => state.sidebar.bookmarks || [])

function isActive(path) {
  return state.cwd === path || (path !== '/' && state.cwd.startsWith(path + '/'))
}

function meterClass(volume) {
  if (volume.usedPercent > 92) return 'danger'
  if (volume.usedPercent > 78) return 'warn'
  return ''
}

function go(place) {
  navigate(place.path)
}
</script>
