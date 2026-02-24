<template>
  <div class="flex flex-col h-full">
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

        <!-- More Actions Dropdown -->
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" class="h-7 w-7">
              <MoreHorizontal class="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="conversationStore.conversation.data?.status !== 'Trashed'" 
              @click="handleMoveToTrash"
            >
              <Trash2 class="w-4 h-4 mr-2" />
              Move to Trash
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="conversationStore.conversation.data?.status === 'Trashed'" 
              @click="handleRestore"
            >
              <RotateCcw class="w-4 h-4 mr-2" />
              Restore
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              v-if="conversationStore.conversation.data?.status !== 'Spam'" 
              @click="handleMarkAsSpam"
            >
              <ShieldAlert class="w-4 h-4 mr-2" />
              Mark as Spam
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="conversationStore.conversation.data?.status === 'Spam'" 
              @click="handleMarkAsNotSpam"
            >
              <ShieldCheck class="w-4 h-4 mr-2" />
              Not Spam
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Fresh theme: unified scroll with collapsible reply -->
    <template v-if="isFresh">
      <!-- Scrollable area: messages + expanded reply -->
      <div class="flex-1 overflow-y-auto fresh-unified-scroll" ref="scrollContainer">
        <MessageList />
        <!-- Expanded reply box flows inline with messages -->
        <div v-if="replyExpanded" class="border-t">
          <ReplyBox />
        </div>
      </div>

      <!-- Collapsed reply bar: fixed at bottom, outside scroll -->
      <div
        v-if="!replyExpanded"
        class="flex-shrink-0 bg-background border-t px-4 py-2.5 mb-4 flex gap-2"
      >
        <Button size="sm" variant="outline" @click="expandReply">
          <Reply class="h-4 w-4 mr-1.5" />
          Reply
        </Button>
        <Button size="sm" variant="outline" @click="expandReply">
          <StickyNote class="h-4 w-4 mr-1.5" />
          Private note
        </Button>
      </div>
    </template>

    <!-- Default theme: original layout with sticky reply -->
    <template v-else>
      <div class="flex flex-col flex-grow overflow-hidden">
        <MessageList class="flex-1 overflow-y-auto" />
        <div class="sticky bottom-0">
          <ReplyBox />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useTheme } from '@/composables/useTheme'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { Reply, StickyNote, MoreHorizontal, Trash2, RotateCcw, ShieldAlert, ShieldCheck } from 'lucide-vue-next'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { Skeleton } from '@/components/ui/skeleton'
import { useRouter } from 'vue-router'
import api from '@/api'
import { handleHTTPError } from '@/utils/http'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const router = useRouter()
const { currentTheme } = useTheme()

const isFresh = computed(() => currentTheme.value === 'fresh')
const replyExpanded = ref(false)
const scrollContainer = ref(null)

const expandReply = async () => {
  replyExpanded.value = true
  await nextTick()
  // Scroll to bottom so the full reply editor (including action icons) is visible
  if (scrollContainer.value) {
    // Use setTimeout to allow ReplyBox to fully render before scrolling
    setTimeout(() => {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }, 100)
  }
}


const handleMoveToTrash = async () => {
  try {
    await api.moveToTrash(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Moved to trash' })
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleRestore = async () => {
  try {
    await api.restoreFromTrash(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Conversation restored' })
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleMarkAsSpam = async () => {
  try {
    await api.markAsSpam(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Marked as spam' })
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleMarkAsNotSpam = async () => {
  try {
    await api.markAsNotSpam(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Moved to inbox' })
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
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
