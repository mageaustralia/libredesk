<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('conversation.merge.title') }}</DialogTitle>
        <DialogDescription>
          {{ t('conversation.merge.description') }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-3 py-2">
        <!-- Add ticket by reference number -->
        <div class="space-y-1.5">
          <p class="text-sm font-medium">{{ t('conversation.merge.addByNumber') }}</p>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <span class="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground text-sm">#</span>
              <Input
                v-model="ticketNumber"
                :placeholder="t('conversation.merge.numberPlaceholder')"
                class="pl-6"
                @keydown.enter.prevent="lookupTicket"
              />
            </div>
            <Button size="sm" @click="lookupTicket" :disabled="!ticketNumber.trim() || lookingUp">
              <Loader2 v-if="lookingUp" class="w-4 h-4 animate-spin" />
              <Plus v-else class="w-4 h-4" />
            </Button>
          </div>
          <p v-if="lookupError" class="text-xs text-destructive">{{ lookupError }}</p>
          <div v-if="searchResults.length > 0" class="space-y-1 max-h-40 overflow-y-auto">
            <button
              v-for="r in searchResults"
              :key="r.uuid"
              type="button"
              class="flex items-center gap-2 w-full p-2 rounded-md border border-muted text-left hover:bg-muted/30"
              @click="addByUUID(r.uuid, r.reference_number)"
            >
              <Plus class="w-3 h-3 shrink-0 text-muted-foreground" />
              <span class="text-xs font-medium text-muted-foreground shrink-0">#{{ r.reference_number }}</span>
              <span class="text-sm truncate">{{ r.subject || t('conversation.list.noSubject') }}</span>
            </button>
          </div>
        </div>

        <!-- Other conversations from the same contact -->
        <div v-if="suggestions.length > 0" class="space-y-1.5">
          <p class="text-sm font-medium">{{ t('conversation.merge.contactTickets') }}</p>
          <div class="space-y-1 max-h-40 overflow-y-auto">
            <button
              v-for="conv in suggestions"
              :key="conv.uuid"
              type="button"
              class="flex items-center gap-2 w-full p-2 rounded-md border border-muted text-left hover:bg-muted/30"
              @click="addTicket(conv)"
            >
              <Plus class="w-3 h-3 shrink-0 text-muted-foreground" />
              <span class="text-xs font-medium text-muted-foreground shrink-0">#{{ conv.reference_number }}</span>
              <span class="text-sm truncate">{{ conv.subject || t('conversation.list.noSubject') }}</span>
            </button>
          </div>
        </div>

        <!-- Selected tickets -->
        <div v-if="mergeTickets.length > 0">
          <p class="text-sm font-medium mb-1.5">{{ t('conversation.merge.selectPrimary') }}</p>
          <div class="space-y-2 max-h-60 overflow-y-auto">
            <div
              v-for="conv in mergeTickets"
              :key="conv.uuid"
              class="flex items-center gap-2 p-2 rounded-md border transition-colors"
              :class="primaryUUID === conv.uuid ? 'border-primary bg-primary/5' : 'border-muted hover:bg-muted/30'"
            >
              <label class="flex items-center gap-3 flex-1 min-w-0 cursor-pointer">
                <input
                  type="radio"
                  name="primary"
                  :value="conv.uuid"
                  v-model="primaryUUID"
                  class="accent-primary"
                />
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <span class="text-xs font-medium text-muted-foreground shrink-0">#{{ conv.reference_number }}</span>
                    <span class="text-sm font-medium truncate">{{ conv.subject || t('conversation.list.noSubject') }}</span>
                  </div>
                  <span class="text-xs text-muted-foreground">
                    {{ conv.contact?.first_name || '' }} {{ conv.contact?.last_name || '' }}
                  </span>
                </div>
              </label>
              <Button
                variant="ghost"
                size="icon"
                class="h-6 w-6 shrink-0"
                @click="removeTicket(conv.uuid)"
              >
                <X class="w-3 h-3" />
              </Button>
            </div>
          </div>
        </div>
        <div v-else class="text-sm text-muted-foreground text-center py-4">
          {{ t('conversation.merge.addAtLeastTwo') }}
        </div>

        <div
          v-if="mergeTickets.length >= 2"
          class="flex items-center gap-2 p-2 rounded-md bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300 text-xs"
        >
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>{{ t('conversation.merge.warning') }}</span>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="$emit('update:open', false)">
          {{ t('globals.messages.cancel') }}
        </Button>
        <Button @click="handleMerge" :disabled="mergeTickets.length < 2 || !primaryUUID || merging">
          <Loader2 v-if="merging" class="w-4 h-4 mr-1 animate-spin" />
          {{ t('conversation.merge.mergeNTickets', { count: mergeTickets.length }) }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { AlertTriangle, Loader2, Plus, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useToast } from '@main/composables/useToast'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const props = defineProps({
  open: Boolean,
  // Pre-loaded conversation to seed the dialog when launched from a single
  // ticket view (the user is already looking at one of the tickets to merge).
  initialConversation: {
    type: Object,
    default: null
  }
})
const emit = defineEmits(['update:open', 'merged'])

const { t } = useI18n()
const toast = useToast()

const merging = ref(false)
const lookingUp = ref(false)
const lookupError = ref('')
const ticketNumber = ref('')
const mergeTickets = ref([])
const primaryUUID = ref('')
const suggestions = ref([])
const searchResults = ref([])

// Reset + seed when opened.
watch(() => props.open, (val) => {
  if (!val) return

  mergeTickets.value = []
  primaryUUID.value = ''
  ticketNumber.value = ''
  lookupError.value = ''
  suggestions.value = []
  searchResults.value = []

  if (props.initialConversation && props.initialConversation.uuid) {
    mergeTickets.value.push({ ...props.initialConversation })
    primaryUUID.value = props.initialConversation.uuid
  }
  loadSuggestions()
})

// Other conversations from the same contact, excluding ones already merged away.
async function loadSuggestions() {
  const contactID = props.initialConversation?.contact_id
  if (!contactID) return
  try {
    const res = await api.getContactConversations(contactID)
    suggestions.value = (res.data?.data || []).filter(
      c => !mergeTickets.value.some(tk => tk.uuid === c.uuid) && !c.merged_into_id
    )
  } catch {
    // Suggestions are a convenience; the number/search lookup still works.
  }
}

// The earlier (lower-numbered) ticket is the one a split-off reply gets put
// back onto, so it is the default primary. The agent can still change it.
function pickDefaultPrimary() {
  const sorted = [...mergeTickets.value].sort(
    (a, b) => Number(a.reference_number) - Number(b.reference_number)
  )
  primaryUUID.value = sorted[0]?.uuid || ''
}

function addTicket(conv) {
  if (mergeTickets.value.some(tk => tk.uuid === conv.uuid)) return
  mergeTickets.value.push(conv)
  suggestions.value = suggestions.value.filter(c => c.uuid !== conv.uuid)
  pickDefaultPrimary()
}

// Fetch full details (search results don't include contact info), then add.
async function addByUUID(uuid, num) {
  if (mergeTickets.value.some(tk => tk.uuid === uuid)) {
    lookupError.value = t('conversation.merge.alreadyAdded', { num })
    return false
  }
  lookupError.value = ''
  try {
    const res = await api.getConversation(uuid)
    const conv = res.data?.data
    if (!conv) {
      lookupError.value = t('conversation.merge.notFound', { num })
      return false
    }
    if (conv.merged_into_id) {
      lookupError.value = t('conversation.merge.alreadyMerged', { num })
      return false
    }
    addTicket(conv)
    ticketNumber.value = ''
    searchResults.value = []
    return true
  } catch {
    lookupError.value = t('conversation.merge.lookupFailed')
    return false
  }
}

async function lookupTicket() {
  const input = ticketNumber.value.trim().replace(/^#/, '')
  if (!input) return
  lookupError.value = ''
  searchResults.value = []

  const isNumber = /^\d+$/.test(input)
  if (isNumber && mergeTickets.value.some(tk => String(tk.reference_number) === input)) {
    lookupError.value = t('conversation.merge.alreadyAdded', { num: input })
    return
  }

  lookingUp.value = true
  try {
    if (isNumber) {
      // Dedicated by-ref endpoint — avoids the 3-char minimum on the search
      // endpoint, so tickets #1-#99 can be looked up correctly.
      const res = await api.getConversationByRef(input)
      const conv = res.data?.data
      if (!conv) {
        lookupError.value = t('conversation.merge.notFound', { num: input })
        return
      }
      if (conv.merged_into_id) {
        lookupError.value = t('conversation.merge.alreadyMerged', { num: input })
        return
      }
      addTicket(conv)
      ticketNumber.value = ''
      return
    }

    // Free text (subject / contact email): let the agent pick from the hits.
    if (input.length < 3) {
      lookupError.value = t('conversation.merge.minChars')
      return
    }
    const res = await api.searchConversations({ query: input })
    const results = res.data?.data || []
    searchResults.value = results.filter(r => !mergeTickets.value.some(tk => tk.uuid === r.uuid))
    if (searchResults.value.length === 0) {
      lookupError.value = t('conversation.merge.noMatches')
    }
  } catch (err) {
    lookupError.value = handleHTTPError(err).message
  } finally {
    lookingUp.value = false
  }
}

function removeTicket(uuid) {
  mergeTickets.value = mergeTickets.value.filter(tk => tk.uuid !== uuid)
  if (primaryUUID.value === uuid) pickDefaultPrimary()
}

async function handleMerge() {
  if (!primaryUUID.value || mergeTickets.value.length < 2) return
  merging.value = true
  try {
    const secondaryUUIDs = mergeTickets.value
      .map(c => c.uuid)
      .filter(uuid => uuid !== primaryUUID.value)

    const res = await api.mergeConversations({
      primary_uuid: primaryUUID.value,
      secondary_uuids: secondaryUUIDs
    })

    toast.success(t('conversation.merge.success', { count: mergeTickets.value.length }))

    // Surface partial-failure warnings (messages moved but some status
    // updates failed). The agent should refresh to see the current state.
    const warnings = res.data?.data?.warnings
    if (warnings && warnings.length > 0) {
      toast.warning(`Merge succeeded but some status updates failed: ${warnings.join(', ')}. Please refresh.`)
    }

    emit('merged', { primary_uuid: primaryUUID.value, secondary_uuids: secondaryUUIDs })
    emit('update:open', false)
  } catch (err) {
    toast.error(err)
  } finally {
    merging.value = false
  }
}
</script>
