<template>
  <div class="relative">
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
        @keydown.enter.prevent="onEnter"
        @keydown.,.prevent="addEmail"
        @keydown.tab="onTab"
        @keydown.backspace="onBackspace"
        @keydown.down.prevent="highlightNext"
        @keydown.up.prevent="highlightPrev"
        @keydown.escape="closeSuggestions"
        @blur="onBlur"
        @input="onInputChange"
        class="flex-1 min-w-[120px] bg-transparent outline-none text-sm placeholder:text-muted-foreground"
      />
    </div>

    <!-- Contact suggestions dropdown -->
    <div
      v-if="showSuggestions && suggestions.length > 0"
      class="absolute z-50 mt-1 w-full max-h-48 overflow-y-auto bg-background border rounded-md shadow-lg"
    >
      <div
        v-for="(contact, index) in suggestions"
        :key="contact.id || index"
        @mousedown.prevent="selectSuggestion(contact)"
        class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer hover:bg-muted transition-colors"
        :class="{ 'bg-muted': index === highlightedIndex }"
      >
        <div class="flex flex-col min-w-0">
          <span class="font-medium truncate" v-if="contact.name">{{ contact.name }}</span>
          <span class="text-muted-foreground truncate text-xs">{{ contact.email }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { X } from 'lucide-vue-next'
import api from '@/api'

const props = defineProps({
  placeholder: { type: String, default: '' }
})

const modelValue = defineModel({ type: String, default: '' })
const emit = defineEmits(['blur'])

const inputRef = ref(null)
const inputValue = ref('')
const suggestions = ref([])
const showSuggestions = ref(false)
const highlightedIndex = ref(-1)
let searchTimeout = null

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
  const newEmails = val.split(/[,;\s]+/).map(e => e.trim()).filter(e => e.length > 0)
  const current = [...emails.value]
  newEmails.forEach(e => {
    if (!current.includes(e)) current.push(e)
  })
  updateModel(current)
  inputValue.value = ''
  closeSuggestions()
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
  if (highlightedIndex.value >= 0 && showSuggestions.value && suggestions.value.length > 0) {
    e.preventDefault()
    selectSuggestion(suggestions.value[highlightedIndex.value])
    return
  }
  if (inputValue.value.trim()) {
    e.preventDefault()
    addEmail()
  }
}

const onEnter = () => {
  if (highlightedIndex.value >= 0 && showSuggestions.value && suggestions.value.length > 0) {
    selectSuggestion(suggestions.value[highlightedIndex.value])
    return
  }
  addEmail()
}

const onBlur = () => {
  // Delay to allow click on suggestion
  setTimeout(() => {
    closeSuggestions()
    addEmail()
  }, 200)
}

const focusInput = () => {
  inputRef.value?.focus()
}

// Contact search
const onInputChange = () => {
  const query = inputValue.value.trim()
  if (searchTimeout) clearTimeout(searchTimeout)

  if (query.length < 2) {
    closeSuggestions()
    return
  }

  searchTimeout = setTimeout(async () => {
    try {
      const resp = await api.searchContacts({ query: query })
      const contacts = resp.data?.data || []
      suggestions.value = contacts.map(c => ({
        id: c.id,
        name: [c.first_name, c.last_name].filter(Boolean).join(' '),
        email: c.email
      })).filter(c => c.email && !emails.value.includes(c.email))

      showSuggestions.value = suggestions.value.length > 0
      highlightedIndex.value = -1
    } catch (err) {
      closeSuggestions()
    }
  }, 300)
}

const selectSuggestion = (contact) => {
  if (!contact.email) return
  const current = [...emails.value]
  if (!current.includes(contact.email)) {
    current.push(contact.email)
  }
  updateModel(current)
  inputValue.value = ''
  closeSuggestions()
  emit('blur')
}

const closeSuggestions = () => {
  showSuggestions.value = false
  suggestions.value = []
  highlightedIndex.value = -1
}

const highlightNext = () => {
  if (!showSuggestions.value || suggestions.value.length === 0) return
  highlightedIndex.value = (highlightedIndex.value + 1) % suggestions.value.length
}

const highlightPrev = () => {
  if (!showSuggestions.value || suggestions.value.length === 0) return
  highlightedIndex.value = highlightedIndex.value <= 0
    ? suggestions.value.length - 1
    : highlightedIndex.value - 1
}
</script>
