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
          <p class="text-sm font-medium">Add ticket by number:</p>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <span class="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground text-sm">#</span>
              <Input
                v-model="ticketNumber"
                placeholder="e.g. 105"
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

// Initialize when dialog opens
watch(() => props.open, (val) => {
  if (!val) return

  mergeTickets.value = []
  primaryUUID.value = ''
  ticketNumber.value = ''
  lookupError.value = ''

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
    if (!primaryUUID.value && mergeTickets.value.length > 0) {
      primaryUUID.value = mergeTickets.value[0].uuid
    }
  }
})

async function lookupTicket() {
  const num = ticketNumber.value.trim().replace(/^#/, '')
  if (!num) return

  // Check if already added
  if (mergeTickets.value.some(t => String(t.reference_number) === num)) {
    lookupError.value = `Ticket #${num} is already in the list`
    return
  }

  lookingUp.value = true
  lookupError.value = ''

  try {
    // Search for the ticket by reference number
    const res = await api.searchConversations({ query: num })
    const results = res.data?.data || []

    // Find exact match by reference number
    const match = results.find(c => String(c.reference_number) === num)
    if (!match) {
      lookupError.value = `Ticket #${num} not found`
      return
    }

    // Fetch full conversation details (search results don't include contact info)
    const fullRes = await api.getConversation(match.uuid)
    const fullConv = fullRes.data?.data
    if (!fullConv) {
      lookupError.value = `Ticket #${num} not found`
      return
    }

    // Check if it's already merged
    if (fullConv.merged_into_id) {
      lookupError.value = `Ticket #${num} is already merged into another ticket`
      return
    }

    mergeTickets.value.push(fullConv)
    ticketNumber.value = ''

    // Auto-select primary if this is the first ticket
    if (!primaryUUID.value) {
      primaryUUID.value = fullConv.uuid
    }
  } catch (err) {
    lookupError.value = 'Failed to look up ticket'
  } finally {
    lookingUp.value = false
  }
}

function removeTicket(uuid) {
  mergeTickets.value = mergeTickets.value.filter(t => t.uuid !== uuid)
  if (primaryUUID.value === uuid) {
    primaryUUID.value = mergeTickets.value.length > 0 ? mergeTickets.value[0].uuid : ''
  }
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
