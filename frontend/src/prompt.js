import { reactive } from 'vue'

/**
 * Tiny promise-less prompt controller: any component can call openPrompt(...)
 * to request a single-field dialog, and PromptModal renders it.
 */
export const promptState = reactive({
  open: false,
  title: '',
  label: '',
  value: '',
  placeholder: '',
  confirmLabel: 'OK',
  onSubmit: null
})

export function openPrompt({ title, label, value = '', placeholder = '', confirmLabel = 'OK', onSubmit }) {
  promptState.open = true
  promptState.title = title
  promptState.label = label
  promptState.value = value
  promptState.placeholder = placeholder
  promptState.confirmLabel = confirmLabel
  promptState.onSubmit = onSubmit
}

export function closePrompt() {
  promptState.open = false
  promptState.onSubmit = null
}

export async function submitPrompt(value) {
  const handler = promptState.onSubmit
  closePrompt()
  if (handler && value) await handler(value)
}
