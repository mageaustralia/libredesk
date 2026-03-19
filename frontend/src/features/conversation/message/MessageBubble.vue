<template>
  <div class="flex flex-col text-left" :class="isOutgoing ? 'items-end' : 'items-start'">
    <!-- Sender Name + Timestamp -->
    <div class="mb-1 flex items-baseline gap-2" :class="isOutgoing ? 'pr-[47px] justify-end' : 'pl-[47px]'">
      <router-link
        v-if="!isOutgoing"
        :to="{ name: 'contact-detail', params: { id: message.author?.id } }"
        class="text-muted-foreground text-sm font-medium hover:underline hover:text-primary"
      >
        {{ getFullName }}
      </router-link>
      <p v-else class="text-muted-foreground text-sm font-medium">
        {{ getFullName }}
      </p>
      <Tooltip>
        <TooltipTrigger>
          <span class="text-muted-foreground/60 text-xs">
            {{ formatMessageTimestamp(message.created_at) }}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{{ formatFullTimestamp(message.created_at) }}</p>
        </TooltipContent>
      </Tooltip>
    </div>

    <!-- Message Bubble -->
    <div class="flex flex-row gap-2 w-full" :class="{ 'justify-end': isOutgoing }">
      <!-- Avatar (left for incoming) -->
      <router-link
        v-if="!isOutgoing"
        :to="{ name: 'contact-detail', params: { id: message.author?.id } }"
        class="flex-shrink-0"
      >
        <Avatar class="cursor-pointer w-8 h-8 hover:opacity-80 transition-opacity">
          <AvatarImage :src="getAvatar" />
          <AvatarFallback class="font-medium">
            {{ avatarFallback }}
          </AvatarFallback>
        </Avatar>
      </router-link>

      <!-- Bubble Wrapper with max 80% width -->
      <div
        class="w-4/5"
        :class="{ 'flex justify-end': isOutgoing }"
        style="contain: inline-size"
      >
        <div
          class="flex flex-col justify-end message-bubble"
          :class="bubbleClasses"
        >
          <!-- PCI Data Warning -->
          <div v-if="message.has_pci_data" class="flex items-center gap-2 mb-2 px-3 py-2 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md">
            <ShieldAlert class="w-4 h-4 text-red-500 shrink-0" />
            <span class="text-xs text-red-700 dark:text-red-300 font-medium flex-1">This message contains credit card data</span>
            <button
              v-if="!redacting"
              class="text-xs font-medium text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-200 underline"
              @click="redactPCI"
            >
              Redact Now
            </button>
            <Spinner v-if="redacting" size="w-3 h-3" />
          </div>

          <!-- Message Envelope -->
          <MessageEnvelope :message="message" v-if="showEnvelope" />

          <hr class="mb-2" v-if="showEnvelope" />

          <!-- Message Content -->
          <div
            v-if="message.content_type === 'text'"
            class="mb-1 native-html whitespace-pre-wrap"
            :class="{ 'mb-3': message.attachments.length > 0 }"
          >
            {{ sanitizedContent }}
          </div>
          <Letter
            v-else
            :html="sanitizedContent"
            :rewriteExternalResources="rewriteResource"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="mb-1 native-html break-words"
            :class="{ 'mb-3': message.attachments.length > 0 }"
          />

          <!-- Quoted Text Toggle (incoming only) -->
          <div
            v-if="hasQuotedContent"
            @click="toggleQuote"
            class="text-xs cursor-pointer text-muted-foreground px-2 py-1 w-max hover:bg-muted hover:text-primary rounded transition-all"
          >
            {{ showQuotedText ? t('conversation.hideQuotedText') : t('conversation.showQuotedText') }}
          </div>

          <!-- Attachments -->
          <MessageAttachmentPreview :attachments="nonInlineAttachments" />

          <!-- Spinner for Pending Messages (outgoing only) -->
          <Spinner v-if="isOutgoing && message.status === 'pending'" size="w-4 h-4" />

          <!-- Status Icons (outgoing only) -->
          <div v-if="isOutgoing" class="flex items-center space-x-2 mt-2 self-end">
            <Lock :size="10" v-if="isPrivateMessage" class="text-muted-foreground" />
            <Check :size="14" v-if="showCheckCheck" class="text-green-500" />
            <RotateCcw
              size="10"
              @click="retryMessage(message)"
              class="cursor-pointer text-muted-foreground hover:text-foreground transition-colors duration-200"
              v-if="showRetry"
            />
          </div>
        </div>
      </div>

      <!-- Avatar (right for outgoing) -->
      <Avatar v-if="isOutgoing" class="cursor-pointer w-8 h-8">
        <AvatarImage :src="getAvatar" />
        <AvatarFallback class="font-medium">
          {{ avatarFallback }}
        </AvatarFallback>
      </Avatar>
    </div>

    <!-- Forward button -->
    <div
      v-if="!isActivity && !message.private"
      class="flex items-center gap-1 mt-1"
      :class="isOutgoing ? 'pr-[47px] justify-end' : 'pl-[47px]'"
    >
      <button
        @click="forwardMessage"
        class="text-xs text-muted-foreground/50 hover:text-muted-foreground transition-colors flex items-center gap-1"
      >
        <Forward class="w-3 h-3" />
        <span>Forward</span>
      </button>
    </div>

    <!-- Forwarded badge -->
    <div
      v-if="forwardedTo"
      class="flex items-center gap-1 mt-1"
      :class="isOutgoing ? 'pr-[47px] justify-end' : 'pl-[47px]'"
    >
      <span class="text-xs text-muted-foreground/70 italic flex items-center gap-1">
        <Forward class="w-3 h-3" />
        Forwarded to: {{ forwardedTo }}
      </span>
    </div>

  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useI18n } from 'vue-i18n'
import { Lock, RotateCcw, Check, ShieldAlert, Forward } from 'lucide-vue-next'
import { revertCIDToImageSrc } from '@/utils/strings'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Spinner } from '@/components/ui/spinner'
import { formatMessageTimestamp, formatFullTimestamp } from '@/utils/datetime'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Letter } from 'vue-letter'
import MessageAttachmentPreview from '@/features/conversation/message/attachment/MessageAttachmentPreview.vue'
import MessageEnvelope from './MessageEnvelope.vue'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'

const props = defineProps({
  message: Object,
  direction: {
    type: String,
    validator: (v) => ['incoming', 'outgoing'].includes(v)
  }
})

const emitter = useEmitter()

const convStore = useConversationStore()
const settingsStore = useAppSettingsStore()
const { t } = useI18n()

// Activity check
const isActivity = computed(() => props.message.type === 'activity')

// Direction helpers
const isOutgoing = computed(() => props.direction === 'outgoing')

// Author info from message
const getFullName = computed(() => {
  const author = props.message.author ?? {}
  const firstName = author.first_name ?? 'User'
  const lastName = author.last_name ?? ''
  return `${firstName} ${lastName}`.trim()
})

const getAvatar = computed(() => {
  return props.message.author?.avatar_url || ''
})

const avatarFallback = computed(() => {
  const firstName = props.message.author?.first_name ?? (isOutgoing.value ? 'A' : 'U')
  return firstName.toUpperCase().substring(0, 2)
})

// Content sanitization - different processing for incoming vs outgoing
const sanitizedContent = computed(() => {
  let content = props.message.content || ''

  if (isOutgoing.value) {
    return revertCIDToImageSrc(content)
  } else {
    const baseUrl = settingsStore.settings['app.root_url']
    content = props.message.attachments.reduce(
      (acc, { content_id, url }) => acc.replace(new RegExp(`cid:${content_id}`, 'g'), url),
      content
    )
    content = content.replace(/src="\/uploads\//g, `src="${baseUrl}/uploads/`)
    // Strip runs of 3+ consecutive empty paragraphs down to one (preserve intentional spacing)
    content = content.replace(/(<p[^>]*>\s*(&nbsp;|\u00a0|\s|<br\s*\/?>)*\s*<\/p>\s*){3,}/gi, '<p>&nbsp;</p>')
    return content
  }
})

const rewriteResource = (url) => {
  const baseUrl = settingsStore.settings['app.root_url'] || ''
  // Rewrite relative /uploads/ paths to absolute
  if (url.startsWith('/uploads/')) {
    return baseUrl + url
  }
  return url
}

const nonInlineAttachments = computed(() =>
  props.message.attachments.filter((attachment) => attachment.disposition !== 'inline')
)

// Bubble classes - conditional based on direction
const redacting = ref(false)

const redactPCI = async () => {
  if (!confirm('This will permanently redact credit card data from this message and attempt to delete the original email. This cannot be undone.')) return
  redacting.value = true
  try {
    const cuuid = convStore.current?.uuid
    await api.redactMessagePCI(cuuid, props.message.uuid)
    window.location.reload()
  } catch (err) {
    redacting.value = false
    alert('Failed to redact: ' + (err?.response?.data?.message || err.message))
  }
}

const bubbleClasses = computed(() => ({
  // Outgoing-specific: private message styling
  'bg-private': isOutgoing.value && props.message.private,
  'border border-border': isOutgoing.value && !props.message.private,
  'bg-agent-bubble': isOutgoing.value && !props.message.private,
  'bg-customer-bubble': !isOutgoing.value,
  'opacity-50 animate-pulse': isOutgoing.value && props.message.status === 'pending',
  'border-red-400': isOutgoing.value && props.message.status === 'failed',
  relative: isOutgoing.value,
  // Incoming-specific: quoted text visibility
  'show-quoted-text': showQuotedText.value,
  'hide-quoted-text': hasQuotedContent.value && !showQuotedText.value
}))

// Outgoing-only computed properties
const isPrivateMessage = computed(() => isOutgoing.value && props.message.private)
const showCheckCheck = computed(
  () => isOutgoing.value && props.message.status === 'sent' && !isPrivateMessage.value
)
const showRetry = computed(() => isOutgoing.value && props.message.status === 'failed')

const retryMessage = (msg) => {
  api.retryMessage(convStore.current.uuid, msg.uuid)
}

// Incoming-only: quoted text toggle
const showQuotedText = ref(false)
const hasQuotedContent = computed(
  () => sanitizedContent.value.includes('<blockquote') || sanitizedContent.value.includes('gmail_quote')
)
const toggleQuote = () => {
  showQuotedText.value = !showQuotedText.value
}

// Forward functionality
const forwardedTo = computed(() => {
  if (!props.message.meta?.forwarded) return null
  const fwdTo = props.message.meta?.forwarded_to
  if (Array.isArray(fwdTo)) return fwdTo.join(', ')
  return null
})

const forwardMessage = () => {
  emitter.emit('forward-message', props.message)
}

// Envelope visibility (both directions)
const showEnvelope = computed(() => {
  return (
    props.message.meta?.from?.length ||
    props.message.meta?.to?.length ||
    props.message.meta?.cc?.length ||
    props.message.meta?.bcc?.length ||
    props.message.meta?.subject
  )
})
</script>