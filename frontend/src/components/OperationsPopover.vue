<template>
  <div style="position: relative">
    <button
      class="op-ring"
      :class="{ active: state.ops.length }"
      :title="title"
      @click="open = !open"
    >
      <svg class="track" viewBox="0 0 32 32">
        <circle cx="16" cy="16" r="13" />
      </svg>
      <svg class="bar" viewBox="0 0 32 32">
        <circle
          cx="16" cy="16" r="13"
          :stroke-dasharray="CIRCUMFERENCE"
          :stroke-dashoffset="CIRCUMFERENCE * (1 - opProgress / 100)"
        />
      </svg>
      <span class="pct">{{ activeOps.length ? Math.round(opProgress) : '' }}</span>
    </button>

    <div v-if="open" class="popover op-popover" style="top: 46px; right: 0" @mousedown.stop>
      <div class="menu-label">Operations</div>
      <p v-if="!state.ops.length" class="hint" style="padding: 8px 10px">
        No recent operations. Copies, moves, extractions, and trash actions appear here with a
        cancel button while they run.
      </p>

      <div v-for="op in state.ops" :key="op.id" class="op-card">
        <div class="title">
          <span class="kind">{{ op.kind }}</span>
          <span class="grow truncate">{{ opTitle(op) }}</span>
          <span class="pill" :class="stateClass(op)">{{ op.state }}</span>
        </div>

        <div class="bar">
          <span :style="{ width: Math.min(100, op.progress || 0) + '%' }" />
        </div>

        <div class="meta">
          <span>{{ Math.round(op.progress || 0) }}%</span>
          <span>{{ formatSize(op.doneBytes) }} / {{ formatSize(op.totalBytes) }}</span>
          <span v-if="op.speedBps">{{ formatSpeed(op.speedBps) }}</span>
          <span v-if="op.etaSeconds">{{ formatEta(op.etaSeconds) }}</span>
        </div>

        <div v-if="op.currentFile" class="file" :title="op.currentFile">{{ op.currentFile }}</div>
        <div v-if="op.message" class="hint">{{ op.message }}</div>
        <div v-if="op.error" class="hint" style="color: var(--error)">{{ op.error }}</div>

        <div class="op-actions">
          <button
            v-if="op.state === 'running'"
            class="btn subtle" title="Pause" @click="pauseOp(op.id)"
          >
            <Icon name="pause" /> Pause
          </button>
          <button
            v-if="op.state === 'paused'"
            class="btn subtle" title="Resume" @click="resumeOp(op.id)"
          >
            <Icon name="play" /> Resume
          </button>
          <span v-if="op.state === 'cancelling'" class="pill warn">cancelling…</span>
          <button
            v-if="op.state === 'running' || op.state === 'paused'"
            class="btn subtle danger" title="Cancel and remove partial files" @click="cancelOp(op.id)"
          >
            <Icon name="cancel" /> Cancel
          </button>
          <button v-if="op.canUndo" class="btn subtle" @click="undoOp(op.id)">
            <Icon name="undo" /> Undo
          </button>
        </div>
      </div>

      <div v-if="state.ops.length" class="op-actions" style="padding: 6px">
        <button class="btn subtle" @click="clearFinishedOps()">Clear finished</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import { state, activeOps, opProgress } from '../store'
import { cancelOp, pauseOp, resumeOp, undoOp, clearFinishedOps } from '../storeOps'
import { formatSize, formatSpeed, formatEta } from '../utils/format'

const CIRCUMFERENCE = 2 * Math.PI * 13
const open = ref(false)

const title = computed(() =>
  activeOps.value.length
    ? `${activeOps.value.length} operation(s) running — click to manage`
    : 'Operations'
)

function opTitle(op) {
  const name = (op.destDir || '').split('/').filter(Boolean).pop() || op.destDir
  switch (op.kind) {
    case 'copy': return `Copying to ${name}`
    case 'move': return `Moving to ${name}`
    case 'extract': return `Extracting into ${name}`
    default: return `${op.kind} ${name || ''}`
  }
}

function stateClass(op) {
  if (op.state === 'error') return 'danger'
  if (op.state === 'completed') return ''
  if (op.state === 'cancelled') return 'warn'
  return 'warn'
}
</script>
