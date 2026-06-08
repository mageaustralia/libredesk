<template>
  <div
    v-if="
      conversationStore.current?.previous_conversations?.length === 0 ||
      conversationStore.conversation?.loading
    "
    class="text-center text-sm text-muted-foreground py-4"
  >
    {{ $t('conversation.sidebar.noPreviousConvo') }}
  </div>
  <div v-else class="space-y-1">
    <router-link
      v-for="conversation in conversationStore.current?.previous_conversations || []"
      :key="conversation.uuid"
      :to="{
        name: 'inbox-conversation',
        params: {
          uuid: conversation.uuid,
          type: 'assigned'
        }
      }"
      class="block p-2 rounded hover:bg-muted"
    >
      <div class="flex flex-wrap items-start justify-between gap-1">
        <div class="flex flex-col flex-1 min-w-[120px]">
          <Tooltip>
            <TooltipTrigger asChild>
              <span class="sidebar-value font-medium truncate block">
                <span
                  v-if="conversation.reference_number"
                  class="text-muted-foreground font-mono mr-1"
                >#{{ conversation.reference_number }}</span>
                {{ conversation.subject || '(no subject)' }}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span v-if="conversation.reference_number">#{{ conversation.reference_number }} — </span>{{ conversation.subject || '(no subject)' }}
            </TooltipContent>
          </Tooltip>
          <span class="sidebar-label truncate block">
            {{ conversation.last_message }}
          </span>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <!-- Single relative time = last activity (last message, falling
                 back to created). Showing created • last-message read as a
                 pointless duplicate ("23h • 23h") whenever both landed in
                 the same coarse bucket. Both exact times stay in the tooltip. -->
            <div class="sidebar-label flex-shrink-0">
              <span v-if="conversation.last_message_at || conversation.created_at">
                {{ getRelativeTime(new Date(conversation.last_message_at || conversation.created_at)) }}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <div class="space-y-1 text-xs">
              <p>
                {{ $t('globals.terms.createdAt') }}:
                {{ formatFullTimestamp(new Date(conversation.created_at)) }}
              </p>
              <p v-if="conversation.last_message_at">
                {{ $t('globals.terms.lastMessageAt') }}:
                {{ formatFullTimestamp(new Date(conversation.last_message_at)) }}
              </p>
            </div>
          </TooltipContent>
        </Tooltip>
      </div>
    </router-link>
  </div>
</template>

<script setup>
import { useConversationStore } from '@/stores/conversation'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { formatFullTimestamp, getRelativeTime } from '@shared-ui/utils/datetime.js'

const conversationStore = useConversationStore()
</script>
