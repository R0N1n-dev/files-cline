<template>
  <aside v-if="state.inspectorOpen" class="inspector">
    <div class="inspector-header">
      <Icon name="info" />
      <span class="grow">Properties</span>
      <button class="icon-btn" @click="state.inspectorOpen = false"><Icon name="close" /></button>
    </div>

    <div v-if="!state.inspector" class="inspector-body hint">Nothing selected.</div>

    <div v-else class="inspector-body">
      <img
        v-if="state.inspector.thumbnailUri"
        class="inspector-preview"
        :src="state.inspector.thumbnailUri"
        alt="preview"
      />

      <div>
        <strong style="display: block; margin-bottom: 6px; overflow-wrap: anywhere">
          {{ state.inspector.entry.name }}
        </strong>
        <span class="pill">{{ state.inspector.entry.mimeType }}</span>
      </div>

      <dl class="kv">
        <dt>Size</dt>
        <dd>{{ state.inspector.entry.isDir ? formatSize(state.inspector.calculatedSize) + ' in files' : formatSize(state.inspector.entry.size) }}</dd>
        <dt>Modified</dt>
        <dd>{{ formatDate(state.inspector.entry.modTimeUnix) }}</dd>
        <dt>Owner</dt>
        <dd>{{ state.inspector.ownerName }}</dd>
        <dt>Permissions</dt>
        <dd class="mono">{{ state.inspector.entry.permissions }}</dd>
        <template v-if="state.inspector.entry.isSymlink">
          <dt>Link target</dt>
          <dd class="mono">{{ state.inspector.entry.symlinkTarget }}</dd>
        </template>
        <template v-if="state.inspector.entry.isDir">
          <dt>Contains</dt>
          <dd>{{ state.inspector.subfolderCount }} folders, {{ state.inspector.subfileCount }} files</dd>
        </template>
        <template v-if="state.inspector.entry.trashId">
          <dt>Trashed from</dt>
          <dd class="mono">{{ state.inspector.entry.originalPath }}</dd>
          <dt>Trash on</dt>
          <dd class="mono">{{ state.inspector.entry.trashMountPoint || 'home' }}</dd>
        </template>
        <template v-if="state.inspector.defaultApp">
          <dt>Opens with</dt>
          <dd>{{ state.inspector.defaultApp.name }}</dd>
        </template>
      </dl>

      <div v-if="state.inspector.exif" class="badge-row">
        <span v-if="state.inspector.exif.orientation > 1" class="badge">
          <Icon name="check" style="width: 12px; height: 12px" />
          EXIF {{ state.inspector.exif.orientationLabel }}
        </span>
        <span v-if="state.inspector.exif.pixelWidth" class="badge">
          {{ state.inspector.exif.pixelWidth }} × {{ state.inspector.exif.pixelHeight }}
        </span>
        <span v-if="state.inspector.exif.model" class="badge">{{ state.inspector.exif.model }}</span>
        <span v-if="state.inspector.exif.iso" class="badge">ISO {{ state.inspector.exif.iso }}</span>
        <span v-if="state.inspector.exif.fNumber" class="badge">{{ state.inspector.exif.fNumber }}</span>
        <span v-if="state.inspector.exif.exposureTime" class="badge">{{ state.inspector.exif.exposureTime }}</span>
        <span v-if="state.inspector.exif.focalLength" class="badge">{{ state.inspector.exif.focalLength }}</span>
        <span v-if="state.inspector.exif.dateTime" class="badge">{{ state.inspector.exif.dateTime }}</span>
      </div>

      <div v-if="state.inspector.archiveEntries && state.inspector.archiveEntries.length">
        <div class="menu-label" style="padding-left: 0">
          Archive contents ({{ state.inspector.archiveTotal }})
        </div>
        <div class="scroll-list">
          <div
            v-for="item in state.inspector.archiveEntries.slice(0, 60)"
            :key="item.name"
            class="row-inline"
            style="padding: 6px 10px"
          >
            <Icon :name="item.isDir ? 'home' : 'open'" style="width: 14px; height: 14px" />
            <span class="grow truncate mono" style="font-size: 11.5px">{{ item.name }}</span>
            <span class="hint">{{ formatSize(item.size) }}</span>
          </div>
        </div>
      </div>

      <div v-if="state.inspector.textPreview">
        <div class="menu-label" style="padding-left: 0">
          Preview ({{ state.inspector.textLineCount }} lines)
        </div>
        <pre class="preview">{{ state.inspector.textPreview.slice(0, 4000) }}</pre>
      </div>
    </div>
  </aside>
</template>

<script setup>
import Icon from './Icon.vue'
import { state } from '../store'
import { formatSize, formatDate } from '../utils/format'
</script>
