<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span v-if="!conversationStore.conversation.loading">
          {{ conversationStore.currentContactName }}
        </span>
        <Skeleton class="w-[130px] h-6" v-else />
        <!-- Presence: other agents viewing -->
        <div v-if="otherViewers.length > 0" class="flex items-center gap-1 ml-2">
          <Eye class="w-3.5 h-3.5 text-blue-500 animate-blink" />
          <TooltipProvider :delay-duration="200">
            <div class="flex -space-x-1.5">
              <Tooltip v-for="viewer in otherViewers.slice(0, 3)" :key="viewer.user_id">
                <TooltipTrigger asChild>
                  <Avatar class="w-5 h-5 rounded-full border border-background cursor-default">
                    <AvatarImage :src="viewer.avatar_url || ''" v-if="viewer.avatar_url" />
                    <AvatarFallback class="text-[8px]">{{ (viewer.first_name || '?').substring(0, 1) }}</AvatarFallback>
                  </Avatar>
                </TooltipTrigger>
                <TooltipContent side="bottom" class="text-xs">
                  {{ viewer.first_name || 'Unknown' }}
                </TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>
          <span v-if="otherViewers.length > 3" class="text-[10px] text-muted-foreground">+{{ otherViewers.length - 3 }}</span>
        </div>
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
              <ChevronDown class="w-3 h-3 text-secondary" />
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
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="toggleFollow">
              <EyeOff v-if="isFollowing" class="w-4 h-4 mr-2" />
              <Eye v-else class="w-4 h-4 mr-2" />
              {{ isFollowing ? 'Unfollow' : 'Follow' }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Merge banner -->
    <div
      v-if="conversationStore.current?.merged_into_id"
      class="flex items-center gap-2 px-4 py-2 bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300 text-sm border-b"
    >
      <GitMerge class="w-4 h-4 shrink-0" />
      <span>
        This ticket was merged into
        <router-link
          v-if="conversationStore.current?.merged_into_uuid"
          :to="{ name: 'inbox-conversation', params: { uuid: conversationStore.current.merged_into_uuid } }"
          class="font-medium underline"
        >#{{ conversationStore.current.merged_into_ref }}</router-link>
      </span>
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
import { ref, computed, nextTick, watch, onMounted, onBeforeUnmount } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { usePresenceStore } from '@/stores/presence'
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
import { Reply, StickyNote, MoreHorizontal, Trash2, RotateCcw, ShieldAlert, ShieldCheck, ChevronDown, GitMerge, Eye, EyeOff } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { sendMessage as wsSendMessage } from '@/websocket'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { Skeleton } from '@/components/ui/skeleton'
import { useRouter } from 'vue-router'
import api from '@/api'
import { handleHTTPError } from '@/utils/http'

import { useUserStore } from '@/stores/user'

const conversationStore = useConversationStore()
const presenceStore = usePresenceStore()
const userStore = useUserStore()
const emitter = useEmitter()
const router = useRouter()
const { currentTheme } = useTheme()

// Follow/unfollow state
const isFollowing = ref(false)

const checkFollowStatus = async () => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  try {
    const res = await api.getConversationParticipants(uuid)
    const participants = res.data?.data || []
    isFollowing.value = participants.some(p => p.id === userStore.userID)
  } catch { /* ignore */ }
}

const toggleFollow = async () => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  try {
    if (isFollowing.value) {
      await api.unfollowConversation(uuid)
      isFollowing.value = false
    } else {
      await api.followConversation(uuid)
      isFollowing.value = true
    }
  } catch (err) {
    handleHTTPError(err)
  }
}

// Presence tracking
const currentViewingUUID = ref('')

const otherViewers = computed(() => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return []
  return presenceStore.getViewers(uuid, userStore.userID)
})

function sendViewingPresence(uuid) {
  if (currentViewingUUID.value === uuid) return
  currentViewingUUID.value = uuid
  wsSendMessage({ type: 'view_conversation', data: { conversation_uuid: uuid || '' } })
}

// Watch for conversation changes and send presence
watch(
  () => conversationStore.current?.uuid,
  (newUUID) => {
    if (newUUID) {
      sendViewingPresence(newUUID)
      checkFollowStatus()
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  // Clear presence when leaving
  sendViewingPresence('')
})

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
    await conversationStore.fetchFirstPageConversations()
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleRestore = async () => {
  try {
    await api.restoreFromTrash(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Conversation restored' })
    await conversationStore.fetchFirstPageConversations()
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleMarkAsSpam = async () => {
  try {
    await api.markAsSpam(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Marked as spam' })
    await conversationStore.fetchFirstPageConversations()
    router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

const handleMarkAsNotSpam = async () => {
  try {
    await api.markAsNotSpam(conversationStore.conversation.data.uuid)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Moved to inbox' })
    await conversationStore.fetchFirstPageConversations()
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

<style scoped>
@keyframes blink {
  0%, 90%, 100% { transform: scaleY(1); }
  95% { transform: scaleY(0.1); }
}
.animate-blink {
  animation: blink 3s ease-in-out infinite;
  transform-origin: center;
}
</style>
