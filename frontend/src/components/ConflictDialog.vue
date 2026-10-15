<template>
  <div v-if="state.conflict" class="overlay">
    <div class="modal" style="width: min(560px, 92vw)">
      <div class="modal-header">
        <Icon name="shield" />
        <span class="grow">An item named “{{ baseName }}” already exists</span>
      </div>
      <div class="modal-body">
        <dl class="kv" style="margin-bottom: 12px">
          <dt>From</dt>
          <dd class="mono">{{ state.conflict.srcPath }}</dd>
          <dt>Size</dt>
          <dd>{{ formatSize(state.conflict.srcSize) }}</dd>
          <dt>Existing</dt>
          <dd class="mono">{{ state.conflict.dstPath }}</dd>
          <dt>Size</dt>
          <dd>{{ formatSize(state.conflict.dstSize) }}</dd>
        </dl>
        <p class="hint">
          Applying to all remaining conflicts uses the same choice for this operation.
        </p>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="choose('skip')">Skip</button>
        <button class="btn" @click="choose('rename')">Keep both (rename)</button>
        <button class="btn danger" @click="choose('overwrite')">Replace</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { resolveConflict } from '../storeOps'

const baseName = computed(() => {
  const c = state.conflict
  if (!c) return ''
  return c.srcName || (c.srcPath || '').split('/').pop()
})

function choose(decision) {
  resolveConflict(decision)
}
</script>
