<template>
  <div>
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="max-w-5xl w-full h-[90vh] flex flex-col" >
        <DialogHeader>
          <DialogTitle>
            {{ $t('conversation.newConversation') }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>

        <form @submit="createConversation" class="flex flex-col flex-1 overflow-hidden">
          <!-- Form Fields Section -->
          <div class="space-y-4 pb-2 flex-shrink-0">
            <div class="space-y-2">
              <!--
                EC7: Gmail-style TO/CC/BCC composer for new conversation.
                Replaces the single email field with a chip-input that supports
                multiple recipients plus toggleable CC/BCC fields. The first
                TO email is the primary contact (used to find/create the User
                row); additional TO emails are accepted in the chip UI but the
                backend currently only consumes the first (parity with v1.0.3
                EC7). CC/BCC are sent through to QueueReply via new
                createConversationRequest.CC / .BCC fields.

                contact_email (the form-validated field) is kept in sync with
                the FIRST email of the chip-input via the watcher below.
              -->
              <div class="space-y-2">
                <!-- TO field with Cc/Bcc toggles -->
                <FormField name="contact_email">
                  <FormItem class="relative">
                    <div class="flex items-center space-x-2">
                      <label class="w-10 text-sm font-medium text-muted-foreground shrink-0">TO:</label>
                      <FormControl class="flex-grow">
                        <EmailTagInput
                          v-model="emailQuery"
                          :placeholder="t('conversation.searchContact')"
                          class="flex-grow"
                          @blur="handleToBlur"
                          @contactSelected="selectContact"
                        />
                      </FormControl>
                      <div class="flex items-center gap-1 shrink-0">
                        <button
                          v-if="!showCc"
                          type="button"
                          @click="showCc = true"
                          class="text-xs text-muted-foreground hover:text-foreground transition-colors px-1"
                        >{{ $t('conversation.cc') }}</button>
                        <button
                          v-if="!showBcc"
                          type="button"
                          @click="showBcc = true"
                          class="text-xs text-muted-foreground hover:text-foreground transition-colors px-1"
                        >{{ $t('conversation.bcc') }}</button>
                      </div>
                    </div>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <!-- CC field -->
                <div v-if="showCc" class="flex items-center space-x-2">
                  <label class="w-10 text-sm font-medium text-muted-foreground shrink-0">CC:</label>
                  <EmailTagInput
                    v-model="ccEmails"
                    :placeholder="t('conversation.addCcRecipients')"
                    class="flex-grow"
                  />
                  <button
                    type="button"
                    @click="showCc = false; ccEmails = ''"
                    class="text-muted-foreground hover:text-foreground transition-colors shrink-0 p-1"
                    :title="$t('conversation.removeCc')"
                    :aria-label="$t('conversation.removeCc')"
                  >
                    <X class="h-3.5 w-3.5" />
                  </button>
                </div>
                <!-- BCC field -->
                <div v-if="showBcc" class="flex items-center space-x-2">
                  <label class="w-10 text-sm font-medium text-muted-foreground shrink-0">BCC:</label>
                  <EmailTagInput
                    v-model="bccEmails"
                    :placeholder="t('conversation.addBccRecipients')"
                    class="flex-grow"
                  />
                  <button
                    type="button"
                    @click="showBcc = false; bccEmails = ''"
                    class="text-muted-foreground hover:text-foreground transition-colors shrink-0 p-1"
                    :title="$t('conversation.removeBcc')"
                    :aria-label="$t('conversation.removeBcc')"
                  >
                    <X class="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>

              <!-- Name Group -->
              <div class="grid grid-cols-2 gap-4">
                <FormField v-slot="{ componentField }" name="first_name">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.firstName') }}</FormLabel>
                    <FormControl>
                      <Input
                        type="text"
                        placeholder=""
                        v-bind="componentField"
                        :disabled="!!selectedContact"
                        required
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="last_name">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.lastName') }}</FormLabel>
                    <FormControl>
                      <Input
                        type="text"
                        placeholder=""
                        v-bind="componentField"
                        :disabled="!!selectedContact"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Subject and Inbox Group -->
              <div class="grid grid-cols-2 gap-4">
                <FormField v-slot="{ componentField }" name="subject">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.subject') }}</FormLabel>
                    <FormControl>
                      <Input type="text" placeholder="" v-bind="componentField" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="inbox_id">
                  <FormItem>
                    <FormLabel>{{ $t('globals.terms.inbox') }}</FormLabel>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue :placeholder="t('placeholders.selectInbox')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem
                              v-for="option in inboxStore.emailOptions"
                              :key="option.value"
                              :value="option.value"
                            >
                              {{ option.label }}
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Assignment Group -->
              <div class="grid grid-cols-2 gap-4">
                <!-- Set assigned team -->
                <FormField v-slot="{ componentField }" name="team_id">
                  <FormItem>
                    <FormLabel>
                      {{ $t('actions.assignTeam') }}
                      ({{ $t('globals.terms.optional') }})
                    </FormLabel>
                    <FormControl>
                      <SelectComboBox
                        v-bind="componentField"
                        :items="[
                          { value: 'none', label: t('globals.terms.none') },
                          ...teamStore.options
                        ]"
                        :placeholder="t('placeholders.selectTeam')"
                        type="team"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Set assigned agent -->
                <FormField v-slot="{ componentField }" name="agent_id">
                  <FormItem>
                    <FormLabel>
                      {{ $t('actions.assignAgent') }}
                      ({{ $t('globals.terms.optional') }})
                    </FormLabel>
                    <FormControl>
                      <SelectComboBox
                        v-bind="componentField"
                        :items="[
                          { value: 'none', label: t('globals.terms.none') },
                          ...uStore.options
                        ]"
                        :placeholder="t('placeholders.selectAgent')"
                        type="user"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>
            </div>
          </div>

          <!-- Message Editor Section -->
          <div class="flex-1 flex flex-col min-h-0 mt-4">
            <FormField v-slot="{ componentField }" name="content">
              <FormItem class="flex flex-col h-full">
                <FormLabel>{{ $t('globals.terms.message') }}</FormLabel>
                <FormControl class="flex-1 flex flex-col min-h-0">
                  <div class="flex flex-col h-full">
                    <Editor
                      ref="createEditorRef"
                      v-model:htmlContent="componentField.modelValue"
                      @update:htmlContent="(value) => componentField.onChange(value)"
                      :placeholder="t('editor.hint.newLineCtrlK')"
                      :insertContent="insertContent"
                      :autoFocus="false"
                      class="w-full flex-1 overflow-y-auto p-2 box min-h-0"
                      @send="createConversation"
                    />

                    <MacroActionsPreview
                      v-if="
                        conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).actions?.length >
                        0
                      "
                      :actions="
                        conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION)?.actions || []
                      "
                      :onRemove="
                        (action) =>
                          conversationStore.removeMacroAction(
                            action,
                            MACRO_CONTEXT.NEW_CONVERSATION
                          )
                      "
                      class="mt-2 flex-shrink-0"
                    />

                    <AttachmentsPreview
                      :attachments="mediaFiles"
                      :uploadingFiles="uploadingFiles"
                      :onDelete="handleFileDelete"
                      v-if="mediaFiles.length > 0 || uploadingFiles.length > 0"
                      class="mt-2 flex-shrink-0"
                    />
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <DialogFooter class="mt-4 pt-2 flex items-center !justify-between w-full flex-shrink-0">
            <ReplyBoxMenuBar
              :handleFileUpload="handleFileUpload"
              @emojiSelect="handleEmojiSelect"
              @editorCommand="(cmd) => createEditorRef?.runCommand(cmd)"
              :showSendButton="false"
              macroPickerCommand="apply-macro-to-new-conversation"
            />
            <Button type="submit" :disabled="isDisabled" :isLoading="loading">
              {{ $t('globals.messages.submit') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import EmailTagInput from '@/components/EmailTagInput.vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { z } from 'zod'
import { ref, watch, onUnmounted, nextTick, onMounted, computed } from 'vue'
import AttachmentsPreview from '@/features/conversation/message/attachment/AttachmentsPreview.vue'
import { useConversationStore } from '../../stores/conversation'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useInboxStore } from '@main/stores/inbox'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import { useUserStore } from '@main/stores/user'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { useI18n } from 'vue-i18n'
import { useFileUpload } from '@/composables/useFileUpload'
import Editor from '@/components/editor/TextEditor.vue'
import { useMacroStore } from '@/stores/macro'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { UserTypeAgent } from '@/constants/user'
import { X } from 'lucide-vue-next'
import api from '@/api'

const dialogOpen = defineModel({
  required: false,
  default: () => false
})

const inboxStore = useInboxStore()
const { t } = useI18n()
const uStore = useUsersStore()
const teamStore = useTeamStore()
const userStore = useUserStore()
const emitter = useEmitter()
const loading = ref(false)
// EC7: emailQuery is the chip-input model — a comma-joined string of TO
// recipients. The first one is the primary contact (drives the User row
// lookup); contact_email in the form mirrors only the first email.
const emailQuery = ref('')
const conversationStore = useConversationStore()
const macroStore = useMacroStore()
const insertContent = ref('')
// EC12: Editor ref so the formatting toolbar in the embedded ReplyBoxMenuBar
// can call into the editor's exposed runCommand() (bold / italic / list /
// link / image insertion).
const createEditorRef = ref(null)
// Tracks the contact selected from the suggestions dropdown so we can
// auto-disable name fields and reset them when the agent backspaces the
// matching email out of the chip-input.
const selectedContact = ref(null)
// EC7: CC/BCC are toggled visible like Gmail; their chip-input models live
// independently of TO and are submitted as separate fields on the request.
const showCc = ref(false)
const showBcc = ref(false)
const ccEmails = ref('')
const bccEmails = ref('')

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

const { uploadingFiles, handleFileUpload, handleFileDelete, mediaFiles, clearMediaFiles } =
  useFileUpload({
    linkedModel: 'messages'
  })

const isDisabled = computed(() => {
  if (loading.value || uploadingFiles.value.length > 0) {
    return true
  }
  return false
})

const formSchema = z.object({
  subject: z.string().min(1, t('validation.subjectCannotBeEmpty')),
  content: z.string().min(1, t('validation.messageCannotBeEmpty')),
  inbox_id: z
    .any()
    .refine((val) => inboxStore.emailOptions.some((option) => option.value === val), {
      message: t('globals.messages.required')
    }),
  team_id: z.any().optional(),
  agent_id: z.any().optional(),
  contact_email: z.string().email(t('validation.invalidEmail')),
  first_name: z.string().min(1, t('globals.messages.required')),
  last_name: z.string().optional()
})

onUnmounted(() => {
  clearMediaFiles()
  conversationStore.resetMacro(MACRO_CONTEXT.NEW_CONVERSATION)
  emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
    command: null,
    open: false
  })
})

onMounted(() => {
  macroStore.setCurrentView('starting_conversation')
  emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
    command: 'apply-macro-to-new-conversation',
    open: false
  })
  // EC7: EmailTagInput owns its own internal focus — we let the user click
  // it. Auto-focusing the chip's hidden input would be inconsistent with
  // its native click-to-focus UX.
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    inbox_id: null,
    team_id: null,
    agent_id: null,
    subject: '',
    content: '',
    contact_email: '',
    first_name: '',
    last_name: ''
  }
})

// EC16: Smart new-conversation defaults. When the dialog opens, prefill the
// assignee with the current agent — the typical case is "I'm starting this
// conversation, so I own it". Only sets when empty so reopening the dialog
// after the agent explicitly cleared the field doesn't undo their choice.
//
// Adaptation delta vs v1.0.3: the source commit (c7b60817) also auto-selected
// the agent's team default inbox via team.default_inbox_id. v2 has no
// default_inbox_id column on the team model (MP-class data migration not
// ported), so the inbox-defaulting half is dropped here. The existing
// auto-select-first-inbox logic in T2h still picks a sensible inbox; this
// commit only adds the agent-assign half.
watch(
  dialogOpen,
  (open) => {
    if (!open) return
    if (!form.values.agent_id && userStore.userID) {
      form.setFieldValue('agent_id', String(userStore.userID))
    }
  },
  { immediate: true }
)

// Keep the validated contact_email field in sync with the FIRST chip.
// If the agent erases the chip that matches the picked contact, clear the
// auto-filled name fields so they don't ship stale data with a different
// primary recipient.
watch(emailQuery, (newVal) => {
  const firstEmail = newVal.split(',').map((e) => e.trim()).filter((e) => e)[0] || ''
  form.setFieldValue('contact_email', firstEmail)
  if (selectedContact.value && !newVal.includes(selectedContact.value.email)) {
    selectedContact.value = null
    form.setFieldValue('first_name', '')
    form.setFieldValue('last_name', '')
  }
})

// EmailTagInput emits @blur with no payload — re-derive the first email from
// the chip-input model. Same intent as the watcher; this catches the case
// where the chip was added without a reactive update yet (e.g. blur before
// the next tick).
const handleToBlur = () => {
  const firstEmail = emailQuery.value.split(',').map((e) => e.trim()).filter((e) => e)[0] || ''
  form.setFieldValue('contact_email', firstEmail)
}

// Triggered when the agent picks a suggestion from EmailTagInput's contact
// dropdown. The chip is already added by the component itself; here we just
// fill the name fields — but only if they're empty, so picking a SECOND
// contact (an additional recipient) doesn't silently overwrite the first
// contact's name.
const selectContact = (contact) => {
  selectedContact.value = contact
  if (!form.values.first_name) {
    form.setFieldValue('first_name', contact.first_name || '')
  }
  if (!form.values.last_name) {
    form.setFieldValue('last_name', contact.last_name || '')
  }
}

const createConversation = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    // Convert ids to numbers if they are not already
    values.inbox_id = Number(values.inbox_id)
    values.team_id = values.team_id ? Number(values.team_id) : null
    values.agent_id = values.agent_id ? Number(values.agent_id) : null
    // Array of attachment ids.
    values.attachments = mediaFiles.value.map((file) => file.id)
    // EC7: Pass through CC/BCC chip-input contents as comma-separated strings.
    // Backend (cmd/conversation.go) splits, trims, dedupes, then forwards to
    // QueueReply.
    values.cc = ccEmails.value || ''
    values.bcc = bccEmails.value || ''
    // Initiator of this conversation is always agent
    values.initiator = UserTypeAgent
    const conversation = await api.createConversation(values)
    const conversationUUID = conversation.data.data.uuid

    // Get macro from context, and set if any actions are available.
    const macro = conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION)
    if (conversationUUID !== '' && macro?.id && macro?.actions?.length > 0) {
      try {
        await api.applyMacro(conversationUUID, macro.id, macro.actions)
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
    dialogOpen.value = false
    form.resetForm()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
})

/**
 * Watches for changes in the macro id and update message content.
 */
watch(
  () => conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).id,
  () => {
    form.setFieldValue(
      'content',
      conversationStore.getMacro(MACRO_CONTEXT.NEW_CONVERSATION).message_content
    )
  },
  { deep: true }
)
</script>
