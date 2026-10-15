<template>
  <div v-if="promptState.open" class="overlay" @mousedown.self="closePrompt">
    <div class="modal" style="width: min(420px, 92vw)">
      <div class="modal-header">{{ promptState.title }}</div>
      <div class="modal-body">
        <div class="field">
          <label>{{ promptState.label }}</label>
          <input
            ref="input"
            v-model="value"
            :placeholder="promptState.placeholder"
            spellcheck="false"
            @keydown.enter.prevent="submit"
            @keydown.esc.prevent="closePrompt"
          />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn" @click="closePrompt">Cancel</button>
        <button class="btn primary" :disabled="!value.trim()" @click="submit">
          {{ promptState.confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import { promptState, closePrompt, submitPrompt } from '../prompt'

const value = ref('')
const input = ref(null)

watch(
  () => promptState.open,
  (open) => {
    if (!open) return
    value.value = promptState.value
    nextTick(() => {
      input.value && input.value.focus()
      input.value && input.value.select()
    })
  }
)

function submit() {
  const v = value.value.trim()
  if (v) submitPrompt(v)
}
</script>
