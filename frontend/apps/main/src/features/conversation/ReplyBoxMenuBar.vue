<template>
  <div
    class="flex justify-between h-14 relative"
    :class="{ 'items-end': isFullscreen, 'items-center': !isFullscreen }"
  >
    <EmojiPicker
      ref="emojiPickerRef"
      :native="true"
      @select="onSelectEmoji"
      class="absolute bottom-14 left-14"
      v-if="isEmojiPickerVisible"
    />
    <div class="flex justify-items-start gap-2">
      <!-- File inputs -->
      <input type="file" class="hidden" ref="attachmentInput" multiple @change="handleFileUpload" />
      <!-- <input
        type="file"
        class="hidden"
        ref="inlineImageInput"
        accept="image/*"
        @change="handleInlineImageUpload"
      /> -->
      <!-- Editor buttons -->
      <Toggle
        class="px-2 py-2 border-0"
        variant="outline"
        @click="triggerFileUpload"
        :pressed="false"
      >
        <Paperclip class="h-4 w-4" />
      </Toggle>
      <Toggle
        class="px-2 py-2 border-0"
        variant="outline"
        @click="toggleEmojiPicker"
        :pressed="isEmojiPickerVisible"
      >
        <Smile class="h-4 w-4" />
      </Toggle>
    </div>
    <div class="flex items-center" v-if="showSendButton">
      <!-- Delete-draft button. Only surfaces when there's something to discard
           so the chrome doesn't add visual noise on an empty editor. -->
      <Button
        v-if="hasDraft"
        variant="ghost"
        size="sm"
        class="h-8 px-2 mr-1 text-muted-foreground hover:text-destructive"
        @click="$emit('deleteDraft')"
        :title="$t('replyBox.deleteDraft')"
      >
        <Trash2 class="h-4 w-4" />
      </Button>
      <!-- Split Send button: primary action stays on the left, status
           variants live behind a chevron on the right so an agent can
           "Send & Resolve" / "Send & Close" / etc. in one click. -->
      <div class="flex">
        <Button
          class="h-8 px-8 rounded-r-none"
          @click="handleSend"
          :disabled="!enableSend"
          :isLoading="isSending"
        >
          {{ $t('globals.messages.send') }}
        </Button>
        <DropdownMenu v-if="sendStatuses.length > 0">
          <DropdownMenuTrigger as-child>
            <Button
              class="h-8 px-1.5 rounded-l-none border-l border-primary-foreground/20"
              :disabled="!enableSend || isSending"
              :isLoading="isSending"
              :title="$t('replyBox.sendAndSetStatus')"
            >
              <ChevronDown class="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-56">
            <DropdownMenuItem
              v-for="status in sendStatuses"
              :key="status"
              @click="$emit('sendWithStatus', status)"
            >
              {{ $t('replyBox.sendAndSetAs', { status }) }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Toggle } from '@shared-ui/components/ui/toggle'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Paperclip, Smile, ChevronDown, Trash2 } from 'lucide-vue-next'

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([
    import('vue3-emoji-picker'),
    import('vue3-emoji-picker/css'),
  ])
  return mod.default
})

const attachmentInput = ref(null)
// const inlineImageInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['emojiSelect', 'sendWithStatus', 'deleteDraft'])

// Using defineProps for props that don't need two-way binding
defineProps({
  isFullscreen: Boolean,
  isSending: Boolean,
  enableSend: Boolean,
  handleSend: Function,
  showSendButton: {
    type: Boolean,
    default: true
  },
  handleFileUpload: Function,
  handleInlineImageUpload: Function,
  // Whether the editor has anything worth discarding. Drives visibility of
  // the delete-draft button — no point surfacing it on an empty box.
  hasDraft: {
    type: Boolean,
    default: false
  },
  // Status names to expose in the "Send & set as" dropdown. Parent decides
  // which statuses are valid for this conversation (typically excludes
  // Snoozed, Spam, Trashed since those are surfaced via dedicated UI).
  sendStatuses: {
    type: Array,
    default: () => []
  }
})

onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  if (attachmentInput.value) {
    // Clear the value to allow the same file to be uploaded again.
    attachmentInput.value.value = ''
    attachmentInput.value.click()
  }
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

function onSelectEmoji(emoji) {
  emit('emojiSelect', emoji.i)
}
</script>
