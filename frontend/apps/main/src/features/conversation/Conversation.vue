<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span v-if="!conversationStore.conversation.loading">
          {{ conversationStore.currentContactName }}
        </span>
        <Skeleton class="w-[130px] h-6" v-else />
        <!-- Presence: other agents currently viewing this conversation -->
        <div v-if="otherViewers.length > 0" class="flex items-center gap-1 ml-2">
          <Eye class="w-3.5 h-3.5 text-blue-500 animate-blink" />
          <TooltipProvider :delay-duration="200">
            <div class="flex -space-x-1.5">
              <Tooltip v-for="viewer in otherViewers.slice(0, 3)" :key="viewer.user_id">
                <TooltipTrigger asChild>
                  <span
                    class="inline-flex items-center justify-center w-5 h-5 rounded-full border border-background bg-muted text-[8px] font-medium cursor-default"
                    :title="viewer.first_name || 'Agent'"
                  >
                    {{ (viewer.first_name || '?').substring(0, 1) }}
                  </span>
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
      <div class="flex items-center">
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

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <button
              v-if="!conversationStore.conversation.loading"
              class="flex items-center justify-center cursor-pointer hover:bg-muted rounded-md h-7 w-7 ml-2"
              :title="t('globals.terms.actions', 2)"
            >
              <MoreHorizontal class="w-4 h-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              v-if="conversationStore.current?.status !== 'Trashed'"
              @click="handleMoveToTrash"
            >
              <Trash2 class="w-4 h-4 mr-2" />
              {{ t('conversation.moveToTrash') }}
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="conversationStore.current?.status === 'Trashed'"
              @click="handleRestore"
            >
              <RotateCcw class="w-4 h-4 mr-2" />
              {{ t('conversation.restoreFromTrash') }}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              v-if="conversationStore.current?.status !== 'Spam'"
              @click="handleMarkAsSpam"
            >
              <ShieldAlert class="w-4 h-4 mr-2" />
              {{ t('conversation.markAsSpam') }}
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="conversationStore.current?.status === 'Spam'"
              @click="handleMarkAsNotSpam"
            >
              <ShieldCheck class="w-4 h-4 mr-2" />
              {{ t('conversation.markAsNotSpam') }}
            </DropdownMenuItem>
            <DropdownMenuSeparator
              v-if="!conversationStore.current?.merged_into_id"
            />
            <DropdownMenuItem
              v-if="!conversationStore.current?.merged_into_id"
              @click="showMergeDialog = true"
            >
              <GitMerge class="w-4 h-4 mr-2" />
              {{ t('conversation.merge.action') }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Merge banner: shown on a secondary that has been merged into a primary. -->
    <div
      v-if="conversationStore.current?.merged_into_id"
      class="flex items-center gap-2 px-4 py-2 bg-blue-50 dark:bg-blue-950/40 text-blue-800 dark:text-blue-200 text-sm border-b border-blue-200 dark:border-blue-800"
    >
      <GitMerge class="w-4 h-4 shrink-0" />
      <span>
        {{ t('conversation.merge.mergedIntoBanner') }}
        <router-link
          v-if="conversationStore.current?.merged_into_uuid"
          :to="mergedIntoLink"
          class="font-semibold underline text-blue-600 dark:text-blue-300 hover:text-blue-900 dark:hover:text-blue-100"
        >#{{ conversationStore.current.merged_into_ref }}</router-link>
      </span>
    </div>

    <!-- Merge dialog -->
    <MergeDialog
      v-model:open="showMergeDialog"
      :initial-conversation="conversationStore.conversation.data"
      @merged="handleMerged"
    />

    <!-- Messages & reply box -->
    <div class="flex flex-col flex-grow overflow-hidden">
      <MessageList class="flex-1 overflow-y-auto" />
      <ReplyBox />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useConversationStore } from '../../stores/conversation'
import { usePresenceStore } from '../../stores/presence'
import { useUserStore } from '../../stores/user'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import MergeDialog from './MergeDialog.vue'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import { useEmitter } from '../../composables/useEmitter'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { MoreHorizontal, Trash2, RotateCcw, ShieldAlert, ShieldCheck, Eye, GitMerge } from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { sendViewConversation } from '@/websocket'

const conversationStore = useConversationStore()
const presenceStore = usePresenceStore()
const userStore = useUserStore()
const emitter = useEmitter()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const showMergeDialog = ref(false)

// Build a link to the primary the current secondary was merged into. Reuse the
// current route's `type` (assigned/unassigned/etc) so the navigation lands the
// agent in the same inbox view.
const mergedIntoLink = computed(() => ({
  name: 'inbox-conversation',
  params: {
    uuid: conversationStore.current?.merged_into_uuid,
    type: route.params.type || 'assigned'
  }
}))

function handleMerged ({ primary_uuid }) {
  // Navigate to the primary so the agent immediately sees the unified thread.
  if (!primary_uuid) return
  router.push({
    name: 'inbox-conversation',
    params: { uuid: primary_uuid, type: route.params.type || 'assigned' }
  })
}

// Presence tracking
const otherViewers = computed(() => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return []
  return presenceStore.getViewers(uuid, userStore.userID)
})

// Send presence when conversation changes
watch(
  () => conversationStore.current?.uuid,
  (newUUID, oldUUID) => {
    if (newUUID) {
      sendViewConversation(newUUID)
    } else if (oldUUID) {
      sendViewConversation('')
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  sendViewConversation('')
})

const showToast = (description, variant) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, variant ? { variant, description } : { description })
}

const runConversationAction = async (action, successMsg) => {
  const uuid = conversationStore.conversation.data?.uuid
  if (!uuid) return
  try {
    await action(uuid)
    showToast(successMsg)
    // Step back so the user lands on whatever list they came from (e.g. Trash
    // when restoring from trash, custom view when marking as spam) instead of
    // forcing them back to the assigned inbox.
    if (window.history.length > 1) router.back()
    else router.push({ name: 'inbox', params: { type: 'assigned' } })
  } catch (error) {
    showToast(handleHTTPError(error).message, 'destructive')
  }
}

const handleMoveToTrash = () => runConversationAction(api.moveToTrash, t('conversation.moveToTrash'))
const handleRestore = () => runConversationAction(api.restoreFromTrash, t('conversation.restoreFromTrash'))
const handleMarkAsSpam = () => runConversationAction(api.markAsSpam, t('conversation.markAsSpam'))
const handleMarkAsNotSpam = () => runConversationAction(api.markAsNotSpam, t('conversation.markAsNotSpam'))

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
