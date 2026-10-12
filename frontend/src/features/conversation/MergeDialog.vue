<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="w-[80vw] max-w-4xl">
      <DialogHeader>
        <DialogTitle>Merge Conversations</DialogTitle>
        <DialogDescription>
          All messages and tags from secondary tickets will be moved into the primary ticket.
          Secondary tickets will be closed.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-3 py-2">
        <!-- Add ticket by reference number -->
        <div class="space-y-1.5">
          <p class="text-sm font-medium">Search by ticket number, subject or email:</p>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <span class="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground text-sm">#</span>
              <Input
                v-model="ticketNumber"
                placeholder="e.g. 18942, or part of a subject"
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
              <span class="text-sm truncate">{{ r.subject || 'No subject' }}</span>
            </button>
          </div>
        </div>

        <!-- Other conversations from the same contact (matched by email) -->
        <div v-if="suggestions.length > 0" class="space-y-1.5">
          <p class="text-sm font-medium">Other conversations from this contact:</p>
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
              <span class="text-sm truncate">{{ conv.subject || 'No subject' }}</span>
            </button>
          </div>
        </div>

        <!-- Ticket list with primary selection -->
        <div v-if="mergeTickets.length > 0">
          <p class="text-sm font-medium mb-1.5">Select the primary ticket:</p>
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
                    <span class="text-sm font-medium truncate">{{ conv.subject || 'No subject' }}</span>
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
          Add at least 2 tickets to merge
        </div>

        <div v-if="mergeTickets.length >= 2" class="flex items-center gap-2 p-2 rounded-md bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300 text-xs">
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>This action cannot be undone. Messages will be permanently moved.</span>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="$emit('update:open', false)">Cancel</Button>
        <Button variant="destructive" @click="handleMerge" :disabled="mergeTickets.length < 2 || !primaryUUID || merging">
          <Loader2 v-if="merging" class="w-4 h-4 mr-1 animate-spin" />
          Merge {{ mergeTickets.length }} tickets
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
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AlertTriangle, Loader2, Plus, X } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const props = defineProps({
  open: Boolean,
  // Optional: pre-loaded conversation (when opened from single ticket view)
  initialConversation: {
    type: Object,
    default: null
  }
})
const emit = defineEmits(['update:open'])

const conversationStore = useConversationStore()
const emitter = useEmitter()
const merging = ref(false)
const lookingUp = ref(false)
const lookupError = ref('')
const ticketNumber = ref('')
const mergeTickets = ref([])
const primaryUUID = ref('')
const suggestions = ref([])
const searchResults = ref([])

// Initialize when dialog opens
watch(() => props.open, (val) => {
  if (!val) return

  mergeTickets.value = []
  primaryUUID.value = ''
  ticketNumber.value = ''
  lookupError.value = ''
  suggestions.value = []
  searchResults.value = []

  // If opened from single ticket view, add that ticket
  if (props.initialConversation) {
    mergeTickets.value.push({ ...props.initialConversation })
    primaryUUID.value = props.initialConversation.uuid
  }

  // If opened from bulk selection, add selected tickets
  if (conversationStore.selectedUUIDs.size > 0) {
    const uuids = [...conversationStore.selectedUUIDs]
    const selected = conversationStore.conversationsList.filter(c => uuids.includes(c.uuid))
    for (const conv of selected) {
      if (!mergeTickets.value.some(t => t.uuid === conv.uuid)) {
        mergeTickets.value.push({ ...conv })
      }
    }
  }
  pickDefaultPrimary()
  loadSuggestions()
})

// Other conversations from the same contact, excluding ones already merged away.
async function loadSuggestions() {
  const contactID = props.initialConversation?.contact_id
  if (!contactID) return
  try {
    const res = await api.getContactConversations(contactID)
    suggestions.value = (res.data?.data || []).filter(
      c => !mergeTickets.value.some(t => t.uuid === c.uuid) && !c.merged_into_id
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
  if (mergeTickets.value.some(t => t.uuid === conv.uuid)) return
  mergeTickets.value.push(conv)
  suggestions.value = suggestions.value.filter(c => c.uuid !== conv.uuid)
  pickDefaultPrimary()
}

// Fetch full details (search results don't include contact info), then add.
async function addByUUID(uuid, num) {
  if (mergeTickets.value.some(t => t.uuid === uuid)) {
    lookupError.value = `Ticket #${num} is already in the list`
    return false
  }
  lookupError.value = ''
  try {
    const fullRes = await api.getConversation(uuid)
    const fullConv = fullRes.data?.data
    if (!fullConv) {
      lookupError.value = `Ticket #${num} not found`
      return false
    }
    if (fullConv.merged_into_id) {
      lookupError.value = `Ticket #${num} is already merged into another ticket`
      return false
    }
    addTicket(fullConv)
    ticketNumber.value = ''
    searchResults.value = []
    return true
  } catch (err) {
    lookupError.value = 'Failed to look up ticket'
    return false
  }
}

async function lookupTicket() {
  const input = ticketNumber.value.trim().replace(/^#/, '')
  if (!input) return
  lookupError.value = ''
  searchResults.value = []

  const isNumber = /^\d+$/.test(input)
  if (isNumber && mergeTickets.value.some(t => String(t.reference_number) === input)) {
    lookupError.value = `Ticket #${input} is already in the list`
    return
  }
  if (input.length < 3) {
    lookupError.value = 'Enter at least 3 characters to search'
    return
  }

  lookingUp.value = true
  try {
    const res = await api.searchConversations({ query: input })
    const results = res.data?.data || []

    if (isNumber) {
      // Exact match by reference number
      const match = results.find(c => String(c.reference_number) === input)
      if (!match) {
        lookupError.value = `Ticket #${input} not found`
        return
      }
      await addByUUID(match.uuid, input)
      return
    }

    // Free text (subject / contact email): let the agent pick from the hits.
    searchResults.value = results.filter(r => !mergeTickets.value.some(t => t.uuid === r.uuid))
    if (searchResults.value.length === 0) {
      lookupError.value = 'No matching tickets found'
    }
  } catch (err) {
    lookupError.value = 'Failed to look up ticket'
  } finally {
    lookingUp.value = false
  }
}

function removeTicket(uuid) {
  mergeTickets.value = mergeTickets.value.filter(t => t.uuid !== uuid)
  if (primaryUUID.value === uuid) pickDefaultPrimary()
}

async function handleMerge() {
  if (!primaryUUID.value || mergeTickets.value.length < 2) return
  merging.value = true
  try {
    const secondaryUUIDs = mergeTickets.value
      .map(c => c.uuid)
      .filter(uuid => uuid !== primaryUUID.value)

    await api.mergeConversations({
      primary_uuid: primaryUUID.value,
      secondary_uuids: secondaryUUIDs
    })

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: `Merged ${secondaryUUIDs.length + 1} conversations`
    })

    conversationStore.clearSelection()
    conversationStore.fetchFirstPageConversations()
    emit('update:open', false)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    merging.value = false
  }
}
</script>
