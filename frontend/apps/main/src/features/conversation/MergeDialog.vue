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
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
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
const emitter = useEmitter()

const merging = ref(false)
const lookingUp = ref(false)
const lookupError = ref('')
const ticketNumber = ref('')
const mergeTickets = ref([])
const primaryUUID = ref('')

// Reset + seed when opened.
watch(() => props.open, (val) => {
  if (!val) return

  mergeTickets.value = []
  primaryUUID.value = ''
  ticketNumber.value = ''
  lookupError.value = ''

  if (props.initialConversation && props.initialConversation.uuid) {
    mergeTickets.value.push({ ...props.initialConversation })
    primaryUUID.value = props.initialConversation.uuid
  }
})

async function lookupTicket() {
  const num = ticketNumber.value.trim().replace(/^#/, '')
  if (!num) return

  if (mergeTickets.value.some(tk => String(tk.reference_number) === num)) {
    lookupError.value = t('conversation.merge.alreadyAdded', { num })
    return
  }

  lookingUp.value = true
  lookupError.value = ''

  try {
    // Use the dedicated by-ref endpoint — avoids the 3-char minimum on the
    // search endpoint, so tickets #1-#99 can be looked up correctly.
    const res = await api.getConversationByRef(num)
    const conv = res.data?.data
    if (!conv) {
      lookupError.value = t('conversation.merge.notFound', { num })
      return
    }

    if (conv.merged_into_id) {
      lookupError.value = t('conversation.merge.alreadyMerged', { num })
      return
    }

    mergeTickets.value.push(conv)
    ticketNumber.value = ''

    if (!primaryUUID.value) {
      primaryUUID.value = conv.uuid
    }
  } catch (err) {
    lookupError.value = handleHTTPError(err).message
  } finally {
    lookingUp.value = false
  }
}

function removeTicket(uuid) {
  mergeTickets.value = mergeTickets.value.filter(tk => tk.uuid !== uuid)
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

    const res = await api.mergeConversations({
      primary_uuid: primaryUUID.value,
      secondary_uuids: secondaryUUIDs
    })

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.merge.success', { count: mergeTickets.value.length })
    })

    // Surface partial-failure warnings (messages moved but some status
    // updates failed). The agent should refresh to see the current state.
    const warnings = res.data?.data?.warnings
    if (warnings && warnings.length > 0) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: `Merge succeeded but some status updates failed: ${warnings.join(', ')}. Please refresh.`
      })
    }

    emit('merged', { primary_uuid: primaryUUID.value, secondary_uuids: secondaryUUIDs })
    emit('update:open', false)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    merging.value = false
  }
}
</script>
