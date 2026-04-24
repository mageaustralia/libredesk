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
import { useConversationStore } from '../../stores/conversation'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import { useEmitter } from '../../composables/useEmitter'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { MoreHorizontal, Trash2, RotateCcw, ShieldAlert, ShieldCheck } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const router = useRouter()
const { t } = useI18n()

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
