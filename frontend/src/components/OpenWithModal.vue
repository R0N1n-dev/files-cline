<template>
  <div v-if="state.openWith" class="overlay" @mousedown.self="state.openWith = null">
    <div class="modal">
      <div class="modal-header">
        <Icon name="external" />
        <span class="grow">Open “{{ baseName }}” with</span>
        <button class="icon-btn" @click="state.openWith = null"><Icon name="close" /></button>
      </div>

      <div class="modal-body">
        <div class="row-inline" style="margin-bottom: 10px">
          <span class="pill">{{ state.openWith.mimeType }}</span>
          <span v-if="state.openWith.defaultApp" class="pill warn">
            Default: {{ state.openWith.defaultApp.name }}
          </span>
        </div>

        <div v-if="recommended.length" class="menu-label">Recommended</div>
        <button
          v-for="app in recommended"
          :key="app.id"
          class="menu-item"
          @click="openWithApp(app.id)"
        >
          <img v-if="app.iconPath" :src="'file://' + app.iconPath" class="app-icon" alt="" />
          <Icon v-else name="open" />
          <span class="grow truncate">{{ app.name }}</span>
          <span v-if="app.isDefault" class="pill">default</span>
        </button>

        <div v-if="others.length" class="menu-label">Other applications</div>
        <div class="scroll-list">
          <button v-for="app in others" :key="app.id" class="menu-item" @click="openWithApp(app.id)">
            <img v-if="app.iconPath" :src="'file://' + app.iconPath" class="app-icon" alt="" />
            <Icon v-else name="open" />
            <span class="grow truncate">{{ app.name }}</span>
            <span v-if="app.comment" class="shortcut truncate" style="max-width: 190px">{{ app.comment }}</span>
          </button>
        </div>

        <div class="field" style="margin-top: 14px">
          <label>Run a custom command</label>
          <div class="row-inline">
            <input v-model="custom" class="grow" placeholder="e.g. gimp %F" spellcheck="false" />
            <button class="btn" :disabled="!custom.trim()" @click="runCustomCommand(custom.trim())">Run</button>
          </div>
          <span class="hint">
            Field codes: %f one file · %F all files · %u/%U URIs. Without codes the selection is appended.
          </span>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn" @click="state.openWith = null">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { openWithApp, runCustomCommand } from '../storeOps'

const custom = ref('')

const baseName = computed(() => {
  const p = state.openWith ? state.openWith.path : ''
  return p.split('/').filter(Boolean).pop() || p
})

const apps = computed(() => (state.openWith && state.openWith.apps) || [])
const recommended = computed(() => apps.value.slice(0, 4))
const others = computed(() => apps.value.slice(4))
</script>

<style scoped>
.app-icon {
  width: 18px;
  height: 18px;
  object-fit: contain;
}
</style>
