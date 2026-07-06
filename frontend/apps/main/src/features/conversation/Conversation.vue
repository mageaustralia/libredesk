<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between">
      <div>
        <span>{{ conversationStore.currentContactName }}</span>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <div
              class="flex items-center space-x-1 cursor-pointer px-2 py-1 rounded text-sm font-medium"
              :style="currentStatusStyle"
              v-if="conversationStore.current?.status && !conversationStore.conversation.loading"
            >
              <span class="font-medium inline-block">
                {{ conversationStore.current?.status }}
              </span>
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              v-for="status in conversationStore.statusOptions"
              :key="status.value"
              @click="handleUpdateStatus(status.label)"
            >
              {{ status.label }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Messages & reply box -->
    <div class="flex flex-col flex-grow overflow-hidden">
      <MessageList class="flex-1 overflow-y-auto" />
      <ReplyBox />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useConversationStore } from '../../stores/conversation'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import { statusColorStyle } from '../../constants/statusColors'
import { useEmitter } from '../../composables/useEmitter'
const conversationStore = useConversationStore()
const emitter = useEmitter()

// Match the pill colour to the configured status colour (FS17). Falls back
// to gray when the status row hasn't loaded yet or when the conversation's
// status name doesn't match any known row.
const currentStatusStyle = computed(() => {
  const name = conversationStore.current?.status
  if (!name) return statusColorStyle(undefined)
  const match = (conversationStore.statuses || []).find(s => s.name === name)
  return statusColorStyle(match?.color)
})

const handleUpdateStatus = (status) => {
  if (status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
      command: 'snooze',
      open: true
    })
    return
  }
  conversationStore.updateStatus(status)
}
</script>
