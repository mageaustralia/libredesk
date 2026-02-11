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

  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[60%] max-h-[75%] h-[70%] bg-card text-card-foreground p-4 flex flex-col"
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
          @aiPromptSelected="handleAiPromptSelected"
          @generateResponse="handleGenerateResponse"
          @generateWithOrders="handleGenerateWithOrders"
          @inboxChange="handleInboxChange"
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
        @aiPromptSelected="handleAiPromptSelected"
        @generateResponse="handleGenerateResponse"
        @generateWithOrders="handleGenerateWithOrders"
        @inboxChange="handleInboxChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, toRaw, onMounted } from 'vue'
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

  const sigBlock = '<div class="email-signature">' + inboxSignature.value + '</div>'
  const newContent = '<p><br></p>' + sigBlock

  // If editor has existing signature, replace it
  if (htmlContent.value && htmlContent.value.includes('class="email-signature"')) {
    htmlContent.value = htmlContent.value.replace(
      /<div class="email-signature">[\s\S]*?<\/div>/,
      sigBlock
    )
    return
  }

  // If editor is empty or only whitespace/br tags, set signature as content
  const strippedContent = htmlContent.value
    ? htmlContent.value.replace(/<[^>]*>/g, '').trim()
    : ''

  if (!strippedContent) {
    htmlContent.value = newContent
  } else {
    // Append signature to existing content
    htmlContent.value = htmlContent.value + '<p><br></p>' + sigBlock
  }
}

/**
 * Handle inbox change from the From switcher.
 */
const handleInboxChange = async (newInboxId) => {
  selectedInboxId.value = newInboxId
  await fetchInboxSignature(newInboxId)
  // Replace signature in editor
  if (inboxSignature.value) {
    const sigBlock = '<div class="email-signature">' + inboxSignature.value + '</div>'
    if (htmlContent.value && htmlContent.value.includes('class="email-signature"')) {
      htmlContent.value = htmlContent.value.replace(
        /<div class="email-signature">[\s\S]*?<\/div>/,
        sigBlock
      )
    } else {
      htmlContent.value = (htmlContent.value || '') + '<p><br></p>' + sigBlock
    }
  } else {
    // Remove existing signature if new inbox has none
    if (htmlContent.value && htmlContent.value.includes('class="email-signature"')) {
      htmlContent.value = htmlContent.value.replace(
        /<p><br><\/p><div class="email-signature">[\s\S]*?<\/div>/,
        ''
      )
    }
  }
}

// Fetch data on mount
onMounted(() => {
  fetchAiPrompts()
  fetchEcommerceStatus()
  fetchInboxes()
})

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
    if (!strippedContent && inboxSignature.value) {
      insertSignature()
    }
  }, 200)
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
      const tempDiv = document.createElement("div")
      tempDiv.innerHTML = m.content || ""
      const text = tempDiv.textContent || tempDiv.innerText || ""
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
      customer_message: conversationText,
      include_ecommerce: includeEcommerce
    })

    if (resp.data?.data?.response) {
      const response = resp.data.data.response
      let generatedHtml
      if (/<[a-z][\s\S]*>/i.test(response)) {
        generatedHtml = response.replace(/\n+/g, '')
      } else {
        generatedHtml = response.replace(/\n/g, '<br>')
      }

      // Preserve signature if present
      if (htmlContent.value && htmlContent.value.includes('class="email-signature"')) {
        const sigMatch = htmlContent.value.match(/<div class="email-signature">[\s\S]*?<\/div>/)
        if (sigMatch) {
          htmlContent.value = generatedHtml + '<p><br></p>' + sigMatch[0]
        } else {
          htmlContent.value = generatedHtml
        }
      } else if (inboxSignature.value) {
        // Add signature after generated content
        const sigBlock = '<div class="email-signature">' + inboxSignature.value + '</div>'
        htmlContent.value = generatedHtml + '<p><br></p>' + sigBlock
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

/**
 * Processes the send action.
 */
const processSend = async () => {
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
    }
    isSending.value = false
  }
}

watch(
  () => conversationStore.getMacro('reply').id,
  (newId) => {
    if (!newId) return
    if (conversationStore.getMacro('reply').message_content) {
      const macroContent = conversationStore.getMacro('reply').message_content
      htmlContent.value = htmlContent.value ? htmlContent.value + macroContent : macroContent
    }
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
    setTimeout(() => {
      replyBoxContentRef.value?.focus()
    }, 100)
  }
)
</script>
