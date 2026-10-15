<template>
  <svg v-if="!thumbnail" viewBox="0 0 24 24" :class="tint" aria-hidden="true">
    <path :d="iconPath" />
  </svg>
  <img v-else :src="thumbnail" :alt="entry?.name || ''" loading="lazy" />
</template>

<script setup>
import { computed } from 'vue'
import { state } from '../store'
import { iconPath as pathFor, iconTint } from '../utils/format'

const props = defineProps({
  entry: { type: Object, required: true },
  allowThumbnail: { type: Boolean, default: true }
})

const thumbnail = computed(() => {
  if (!props.allowThumbnail) return null
  if (!props.entry || props.entry.isDir || props.entry.category !== 'image') return null
  return state.thumbs[props.entry.path] || null
})

const iconPath = computed(() => pathFor(props.entry))
const tint = computed(() => iconTint(props.entry))
</script>
