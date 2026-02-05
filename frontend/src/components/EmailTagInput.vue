<template>
  <div
    class="flex flex-wrap items-center gap-1.5 w-full min-h-[36px] px-2 py-1.5 text-sm border rounded bg-background focus-within:ring-2 focus-within:ring-ring cursor-text"
    @click="focusInput"
  >
    <!-- Email chips -->
    <span
      v-for="(email, index) in emails"
      :key="index"
      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-muted text-foreground text-xs max-w-[220px]"
    >
      <span class="truncate">{{ email }}</span>
      <button
        type="button"
        @click.stop="removeEmail(index)"
        class="flex-shrink-0 text-muted-foreground hover:text-foreground transition-colors"
      >
        <X class="h-3 w-3" />
      </button>
    </span>
    <!-- Input for new emails -->
    <input
      ref="inputRef"
      type="text"
      :placeholder="emails.length === 0 ? placeholder : ''"
      v-model="inputValue"
      @keydown.enter.prevent="addEmail"
      @keydown.,.prevent="addEmail"
      @keydown.tab="onTab"
      @keydown.backspace="onBackspace"
      @blur="addEmail"
      class="flex-1 min-w-[120px] bg-transparent outline-none text-sm placeholder:text-muted-foreground"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps({
  placeholder: { type: String, default: '' }
})

const modelValue = defineModel({ type: String, default: '' })
const emit = defineEmits(['blur'])

const inputRef = ref(null)
const inputValue = ref('')

// Parse comma-separated string into array of trimmed emails
const emails = computed(() => {
  if (!modelValue.value) return []
  return modelValue.value
    .split(',')
    .map(e => e.trim())
    .filter(e => e.length > 0)
})

const updateModel = (emailArray) => {
  modelValue.value = emailArray.join(', ')
}

const addEmail = () => {
  const val = inputValue.value.trim().replace(/,$/, '').trim()
  if (!val) {
    emit('blur')
    return
  }
  // Support pasting multiple emails separated by commas or spaces
  const newEmails = val.split(/[,;\s]+/).map(e => e.trim()).filter(e => e.length > 0)
  const current = [...emails.value]
  newEmails.forEach(e => {
    if (!current.includes(e)) current.push(e)
  })
  updateModel(current)
  inputValue.value = ''
  emit('blur')
}

const removeEmail = (index) => {
  const current = [...emails.value]
  current.splice(index, 1)
  updateModel(current)
  emit('blur')
}

const onBackspace = () => {
  if (inputValue.value === '' && emails.value.length > 0) {
    removeEmail(emails.value.length - 1)
  }
}

const onTab = (e) => {
  if (inputValue.value.trim()) {
    e.preventDefault()
    addEmail()
  }
}

const focusInput = () => {
  inputRef.value?.focus()
}
</script>
