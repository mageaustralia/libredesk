<template>
  <Dialog :open="openAIKeyPrompt" @update:open="openAIKeyPrompt = false">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader class="space-y-2">
        <DialogTitle>{{ $t('ai.enterOpenAIAPIKey') }}</DialogTitle>
        <DialogDescription>
          {{
            $t('ai.apiKey.description', {
              provider: 'OpenAI'
            })
          }}
        </DialogDescription>
      </DialogHeader>
      <Form v-slot="{ handleSubmit }" as="" keep-values :validation-schema="formSchema">
        <form id="apiKeyForm" @submit="handleSubmit($event, updateProvider)">
          <FormField v-slot="{ componentField }" name="apiKey">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.apiKey') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="sk-am1RLw7XUWGX.." v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </form>
        <DialogFooter>
          <Button
            type="submit"
            form="apiKeyForm"
            :is-loading="isOpenAIKeyUpdating"
            :disabled="isOpenAIKeyUpdating"
          >
            {{ $t('globals.messages.save') }}
          </Button>
        </DialogFooter>
      </Form>
    </DialogContent>
  </Dialog>

  <!-- Collision confirmation dialog -->
  <AlertDialog :open="showCollisionConfirm" @update:open="showCollisionConfirm = false">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Another agent has replied</AlertDialogTitle>
        <AlertDialogDescription>
          {{ collisionAgentName }} sent a reply while you were composing. Review their message before sending to avoid a duplicate response.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction @click="confirmSend">Send anyway</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <!-- Discard draft confirmation -->
  <AlertDialog :open="showDiscardDraft" @update:open="showDiscardDraft = false">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Discard draft</AlertDialogTitle>
        <AlertDialogDescription>
          Your draft will be discarded. Are you sure?
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>No</AlertDialogCancel>
        <AlertDialogAction @click="confirmDeleteDraft">Yes</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <div class="text-foreground bg-background">
    <!-- Collision warning banner -->
    <div
      v-if="collisionWarning"
      class="flex items-center gap-2 px-3 py-2 mx-2 mt-2 rounded-md text-sm border" style="background-color: #fefce8; border-color: #fde047; color: #854d0e;"
    >
      <AlertTriangle class="w-4 h-4 shrink-0" />
      <span class="flex-1">{{ collisionAgentName }} just sent a reply. Check before sending yours.</span>
      <button @click="dismissCollisionWarning" style="color: #a16207;" class="hover:opacity-70">
        <X class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Customer reply warning banner -->
    <div
      v-if="customerReplyWarning"
      class="flex items-center gap-2 px-3 py-2 mx-2 mt-2 rounded-md text-sm border" style="background-color: #eff6ff; border-color: #93c5fd; color: #1e40af;"
    >
      <AlertTriangle class="w-4 h-4 shrink-0" />
      <span class="flex-1">The customer sent a new message while you were composing. Scroll down to review before sending.</span>
      <button @click="dismissCustomerReplyWarning" style="color: #1e40af;" class="hover:opacity-70">
        <X class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[92%] w-[92%] max-h-[90%] h-[88%] bg-card text-card-foreground p-4 flex flex-col"
        :class="{ '!bg-private': messageType === 'private_note' }"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          :isFullscreen="true"
          :aiPrompts="aiPrompts"
          :isSending="isSending"
          :isDraftLoading="isDraftLoading"
          :isGenerating="isGenerating"
          :uploadingFiles="uploadingFiles"
          :uploadedFiles="mediaFiles"
          :ecommerceConfigured="ecommerceConfigured"
          :inboxes="inboxes"
          :selectedInboxId="selectedInboxId"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          v-model:to="to"
          v-model:cc="cc"
          v-model:bcc="bcc"
          v-model:emailErrors="emailErrors"
          v-model:messageType="messageType"
          v-model:showBcc="showBcc"
          v-model:mentions="mentions"
          @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
          @send="processSend"
          @fileUpload="handleFileUpload"
          @fileDelete="handleFileDelete"
          @filesDropped="(files) => uploadFiles(files)"
          @aiPromptSelected="handleAiPromptSelected"
          @generateResponse="handleGenerateResponse"
          @generateWithOrders="handleGenerateWithOrders"
          @inboxChange="handleInboxChange"
          @sendWithStatus="processSendWithStatus"
          @deleteDraft="handleDeleteDraft"
          :hasDraft="hasDraftContent"
          :sendStatuses="availableSendStatuses"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-background text-card-foreground box m-2 px-2 pt-2 flex flex-col"
      :class="{ '!bg-private': messageType === 'private_note' }"
      v-if="!isEditorFullscreen"
    >
      <ReplyBoxContent
        ref="replyBoxContentRef"
        :isFullscreen="false"
        :aiPrompts="aiPrompts"
        :isSending="isSending"
        :isDraftLoading="isDraftLoading"
        :isGenerating="isGenerating"
        :uploadingFiles="uploadingFiles"
        :uploadedFiles="mediaFiles"
        :ecommerceConfigured="ecommerceConfigured"
        :inboxes="inboxes"
        :selectedInboxId="selectedInboxId"
        v-model:htmlContent="htmlContent"
        v-model:textContent="textContent"
        v-model:to="to"
        v-model:cc="cc"
        v-model:bcc="bcc"
        v-model:emailErrors="emailErrors"
        v-model:messageType="messageType"
        v-model:showBcc="showBcc"
        v-model:mentions="mentions"
        @toggleFullscreen="isEditorFullscreen = !isEditorFullscreen"
        @send="processSend"
        @fileUpload="handleFileUpload"
        @fileDelete="handleFileDelete"
        @filesDropped="(files) => uploadFiles(files)"
        @aiPromptSelected="handleAiPromptSelected"
        @generateResponse="handleGenerateResponse"
        @generateWithOrders="handleGenerateWithOrders"
        @inboxChange="handleInboxChange"
        @sendWithStatus="processSendWithStatus"
        @deleteDraft="handleDeleteDraft"
        :hasDraft="hasDraftContent"
        :sendStatuses="availableSendStatuses"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, toRaw, onMounted, onBeforeUnmount } from 'vue'
import { useStorage } from '@vueuse/core'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@/constants/conversation'
import { useUserStore } from '@/stores/user'
import { useDraftManager } from '@/composables/useDraftManager'
import api from '@/api'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@/stores/conversation'
import { useTeamStore } from '@/stores/team'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useEmitter } from '@/composables/useEmitter'
import { useFileUpload } from '@/composables/useFileUpload'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'
import { UserTypeAgent } from '@/constants/user'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { AlertTriangle, X } from 'lucide-vue-next'
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@/components/ui/form'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'

const formSchema = toTypedSchema(
  z.object({
    apiKey: z.string().min(1, 'API key is required')
  })
)

const { t } = useI18n()
const conversationStore = useConversationStore()
const teamStore = useTeamStore()
const emitter = useEmitter()
const userStore = useUserStore()

// Setup file upload composable
const {
  uploadingFiles,
  handleFileUpload,
  handleFileDelete,
  uploadFiles,
  mediaFiles,
  clearMediaFiles,
  setMediaFiles
} = useFileUpload({
  linkedModel: 'messages'
})

// Setup draft management composable
const currentDraftKey = computed(() => conversationStore.current?.uuid || null)
const {
  htmlContent,
  textContent,
  isLoading: isDraftLoading,
  clearDraft,
  loadedAttachments,
  loadedMacroActions
} = useDraftManager(currentDraftKey, mediaFiles)

// Rest of existing state
const openAIKeyPrompt = ref(false)
const isOpenAIKeyUpdating = ref(false)
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const isGenerating = ref(false)
const messageType = useStorage('replyBoxMessageType', 'reply')
const to = ref('')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])
const aiPrompts = ref([])
const replyBoxContentRef = ref(null)
const mentions = ref([])
const ecommerceConfigured = ref(false)

// Collision detection state
const composingStartedAt = ref(null)
const collisionWarning = ref(false)
const collisionAgentName = ref('')
const customerReplyWarning = ref(false)
const showCollisionConfirm = ref(false)
let pendingSendAction = null

// Inbox switcher state
const inboxes = ref([])
const selectedInboxId = ref(null)

/**
 * Fetches AI prompts from the server.
 */
const fetchAiPrompts = async () => {
  try {
    const resp = await api.getAiPrompts()
    aiPrompts.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

/**
 * Fetches ecommerce configuration status.
 */
const fetchEcommerceStatus = async () => {
  try {
    const resp = await api.getEcommerceStatus()
    ecommerceConfigured.value = resp.data?.data?.configured || false
  } catch (error) {
    ecommerceConfigured.value = false
  }
}

/**
 * Fetches available inboxes for the From switcher.
 */
const fetchInboxes = async () => {
  try {
    const resp = await api.getInboxes()
    inboxes.value = (resp.data?.data || []).filter(i => i.enabled)
  } catch (error) {
    inboxes.value = []
  }
}

// Signature for current inbox
const inboxSignature = ref('')

// Fetch signature for a given inbox
const fetchInboxSignature = async (inboxId) => {
  if (!inboxId) return
  const conv = conversationStore.current
  try {
    const resp = await api.getInboxSignature(inboxId, conv?.uuid || '')
    inboxSignature.value = resp.data?.data?.signature || ''
  } catch (err) {
    inboxSignature.value = ''
  }
}

/**
 * Insert signature into editor content.
 * Replaces existing signature div if present, otherwise appends.
 */
const insertSignature = () => {
  if (!inboxSignature.value) return

  // Use a marker comment so we can find and replace the signature later.
  // TipTap strips div wrappers, so we use an HTML comment as a boundary.
  const sigMarker = '<!-- sig -->'
  const sigBlock = sigMarker + inboxSignature.value

  // If editor has existing signature, replace it
  if (htmlContent.value && htmlContent.value.includes(sigMarker)) {
    // Replace everything from marker to end
    htmlContent.value = htmlContent.value.substring(0, htmlContent.value.indexOf(sigMarker)) + sigBlock
    return
  }

  // If editor is empty or only whitespace/br tags, set signature as content
  const strippedContent = htmlContent.value
    ? htmlContent.value.replace(/<[^>]*>/g, '').trim()
    : ''

  if (!strippedContent) {
    htmlContent.value = '<p><br></p><p>' + sigBlock + '</p>'
  } else {
    // Append signature with a blank line separator
    htmlContent.value = htmlContent.value + '<p><br></p><p>' + sigBlock + '</p>'
  }
}

/**
 * Handle inbox change from the From switcher.
 */
const handleInboxChange = async (newInboxId) => {
  selectedInboxId.value = newInboxId
  await fetchInboxSignature(newInboxId)
  // Replace signature in editor
  const sigMarker = '<!-- sig -->'
  if (inboxSignature.value) {
    const sigBlock = sigMarker + inboxSignature.value
    if (htmlContent.value && htmlContent.value.includes(sigMarker)) {
      htmlContent.value = htmlContent.value.substring(0, htmlContent.value.indexOf(sigMarker)) + sigBlock
    } else {
      htmlContent.value = (htmlContent.value || '') + '<p><br></p><p>' + sigBlock + '</p>'
    }
  } else {
    // Remove existing signature if new inbox has none
    if (htmlContent.value && htmlContent.value.includes(sigMarker)) {
      htmlContent.value = htmlContent.value.substring(0, htmlContent.value.indexOf(sigMarker))
    }
  }
}

// Fetch data on mount
onMounted(() => {
  fetchAiPrompts()
  fetchEcommerceStatus()
  fetchInboxes()

  // Listen for new messages to detect other agent replies while composing
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, handleNewMessageCollision)
  emitter.on('set-reply-type', (type) => { messageType.value = type })
  emitter.on('shortcut-discard-or-collapse', handleEscapeShortcut)
})

// Clean up listener
onBeforeUnmount(() => {
  emitter.off(EMITTER_EVENTS.NEW_MESSAGE, handleNewMessageCollision)
  emitter.off('shortcut-discard-or-collapse', handleEscapeShortcut)
})

function handleEscapeShortcut() {
  // If fullscreen editor is open, close it first
  if (isEditorFullscreen.value) {
    isEditorFullscreen.value = false
    return
  }
  // If there's draft content, show discard confirmation
  if (hasDraftContent.value) {
    showDiscardDraft.value = true
  } else {
    // No content, just collapse
    emitter.emit('collapse-reply')
  }
}

/**
 * Handles collision detection when a new message arrives while composing.
 */
function handleNewMessageCollision({ conversation_uuid, message }) {
  if (!composingStartedAt.value) return
  if (conversation_uuid !== conversationStore.current?.uuid) return
  // Ignore private notes
  if (message?.private) return

  if (message?.type === 'incoming') {
    // Customer sent a new message while agent is composing
    customerReplyWarning.value = true
  } else if (message?.type === 'outgoing') {
    // Another agent replied while composing
    if (message?.sender_id === userStore.userID) return
    collisionWarning.value = true
    collisionAgentName.value = message?.sender?.first_name || 'Another agent'
  }
}

function dismissCollisionWarning() {
  collisionWarning.value = false
}

function dismissCustomerReplyWarning() {
  customerReplyWarning.value = false
}

function confirmSend() {
  showCollisionConfirm.value = false
  collisionWarning.value = false
  if (pendingSendAction === 'send') {
    doSend()
  } else if (pendingSendAction) {
    doSendWithStatus(pendingSendAction)
  }
  pendingSendAction = null
}

// When conversation changes, set selected inbox and fetch signature
watch(() => conversationStore.current?.uuid, async (newUuid) => {
  if (!newUuid) return

  // Ensure inboxes are loaded
  if (!inboxes.value.length) {
    await fetchInboxes()
  }

  const conv = conversationStore.current

  // Immediately set to conversation inbox (never leave as null)
  selectedInboxId.value = conv?.inbox_id || (inboxes.value.length ? inboxes.value[0].id : null)

  // Then try to resolve team default inbox override
  const teamId = conv?.assigned_team_id
  if (teamId) {
    try {
      const resp = await api.getTeamsCompact()
      const teams = resp.data?.data || []
      const team = teams.find(t => t.id === teamId)
      if (team?.default_inbox_id) {
        selectedInboxId.value = team.default_inbox_id
      }
    } catch (e) {
      // Keep conversation inbox as fallback
    }
  }

  await fetchInboxSignature(selectedInboxId.value)

  // Wait for draft to load, then insert signature if editor is empty
  setTimeout(() => {
    const strippedContent = htmlContent.value
      ? htmlContent.value.replace(/<[^>]*>/g, '').trim()
      : ''
    if (!strippedContent && inboxSignature.value && messageType.value !== 'private_note') {
      insertSignature()
    }
  }, 200)
}, { immediate: true })

// Toggle signature when switching between reply and private note
watch(messageType, (newType, oldType) => {
  const sigMarker = '<!-- sig -->'
  if (newType === 'private_note') {
    // Remove signature
    if (htmlContent.value && htmlContent.value.includes(sigMarker)) {
      htmlContent.value = htmlContent.value.substring(0, htmlContent.value.indexOf(sigMarker))
    }
  } else if (oldType === 'private_note' && inboxSignature.value) {
    // Re-add signature when switching back to reply
    if (!htmlContent.value || !htmlContent.value.includes(sigMarker)) {
      const sigBlock = sigMarker + inboxSignature.value
      const strippedContent = htmlContent.value
        ? htmlContent.value.replace(/<[^>]*>/g, '').trim()
        : ''
      if (!strippedContent) {
        htmlContent.value = '<p><br></p><p>' + sigBlock + '</p>'
      } else {
        htmlContent.value = htmlContent.value + '<p><br></p><p>' + sigBlock + '</p>'
      }
    }
  }
})

/**
 * Handles the AI prompt selection event.
 */
const handleAiPromptSelected = async (key) => {
  try {
    const resp = await api.aiCompletion({
      prompt_key: key,
      content: textContent.value
    })
    htmlContent.value = resp.data.data.replace(/\n/g, '<br>')
  } catch (error) {
    if (error.response?.status === 400 && userStore.can('ai:manage')) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'default',
        description: 'Please configure an AI provider in Settings > AI Settings'
      })
      return
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

/**
 * Handles generating a response using RAG.
 */
const handleGenerateResponse = async (includeEcommerce = false) => {
  isGenerating.value = true
  try {
    const messages = conversationStore.conversationMessages
      .filter(m => !m.private && m.content)
      .sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
      .slice(-10)

    if (!messages.length) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: "destructive",
        description: "No messages found in conversation"
      })
      return
    }

    const conversationText = messages.map(m => {
      const doc = new DOMParser().parseFromString(m.content || "", "text/html")
      const text = doc.body.textContent || ""
      const role = m.type === "incoming" ? "Customer" : "Agent"
      return role + ": " + text.trim()
    }).join("\n\n")

    if (!conversationText.trim()) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: "destructive",
        description: "Conversation content is empty"
      })
      return
    }

    const resp = await api.ragGenerate({
      conversation_id: conversationStore.current.id,
      inbox_id: selectedInboxId.value || conversationStore.current?.inbox_id || 0,
      customer_message: conversationText,
      include_ecommerce: includeEcommerce
    })

    if (resp.data?.data?.response) {
      const response = resp.data.data.response
      let generatedHtml
      if (/<[a-z][\s\S]*>/i.test(response)) {
        // HTML response: strip newlines between/around tags (source formatting),
        // convert remaining newlines (within text) to <br>, then clean up empties.
        generatedHtml = response
          .replace(/>\s*\n\s*/g, '>')   // strip newlines after closing >
          .replace(/\s*\n\s*</g, '<')   // strip newlines before opening <
          .replace(/\n/g, '<br>')         // remaining newlines are within text
        // Clean up empty elements that TipTap would render as blank bullets/lines
        generatedHtml = generatedHtml
          .replace(/<li>\s*(<br\s*\/?>\s*)*<\/li>/gi, '')
          .replace(/<p>\s*(<br\s*\/?>\s*)*<\/p>/gi, '')
      } else {
        // Plain text: convert double newlines to paragraphs, single to line breaks
        generatedHtml = '<p>' + response.replace(/\n{2,}/g, '</p><p>').replace(/\n/g, '<br>') + '</p>'
      }
      // Sanitize AI response: strip script tags and event handlers
      generatedHtml = generatedHtml.replace(/<script[\s\S]*?<\/script>/gi, '')
      generatedHtml = generatedHtml.replace(/\son\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]*)/gi, '')

      // Preserve signature if present
      const sigMarker = '<!-- sig -->'
      if (htmlContent.value && htmlContent.value.includes(sigMarker)) {
        const sigContent = htmlContent.value.substring(htmlContent.value.indexOf(sigMarker))
        htmlContent.value = generatedHtml + '<p><br></p><p>' + sigContent + '</p>'
      } else if (inboxSignature.value) {
        // Add signature after generated content
        htmlContent.value = generatedHtml + '<p><br></p><p>' + sigMarker + inboxSignature.value + '</p>'
      } else {
        htmlContent.value = generatedHtml
      }

      const successMsg = includeEcommerce
        ? "Response generated with order data"
        : "Response generated from knowledge base"
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: successMsg
      })
    }
  } catch (error) {
    if (error.response?.status === 400 && userStore.can("ai:manage")) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: "default",
        description: "Please configure an AI provider and knowledge sources in Settings"
      })
      return
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: "destructive",
      description: handleHTTPError(error).message
    })
  } finally {
    isGenerating.value = false
  }
}

const handleGenerateWithOrders = () => {
  handleGenerateResponse(true)
}

const updateProvider = async (values) => {
  try {
    isOpenAIKeyUpdating.value = true
    await api.updateAIProvider({ api_key: values.apiKey, provider: 'openai' })
    openAIKeyPrompt.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully', {
        name: t('globals.terms.apiKey')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isOpenAIKeyUpdating.value = false
  }
}

const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

// Track when agent starts composing
watch(textContent, (newVal) => {
  if (newVal && newVal.trim().length > 0 && !composingStartedAt.value) {
    composingStartedAt.value = new Date()
  }
})

/**
 * Checks for collision before sending. If another agent replied while composing, shows confirmation.
 */
const processSend = async () => {
  if (collisionWarning.value) {
    pendingSendAction = 'send'
    showCollisionConfirm.value = true
    return
  }
  await doSend()
}

/**
 * Actually sends the message.
 */
const doSend = async () => {
  let hasMessageSendingErrored = false
  isEditorFullscreen.value = false
  try {
    isSending.value = true
    if (hasTextContent.value > 0 || mediaFiles.value.length > 0) {
      const message = htmlContent.value
      const payload = {
        sender_type: UserTypeAgent,
        private: messageType.value === 'private_note',
        message: message,
        attachments: mediaFiles.value.map((file) => file.id),
        mentions: messageType.value === 'private_note' ? mentions.value : [],
        cc: cc.value
          .split(',')
          .map((email) => email.trim())
          .filter((email) => email),
        bcc: bcc.value
          ? bcc.value
              .split(',')
              .map((email) => email.trim())
              .filter((email) => email)
          : [],
        to: to.value
          ? to.value
              .split(',')
              .map((email) => email.trim())
              .filter((email) => email)
          : []
      }

      // Include inbox_id if agent selected a different inbox
      if (selectedInboxId.value && selectedInboxId.value !== conversationStore.current?.inbox_id) {
        payload.inbox_id = selectedInboxId.value
      }

      await api.sendMessage(conversationStore.current.uuid, payload)
    }

    // Apply macro actions if any
    const macroID = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.id
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    if (macroID > 0 && macroActions.length > 0) {
      try {
        await api.applyMacro(conversationStore.current.uuid, macroID, macroActions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  } catch (error) {
    hasMessageSendingErrored = true
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (hasMessageSendingErrored === false) {
      clearDraft(currentDraftKey.value)
      conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
      clearMediaFiles()
      emailErrors.value = []
      mentions.value = []
      // Reset collision state
      composingStartedAt.value = null
      collisionWarning.value = false
      collisionAgentName.value = ''
      customerReplyWarning.value = false
    }
    isSending.value = false
  }
}

/**
 * Send message and set conversation status in one action.
 */
const processSendWithStatus = async (status) => {
  if (collisionWarning.value) {
    pendingSendAction = status
    showCollisionConfirm.value = true
    return
  }
  await doSendWithStatus(status)
}

const doSendWithStatus = async (status) => {
  await doSend()
  // After successful send, update the conversation status
  if (!isSending.value) {
    try {
      await api.updateConversationStatus(conversationStore.current.uuid, { status })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }
}

const showDiscardDraft = ref(false)

/**
 * Show confirmation before deleting draft.
 */
const handleDeleteDraft = () => {
  showDiscardDraft.value = true
}

/**
 * Actually delete the draft after confirmation, then collapse the reply box.
 */
const confirmDeleteDraft = () => {
  clearDraft(currentDraftKey.value)
  clearMediaFiles()
  emailErrors.value = []
  mentions.value = []
  showDiscardDraft.value = false
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    description: 'Draft discarded'
  })
  // Collapse the reply box (Fresh theme)
  emitter.emit('collapse-reply')
}

/**
 * Whether the editor has draft content.
 */
const hasDraftContent = computed(() => {
  return hasTextContent.value > 0 || mediaFiles.value.length > 0
})

/**
 * Statuses available for "Send and set as" dropdown.
 */
const availableSendStatuses = computed(() => {
  return conversationStore.statuses
    .filter(s => s.show_on_send)
    .map(s => s.name)
})

watch(
  () => conversationStore.getMacro('reply').id,
  (newId) => {
    if (!newId) return
    const macroContent = conversationStore.getMacro('reply').message_content
    if (!macroContent) return
    // Insert at cursor position via ReplyBoxContent
    replyBoxContentRef.value?.insertMacro(macroContent)
  },
  { deep: true }
)

watch(
  loadedMacroActions,
  (actions) => {
    if (actions.length > 0) {
      conversationStore.setMacroActions([...toRaw(actions)], MACRO_CONTEXT.REPLY)
    }
  },
  { deep: true }
)

watch(
  loadedAttachments,
  (attachments) => {
    if (attachments.length > 0) {
      setMediaFiles([...attachments])
    }
  },
  { deep: true }
)

watch(
  () => conversationStore.currentCC,
  (newVal) => {
    cc.value = newVal?.join(', ') || ''
  },
  { deep: true, immediate: true }
)

watch(
  () => conversationStore.currentTo,
  (newVal) => {
    to.value = newVal?.join(', ') || ''
  },
  { immediate: true }
)

watch(
  () => conversationStore.currentBCC,
  (newVal) => {
    const newBcc = newVal?.join(', ') || ''
    bcc.value = newBcc
    if (newBcc.length > 0) {
      showBcc.value = true
    }
  },
  { deep: true, immediate: true }
)

watch(
  () => conversationStore.current?.uuid,
  () => {
    clearMediaFiles()
    conversationStore.resetMacro(MACRO_CONTEXT.REPLY)
    // Reset collision state
    composingStartedAt.value = null
    collisionWarning.value = false
    collisionAgentName.value = ''
    customerReplyWarning.value = false
    setTimeout(() => {
      replyBoxContentRef.value?.focus()
    }, 100)
  }
)
</script>
