<template>
  <div class="toast-host">
    <div v-if="state.toast" class="toast" :class="state.toast.kind">
      <span class="dot" />
      <span class="grow">{{ state.toast.message }}</span>
      <button v-if="state.toast.action" class="action" @click="runAction">
        {{ state.toast.action.label }}
      </button>
      <button class="icon-btn" style="width: 22px; height: 22px" @click="dismissToast">
        <Icon name="close" />
      </button>
    </div>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'
import { state, dismissToast } from '../store'

async function runAction() {
  const action = state.toast && state.toast.action
  dismissToast()
  if (action && action.run) await action.run()
}
</script>
