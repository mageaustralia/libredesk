<template>
  <!--
    Override: when an outer unified scroll container is provided by the
    parent (Conversation override), this component drops its own internal
    overflow-y-auto and lets the parent scroll. All scroll math (isAtBottom,
    scrollToBottom, scrollToMessage) routes through getScrollEl(), which
    returns the injected element if present or the inner threadEl otherwise.
    Falling back keeps the upstream behaviour intact for any caller that
    doesn't provide a unified scroll container.
  -->
  <div class="flex flex-col relative" :class="hasUnifiedScroll ? '' : 'h-full'">
    <div
      ref="threadEl"
      :class="hasUnifiedScroll ? '' : 'flex-1 overflow-y-auto'"
      @scroll="hasUnifiedScroll ? null : handleScroll"
    >
      <div class="min-h-full px-4 pb-10">
        <div
          class="text-center mt-3"
          v-if="
            conversationStore.currentConversationHasMoreMessages &&
            !conversationStore.messages.loading
          "
        >
          <Button
            size="sm"
            variant="outline"
            @click="conversationStore.fetchNextMessages"
            class="transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-700 hover:scale-105 active:scale-95"
          >
            <RefreshCw size="17" class="mr-2" />
            {{ $t('globals.terms.loadMore') }}
          </Button>
        </div>

        <MessagesSkeleton :count="10" v-if="conversationStore.messages.loading" />

        <TransitionGroup v-else enter-active-class="animate-slide-in" tag="div" class="space-y-4">
          <div
            v-for="(message, index) in conversationStore.conversationMessages"
            :key="message.uuid"
            :data-message-uuid="message.uuid"
            :class="{
              'my-2': message.type === 'activity',
              'pt-4': index === 0
            }"
          >
            <div v-if="!message.private && message.type !== 'activity'">
              <MessageBubble :message="message" :direction="message.type" />
            </div>
            <div v-else-if="isPrivateNote(message)">
              <MessageBubble :message="message" direction="outgoing" />
            </div>
            <div v-else-if="message.type === 'activity'">
              <ActivityMessageBubble :message="message" />
            </div>
          </div>
        </TransitionGroup>
      </div>

      <!-- Typing indicator -->
      <div v-if="conversationStore.conversation.isTyping" class="px-4 pb-4">
        <TypingIndicator />
      </div>
    </div>

    <!-- Sticky container for the scroll arrow -->
    <ScrollToBottomButton
      :is-at-bottom="isAtBottom"
      :unread-count="unReadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, inject, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
// OVERRIDE: relative ./ imports here would resolve under overrides/.../message/,
// where these siblings don't exist. Use absolute @/ paths so they fall through
// the override resolver to the upstream files.
import MessageBubble from '@/features/conversation/message/MessageBubble.vue'
import ActivityMessageBubble from '@/features/conversation/message/ActivityMessageBubble.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { RefreshCw, ChevronDown } from 'lucide-vue-next'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import MessagesSkeleton from '@/features/conversation/message/MessagesSkeleton.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'

const route = useRoute()

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const emitter = useEmitter()
const isAtBottom = ref(true)
const unReadMessages = ref(0)
const currentConversationUUID = ref('')

// Injected unified scroll container from the Conversation override.
// When present, scroll calculations target it instead of our inner threadEl.
const unifiedScrollEl = inject('unifiedScrollEl', null)
const hasUnifiedScroll = computed(() => unifiedScrollEl !== null)

const getScrollEl = () => (hasUnifiedScroll.value ? unifiedScrollEl.value : threadEl.value)

const checkIfAtBottom = () => {
  const el = getScrollEl()
  if (el) {
    const tolerance = 100
    const isBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= tolerance
    isAtBottom.value = isBottom
  }
}

const handleScroll = () => {
  checkIfAtBottom()
}

const handleScrollToBottom = () => {
  scrollToBottom()
}

const scrollToBottom = () => {
  setTimeout(() => {
    const el = getScrollEl()
    if (el) {
      el.scrollTop = el.scrollHeight
      checkIfAtBottom()
    }
  }, 50)
}

const scrollToMessage = (messageUUID) => {
  if (!messageUUID) {
    scrollToBottom()
    return
  }

  setTimeout(() => {
    const el = getScrollEl()
    const messageEl = threadEl.value?.querySelector(`[data-message-uuid="${messageUUID}"]`)
    if (messageEl && el) {
      // In unified-scroll mode the message's offsetTop is relative to threadEl,
      // not the scroll container. Compute absolute offset by walking up.
      let offsetTop = messageEl.offsetTop
      if (hasUnifiedScroll.value) {
        let node = messageEl.offsetParent
        while (node && node !== el) {
          offsetTop += node.offsetTop
          node = node.offsetParent
        }
      }
      const visibleHeight = el.clientHeight
      const messageHeight = messageEl.offsetHeight
      // Position message at ~1/3 from top of viewport for better visibility
      const targetScroll = offsetTop - visibleHeight / 3 + messageHeight / 2
      el.scrollTop = Math.max(0, targetScroll)

      // Highlight the message briefly
      messageEl.classList.add('highlight-mention')
      setTimeout(() => messageEl.classList.remove('highlight-mention'), 2500)
    } else {
      // Message not found, scroll to bottom instead
      scrollToBottom()
    }
  }, 150)
}

onMounted(() => {
  checkIfAtBottom()
  handleNewMessage()
  // In unified-scroll mode the scroll listener lives on the parent container,
  // not threadEl. Hook into it so isAtBottom stays accurate.
  if (hasUnifiedScroll.value && unifiedScrollEl.value) {
    unifiedScrollEl.value.addEventListener('scroll', handleScroll, { passive: true })
  }
})

onUnmounted(() => {
  if (hasUnifiedScroll.value && unifiedScrollEl.value) {
    unifiedScrollEl.value.removeEventListener('scroll', handleScroll)
  }
})

const newMessageHandler = (data) => {
  if (data.conversation_uuid === conversationStore.current.uuid) {
    // Agent's own message - always scroll to bottom
    if (data.message?.sender_id === userStore.userID) {
      scrollToBottom()
    }
    // Customer message - only scroll if already at bottom
    else if (isAtBottom.value) {
      scrollToBottom()
    }
    // Customer message but not at bottom - don't scroll, increment unread
    else {
      unReadMessages.value++
    }
  }
}

const handleNewMessage = () => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
}

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.NEW_MESSAGE, newMessageHandler)
})

watch(
  () => conversationStore.conversationMessages,
  (messages) => {
    // Scroll to bottom when conversation changes and there are new messages.
    // New messages on next db page should not scroll to bottom.
    if (
      messages.length > 0 &&
      conversationStore?.current?.uuid &&
      currentConversationUUID.value !== conversationStore.current.uuid
    ) {
      currentConversationUUID.value = conversationStore.current.uuid
      unReadMessages.value = 0

      // Check if this is a mentioned conversation
      const scrollToUUID = route.query.scrollTo
      if (scrollToUUID) {
        // Mentioned conversation - only scroll to message, NOT to bottom
        scrollToMessage(scrollToUUID)
      } else {
        // Normal conversation - scroll to bottom
        scrollToBottom()
      }
    }
  }
)

// Watch for typing indicator and auto-scroll if user is at bottom
watch(
  () => conversationStore.conversation.isTyping,
  (isTyping) => {
    if (isTyping && isAtBottom.value) {
      scrollToBottom()
    }
  }
)

// Watch for isAtButtom and set unReadMessages to 0
watch(
  () => isAtBottom.value,
  (atBottom) => {
    if (atBottom) {
      unReadMessages.value = 0
    }
  }
)

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}
</script>

<style scoped>
.highlight-mention {
  animation: highlightPulse 2.5s ease-out;
}

@keyframes highlightPulse {
  0% {
    background-color: rgb(251 191 36 / 0.35);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}

/* Dark mode highlight - softer yellow */
:global(.dark) .highlight-mention {
  animation: highlightPulseDark 2.5s ease-out;
}

@keyframes highlightPulseDark {
  0% {
    background-color: rgb(250 204 21 / 0.2);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}
</style>
