<template>
  <!--
    Every icon carries an explicit size plus the .vela-icon class: an <svg> with
    only a viewBox falls back to the CSS default of 300x150, which would blow up
    any container that has no dedicated sizing rule (e.g. the view-mode toggle).
  -->
  <svg
    class="vela-icon"
    width="16"
    height="16"
    viewBox="0 0 24 24"
    focusable="false"
    aria-hidden="true"
  >
    <path :d="path" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true }
})

const ICONS = {
  back: 'M15 5l-7 7 7 7',
  forward: 'M9 5l7 7-7 7',
  up: 'M12 19V5M5 12l7-7 7 7',
  home: 'M4 11l8-6 8 6v8a1 1 0 0 1-1 1h-4v-6H9v6H5a1 1 0 0 1-1-1z',
  desktop: 'M3 5h18v11H3zM8 20h8M12 16v4',
  documents: 'M7 3h7l4 4v14H7zM14 3v4h4',
  downloads: 'M12 4v10m0 0l-4-4m4 4l4-4M5 19h14',
  music: 'M9 17V6l10-2v11M9 17a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM19 15a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
  pictures: 'M4 5h16v14H4zM8 10a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM4 17l5-5 4 4 3-3 4 4',
  videos: 'M4 6h11v12H4zM15 10l5-3v10l-5-3',
  trash: 'M5 7h14M10 7V5h4v2M7 7l1 13h8l1-13M10 11v6M14 11v6',
  code: 'M9 8l-4 4 4 4M15 8l4 4-4 4',
  bookmark: 'M7 4h10v16l-5-4-5 4z',
  drive: 'M4 6h16v6H4zM4 16h16v2H4zM8 9h.01M8 17h.01',
  usb: 'M12 3v12M12 15l-3-3m3 3l3-3M9 21h6v-3H9z',
  grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
  list: 'M4 6h16M4 12h16M4 18h16',
  search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM16 16l4 4',
  close: 'M6 6l12 12M18 6L6 18',
  sun: 'M12 6V3M12 21v-3M6 12H3M21 12h-3M7 7L5 5M19 19l-2-2M17 7l2-2M5 19l2-2M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
  moon: 'M20 14a8 8 0 1 1-9-11 7 7 0 0 0 9 11z',
  plus: 'M12 5v14M5 12h14',
  folderPlus: 'M3 6.5A1.5 1.5 0 0 1 4.5 5h4l1.6 2h9.4A1.5 1.5 0 0 1 21 8.5v9A1.5 1.5 0 0 1 19.5 19h-15A1.5 1.5 0 0 1 3 17.5zM12 11v5M9.5 13.5h5',
  filePlus: 'M7 3h7l4 4v14H7zM14 3v4h4M12 12v5M9.5 14.5h5',
  split: 'M4 5h7v14H4zM13 5h7v14h-7z',
  info: 'M12 4a8 8 0 1 0 0 16 8 8 0 0 0 0-16zM12 10v6M12 7.5h.01',
  eye: 'M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6zM12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
  eyeOff: 'M4 4l16 16M9.5 6.3A9.6 9.6 0 0 1 12 6c6.5 0 10 6 10 6a17 17 0 0 1-3 3.5M6.4 8.1C3.6 9.6 2 12 2 12s3.5 6 10 6c1 0 1.9-.1 2.7-.4',
  sort: 'M7 4v16M7 20l-3-3M7 4l3 3M17 20V4M17 4l3 3M17 20l-3-3',
  cut: 'M6 4l12 12M18 4L6 16M6 20a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM18 20a2 2 0 1 0 0-4 2 2 0 0 0 0 4z',
  copy: 'M9 9h10v10H9zM5 15V5h10',
  paste: 'M9 4h6v3H9zM6 6h2M16 6h2v14H6V6h2M9 12h6M9 16h4',
  terminal: 'M4 5h16v14H4zM8 10l2 2-2 2M13 14h3',
  rename: 'M4 20h16M6 16l9-9 3 3-9 9H6z',
  archive: 'M4 6h16v4H4zM5 10h14v8H5zM10 13h4',
  extract: 'M12 4v8m0 0l-3-3m3 3l3-3M5 14v5h14v-5',
  open: 'M14 4h6v6M20 4l-8 8M10 5H6a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-4',
  external: 'M14 4h6v6M20 4l-9 9M11 5H6a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-5',
  restore: 'M4 12a8 8 0 1 0 3-6.2M4 4v5h5',
  undo: 'M9 7H5V3M5 7a8 8 0 1 1-1 7',
  cancel: 'M6 6l12 12M18 6L6 18',
  pause: 'M9 5v14M15 5v14',
  play: 'M7 4l13 8-13 8z',
  chevronRight: 'M9 6l6 6-6 6',
  chevronDown: 'M6 9l6 6 6-6',
  hidden: 'M12 6c6.5 0 10 6 10 6s-3.5 6-10 6-10-6-10-6 3.5-6 10-6zM4 4l16 16',
  command: 'M9 6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3z',
  drag: 'M12 3v18M12 3l-3 3M12 3l3 3M12 21l-3-3M12 21l3-3',
  shield: 'M12 3l7 3v6c0 4-3 7-7 9-4-2-7-5-7-9V6z',
  clock: 'M12 4a8 8 0 1 0 0 16 8 8 0 0 0 0-16zM12 8v4l3 2',
  check: 'M5 13l4 4L19 7'
}

const path = computed(() => ICONS[props.name] || ICONS.file || 'M7 3h7l4 4v14H7z')
</script>
