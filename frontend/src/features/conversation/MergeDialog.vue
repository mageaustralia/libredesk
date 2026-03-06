<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Merge Conversations</DialogTitle>
        <DialogDescription>
          All messages and tags from secondary tickets will be moved into the primary ticket.
          Secondary tickets will be closed.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-3 py-2">
        <p class="text-sm font-medium">Select the primary ticket:</p>
        <div class="space-y-2 max-h-60 overflow-y-auto">
          <label
            v-for="conv in conversations"
            :key="conv.uuid"
            class="flex items-center gap-3 p-2 rounded-md border cursor-pointer transition-colors"
            :class="primaryUUID === conv.uuid ? 'border-primary bg-primary/5' : 'border-muted hover:bg-muted/30'"
          >
            <input
              type="radio"
              name="primary"
              :value="conv.uuid"
              v-model="primaryUUID"
              class="accent-primary"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-1.5">
                <span class="text-xs font-medium text-muted-foreground">#{{ conv.reference_number }}</span>
                <span class="text-sm font-medium truncate">{{ conv.subject || 'No subject' }}</span>
              </div>
              <span class="text-xs text-muted-foreground">
                {{ conv.contact?.first_name }} {{ conv.contact?.last_name }}
              </span>
            </div>
          </label>
        </div>

        <div class="flex items-center gap-2 p-2 rounded-md bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300 text-xs">
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>This action cannot be undone. Messages will be permanently moved.</span>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="$emit('update:open', false)">Cancel</Button>
        <Button @click="handleMerge" :disabled="!primaryUUID || merging">
          <Loader2 v-if="merging" class="w-4 h-4 mr-1 animate-spin" />
          Merge {{ conversations.length }} tickets
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertTriangle, Loader2 } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const props = defineProps({
  open: Boolean
})
const emit = defineEmits(['update:open'])

const conversationStore = useConversationStore()
const emitter = useEmitter()
const merging = ref(false)

const conversations = computed(() => {
  const uuids = [...conversationStore.selectedUUIDs]
  return conversationStore.conversationsList.filter(c => uuids.includes(c.uuid))
})

// Default primary = first selected
const primaryUUID = ref('')

// Set default when dialog opens
import { watch } from 'vue'
watch(() => props.open, (val) => {
  if (val && conversations.value.length > 0) {
    primaryUUID.value = conversations.value[0].uuid
  }
})

async function handleMerge() {
  if (!primaryUUID.value) return
  merging.value = true
  try {
    const secondaryUUIDs = conversations.value
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
