<template>
  <div v-if="state.quickLook" class="overlay" @mousedown.self="closeQuickLook">
    <div class="modal quick-look">
      <div class="modal-header">
        <Icon name="eye" />
        <span class="grow truncate">{{ entry.name }}</span>
        <button class="icon-btn" @click="closeQuickLook"><Icon name="close" /></button>
      </div>

      <div class="quick-look-body">
        <div class="quick-look-media">
          <img
            v-if="state.quickLook.thumbnailUri"
            :src="state.quickLook.thumbnailUri"
            alt="quick look preview"
          />
          <pre
            v-else-if="state.quickLook.textPreview"
            class="preview"
            style="width: 100%; height: 100%; max-height: none; border: none; border-radius: 0"
          >{{ state.quickLook.textPreview }}</pre>
          <div v-else class="empty-state">
            <strong>No inline preview</strong>
            <span>Press Enter to open with {{ defaultAppName }}</span>
          </div>
        </div>

        <div class="quick-look-side">
          <dl class="kv">
            <dt>Type</dt>
            <dd>{{ entry.mimeType }}</dd>
            <dt>Size</dt>
            <dd>{{ entry.isDir ? formatSize(state.quickLook.calculatedSize) : formatSize(entry.size) }}</dd>
            <dt>Modified</dt>
            <dd>{{ formatDate(entry.modTimeUnix) }}</dd>
            <dt>Permissions</dt>
            <dd class="mono">{{ entry.permissions }}</dd>
          </dl>

          <div v-if="state.quickLook.exif" class="badge-row">
            <span v-if="state.quickLook.exif.orientation > 1" class="badge">
              Auto-rotated · {{ state.quickLook.exif.orientationLabel }}
            </span>
            <span v-if="state.quickLook.exif.pixelWidth" class="badge">
              {{ state.quickLook.exif.pixelWidth }} × {{ state.quickLook.exif.pixelHeight }}
            </span>
            <span v-if="state.quickLook.exif.make" class="badge">{{ state.quickLook.exif.make }}</span>
            <span v-if="state.quickLook.exif.model" class="badge">{{ state.quickLook.exif.model }}</span>
            <span v-if="state.quickLook.exif.lensModel" class="badge">{{ state.quickLook.exif.lensModel }}</span>
            <span v-if="state.quickLook.exif.iso" class="badge">ISO {{ state.quickLook.exif.iso }}</span>
            <span v-if="state.quickLook.exif.fNumber" class="badge">{{ state.quickLook.exif.fNumber }}</span>
            <span v-if="state.quickLook.exif.exposureTime" class="badge">{{ state.quickLook.exif.exposureTime }}</span>
            <span v-if="state.quickLook.exif.focalLength" class="badge">{{ state.quickLook.exif.focalLength }}</span>
          </div>

          <div v-if="state.quickLook.archiveEntries && state.quickLook.archiveEntries.length">
            <div class="menu-label" style="padding-left: 0">
              Archive ({{ state.quickLook.archiveTotal }} entries)
            </div>
            <div class="scroll-list">
              <div
                v-for="item in state.quickLook.archiveEntries.slice(0, 80)"
                :key="item.name"
                class="row-inline"
                style="padding: 5px 10px"
              >
                <span class="grow truncate mono" style="font-size: 11.5px">{{ item.name }}</span>
                <span class="hint">{{ formatSize(item.size) }}</span>
              </div>
            </div>
          </div>

          <div class="op-actions">
            <button class="btn primary" @click="openIt">Open</button>
            <button class="btn" @click="openWithMenu(entry.path)">Open With…</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { closeQuickLook, openWithMenu } from '../storeOps'
import { openPaths } from '../storeFiles'
import { formatSize, formatDate } from '../utils/format'

const entry = computed(() => (state.quickLook && state.quickLook.entry) || {})
const defaultAppName = computed(() => {
  const app = state.quickLook && state.quickLook.defaultApp
  return app ? app.name : 'the default application'
})

function openIt() {
  const path = entry.value.path
  closeQuickLook()
  openPaths([path])
}
</script>
