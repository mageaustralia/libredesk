<template>
  <div class="space-y-4">
    <div class="flex flex-col" v-if="conversation.subject">
      <p class="font-medium">{{ $t('globals.terms.subject') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <div v-else class="group flex items-start gap-1">
        <p v-if="!editingSubject" class="flex-1 break-words">
          {{ conversation.subject }}
        </p>
        <input
          v-if="editingSubject"
          ref="subjectInput"
          v-model="subjectDraft"
          class="flex-1 text-sm border rounded px-2 py-1"
          @keyup.enter="saveSubject"
          @keyup.escape="editingSubject = false"
        />
        <button
          v-if="!editingSubject"
          class="opacity-0 group-hover:opacity-100 transition-opacity mt-0.5 text-muted-foreground hover:text-foreground"
          @click="startEditSubject"
        >
          <Pencil :size="14" />
        </button>
        <button
          v-if="editingSubject"
          class="mt-0.5 text-muted-foreground hover:text-foreground"
          @click="saveSubject"
        >
          <Check :size="14" />
        </button>
      </div>
    </div>

    <div class="flex flex-col">
      <p class="font-medium">{{ $t('globals.terms.referenceNumber') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-else>
        {{ conversation.reference_number }}
      </p>
    </div>
    <div class="flex flex-col">
      <p class="font-medium">{{ $t('globals.terms.initiatedAt') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-if="conversation.created_at">
        {{ format(conversation.created_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('globals.terms.firstReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.first_response_deadline_at"
          :dueAt="conversation.first_response_deadline_at"
          :actualAt="conversation.first_reply_at"
          :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <div v-else>
        <p v-if="conversation.first_reply_at">
          {{ format(conversation.first_reply_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
      </div>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('globals.terms.resolvedAt') }}</p>
        <SlaBadge
          v-if="conversation.resolution_deadline_at"
          :dueAt="conversation.resolution_deadline_at"
          :actualAt="conversation.resolved_at"
          :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <div v-else>
        <p v-if="conversation.resolved_at">
          {{ format(conversation.resolved_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
      </div>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('globals.terms.lastReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.next_response_deadline_at"
          :dueAt="conversation.next_response_deadline_at"
          :actualAt="conversation.next_response_met_at"
          :key="`${conversation.uuid}-${conversation.next_response_deadline_at}-${conversation.next_response_met_at}`"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-if="conversation.last_reply_at">
        {{ format(conversation.last_reply_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>

    <div class="flex flex-col" v-if="conversation.closed_at">
      <p class="font-medium">{{ $t('globals.terms.closedAt') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-else>
        {{ format(conversation.closed_at, 'PPpp') }}
      </p>
    </div>

    <div class="flex flex-col" v-if="conversation.sla_policy_name">
      <p class="font-medium">{{ $t('globals.terms.slaPolicy') }}</p>
      <div>
        <p>
          {{ conversation.sla_policy_name }}
        </p>
      </div>
    </div>

    <CustomAttributes
      v-if="customAttributeStore.conversationAttributeOptions.length > 0"
      :loading="conversationStore.conversation.loading"
      :attributes="customAttributeStore.conversationAttributeOptions"
      :custom-attributes="conversation.custom_attributes || {}"
      @update:setattributes="updateCustomAttributes"
    />
  </div>
</template>

<script setup>
import { computed, ref, nextTick } from 'vue'
import { format } from 'date-fns'
import { Pencil, Check } from 'lucide-vue-next'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { useI18n } from 'vue-i18n'

const emitter = useEmitter()
const { t } = useI18n()
const customAttributeStore = useCustomAttributeStore()
const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
customAttributeStore.fetchCustomAttributes()

const editingSubject = ref(false)
const subjectDraft = ref('')
const subjectInput = ref(null)

const startEditSubject = () => {
  subjectDraft.value = conversation.value.subject
  editingSubject.value = true
  nextTick(() => subjectInput.value?.focus())
}

const saveSubject = async () => {
  const trimmed = subjectDraft.value.trim()
  if (!trimmed || trimmed === conversation.value.subject) {
    editingSubject.value = false
    return
  }
  try {
    await api.updateConversationSubject(conversation.value.uuid, trimmed)
    conversationStore.current.subject = trimmed
    editingSubject.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Subject updated' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const updateCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.custom_attributes
  try {
    conversationStore.current.custom_attributes = attributes
    await api.updateConversationCustomAttribute(conversation.value.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.attribute')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conversationStore.current.custom_attributes = previousAttributes
  }
}
</script>
