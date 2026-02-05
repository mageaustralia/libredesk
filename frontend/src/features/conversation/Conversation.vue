<template>
  <div class="flex flex-col h-screen">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between">
      <div>
        <span v-if="!conversationStore.conversation.loading">
          {{ conversationStore.currentContactName }}
        </span>
        <Skeleton class="w-[130px] h-6" v-else />
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <div
              class="flex items-center space-x-1 cursor-pointer bg-primary px-2 py-1 rounded text-sm"
              v-if="!conversationStore.conversation.loading"
            >
              <span class="text-secondary font-medium inline-block">
                {{ conversationStore.current?.status }}
              </span>
            </div>
            <Skeleton class="w-[70px] h-6 rounded-full" v-else />
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

    <!-- Freshdesk theme: unified scroll with collapsible reply -->
    <div v-if="isFreshdesk" class="flex flex-col flex-grow overflow-hidden">
      <div class="flex-1 overflow-y-auto" ref="scrollContainer">
        <MessageList />

        <!-- Collapsed reply bar -->
        <div
          v-if="!replyExpanded"
          class="sticky bottom-0 bg-background border-t px-4 py-2 flex gap-2"
        >
          <Button size="sm" variant="outline" @click="expandReply('reply')">
            <Reply class="h-4 w-4 mr-1.5" />
            Reply
          </Button>
          <Button size="sm" variant="outline" @click="expandReply('private_note')">
            <StickyNote class="h-4 w-4 mr-1.5" />
            Private note
          </Button>
        </div>

        <!-- Expanded reply box (inside scroll flow) -->
        <div v-if="replyExpanded" class="border-t">
          <ReplyBox ref="replyBoxRef" :initialMessageType="initialMessageType" />
        </div>
      </div>
    </div>

    <!-- Default theme: original layout with sticky reply -->
    <div v-else class="flex flex-col flex-grow overflow-hidden">
      <MessageList class="flex-1 overflow-y-auto" />
      <div class="sticky bottom-0">
        <ReplyBox />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useTheme } from '@/composables/useTheme'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { Reply, StickyNote } from 'lucide-vue-next'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { Skeleton } from '@/components/ui/skeleton'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const { currentTheme } = useTheme()

const isFreshdesk = computed(() => currentTheme.value === 'freshdesk')
const replyExpanded = ref(false)
const initialMessageType = ref('reply')
const scrollContainer = ref(null)
const replyBoxRef = ref(null)

const expandReply = async (type) => {
  initialMessageType.value = type
  replyExpanded.value = true
  await nextTick()
  // Scroll to bottom so the reply editor is visible
  if (scrollContainer.value) {
    scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
  }
}

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
