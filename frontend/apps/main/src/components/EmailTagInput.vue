<template>
  <!--
    EC7/EC8: Gmail-style chip input for email recipient fields.

    Renders the comma-joined email string in v-model as removable pills, plus a
    typing input for adding more. Each chip has its own remove (X) which fully
    supersedes the previous per-field clear-X (EC9) — the chip-level remove is
    finer-grained, so EC9's all-or-nothing clear button is no longer needed.

    Backed by a comma-separated string in modelValue. Both the chip-rendering
    parse and the addEmail typing/paste parse go through `parseEmailList`
    (shared-ui), so chip splitting and downstream backend parsing
    (`stringutil.SplitEmailList`) agree on `,`, `;`, and whitespace as
    delimiters — typing `a@x.com;b@x.com` produces 2 chips AND submits as 2
    addresses.

    Contact suggestions: when the user types ≥2 chars, hits /contacts/search
    and surfaces a dropdown. Selecting a suggestion adds the email as a chip
    AND emits `contactSelected` with the full contact — used by
    CreateConversation.vue to also auto-fill first/last name.
  -->
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
          :title="$t('replyBox.removeRecipient')"
          :aria-label="$t('replyBox.removeRecipient')"
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
import { ref, computed, onUnmounted } from 'vue'
import { X } from 'lucide-vue-next'
import api from '@main/api'
import { parseEmailList } from '@shared-ui/utils/string'

const props = defineProps({
  placeholder: { type: String, default: '' }
})

const modelValue = defineModel({ type: String, default: '' })
const emit = defineEmits(['blur', 'contactSelected'])

const inputRef = ref(null)
const inputValue = ref('')
const suggestions = ref([])
const showSuggestions = ref(false)
const highlightedIndex = ref(-1)
let searchTimeout = null

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
})

// Parse the comma/semicolon/whitespace-separated string into chip emails.
const emails = computed(() => parseEmailList(modelValue.value))

const updateModel = (emailArray) => {
  modelValue.value = emailArray.join(', ')
}

const addEmail = () => {
  const val = inputValue.value.trim().replace(/,$/, '').trim()
  if (!val) {
    emit('blur')
    return
  }
  // Support pasting multiple emails separated by commas / semicolons / whitespace.
  const newEmails = parseEmailList(val)
  const current = [...emails.value]
  newEmails.forEach((e) => {
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
  // Delay to allow a click on a suggestion to fire before blur closes the menu.
  setTimeout(() => {
    closeSuggestions()
    addEmail()
  }, 200)
}

const focusInput = () => {
  inputRef.value?.focus()
}

// Contact search — debounced 300ms; min 2 chars to avoid hammering the API.
const onInputChange = () => {
  const query = inputValue.value.trim()
  if (searchTimeout) clearTimeout(searchTimeout)

  if (query.length < 2) {
    closeSuggestions()
    return
  }

  searchTimeout = setTimeout(async () => {
    try {
      const resp = await api.searchContacts({ query })
      const contacts = resp.data?.data || []
      suggestions.value = contacts
        .map((c) => ({
          id: c.id,
          name: [c.first_name, c.last_name].filter(Boolean).join(' '),
          first_name: c.first_name || '',
          last_name: c.last_name || '',
          email: c.email
        }))
        .filter((c) => c.email && !emails.value.includes(c.email))

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
  emit('contactSelected', contact)
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
  highlightedIndex.value =
    highlightedIndex.value <= 0 ? suggestions.value.length - 1 : highlightedIndex.value - 1
}
</script>
