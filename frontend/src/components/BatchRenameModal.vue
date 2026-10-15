<template>
  <div v-if="state.batchRenameOpen" class="overlay" @mousedown.self="close">
    <div class="modal wide">
      <div class="modal-header">
        <Icon name="rename" />
        <span class="grow">Batch rename {{ entries.length }} item(s)</span>
        <button class="icon-btn" @click="close"><Icon name="close" /></button>
      </div>

      <div class="modal-body">
        <div class="row-inline" style="align-items: flex-end; gap: 12px; flex-wrap: wrap">
          <div class="field" style="min-width: 150px">
            <label>Find</label>
            <input v-model="find" spellcheck="false" placeholder="text to replace" />
          </div>
          <div class="field" style="min-width: 150px">
            <label>Replace with</label>
            <input v-model="replace" spellcheck="false" placeholder="replacement" />
          </div>
          <div class="field" style="width: 110px">
            <label>Prefix</label>
            <input v-model="prefix" spellcheck="false" />
          </div>
          <div class="field" style="width: 110px">
            <label>Suffix</label>
            <input v-model="suffix" spellcheck="false" />
          </div>
        </div>

        <div class="row-inline" style="align-items: flex-end; gap: 12px; flex-wrap: wrap">
          <div class="field" style="width: 120px">
            <label>Numbering</label>
            <select v-model="numbering">
              <option value="off">Off</option>
              <option value="suffix">After name</option>
              <option value="prefix">Before name</option>
            </select>
          </div>
          <div class="field" style="width: 90px">
            <label>Start at</label>
            <input v-model.number="startAt" type="number" min="0" />
          </div>
          <div class="field" style="width: 90px">
            <label>Digits</label>
            <input v-model.number="digits" type="number" min="1" max="6" />
          </div>
          <div class="field" style="width: 130px">
            <label>Case</label>
            <select v-model="caseMode">
              <option value="keep">Keep</option>
              <option value="lower">lowercase</option>
              <option value="upper">UPPERCASE</option>
            </select>
          </div>
          <div class="field" style="width: 130px">
            <label>Extension</label>
            <input v-model="extension" spellcheck="false" placeholder="keep" />
          </div>
        </div>

        <div class="menu-label" style="padding-left: 0">Preview</div>
        <div class="scroll-list">
          <div v-for="row in preview" :key="row.old" class="row-inline" style="padding: 6px 10px">
            <span class="grow truncate dim">{{ row.old }}</span>
            <Icon name="chevronRight" style="width: 13px; height: 13px; color: var(--text-faint)" />
            <span class="grow truncate mono" :style="{ color: row.conflict ? 'var(--error)' : 'var(--text)' }">
              {{ row.next }}
            </span>
          </div>
        </div>
        <p v-if="conflicts" class="hint" style="color: var(--error)">
          {{ conflicts }} target name(s) collide — adjust the pattern.
        </p>
      </div>

      <div class="modal-footer">
        <button class="btn" @click="close">Cancel</button>
        <button class="btn primary" :disabled="!changed || conflicts > 0" @click="apply">
          Rename {{ changed }} item(s)
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import { state } from '../store'
import { applyBatchRename } from '../storeFiles'

const find = ref('')
const replace = ref('')
const prefix = ref('')
const suffix = ref('')
const numbering = ref('off')
const startAt = ref(1)
const digits = ref(2)
const caseMode = ref('keep')
const extension = ref('')

const entries = computed(() => state.listing.entries.filter((e) => state.selection.includes(e.path)))

function splitName(name) {
  const dot = name.lastIndexOf('.')
  if (dot <= 0) return { stem: name, ext: '' }
  return { stem: name.slice(0, dot), ext: name.slice(dot) }
}

const preview = computed(() =>
  entries.value.map((entry, index) => {
    const { stem, ext } = splitName(entry.name)
    let next = find.value ? stem.split(find.value).join(replace.value) : stem
    if (caseMode.value === 'lower') next = next.toLowerCase()
    if (caseMode.value === 'upper') next = next.toUpperCase()
    next = prefix.value + next + suffix.value
    if (numbering.value !== 'off') {
      const num = String(startAt.value + index).padStart(Math.max(1, digits.value), '0')
      next = numbering.value === 'prefix' ? num + ' ' + next : next + ' ' + num
    }
    let outExt = ext
    if (extension.value.trim()) {
      outExt = extension.value.startsWith('.') ? extension.value.trim() : '.' + extension.value.trim()
    }
    return { old: entry.name, next: next + outExt, path: entry.path }
  })
)

const changed = computed(() => preview.value.filter((r) => r.old !== r.next).length)

const conflicts = computed(() => {
  const seen = new Set()
  let count = 0
  const existing = new Set(state.listing.entries.map((e) => e.name))
  preview.value.forEach((row) => {
    if (seen.has(row.next)) count++
    seen.add(row.next)
    if (row.next !== row.old && existing.has(row.next)) count++
  })
  return count
})

async function apply() {
  const pairs = preview.value
    .filter((r) => r.old !== r.next)
    .map((r) => ({ oldPath: r.path, newName: r.next }))
  await applyBatchRename(pairs)
}

function close() {
  state.batchRenameOpen = false
}
</script>
