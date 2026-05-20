<template>
  <div>
    <ConversationSideBarContact class="p-4" />
    <Accordion type="multiple" collapsible v-model="accordionState">
      <AccordionItem value="actions" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('globals.terms.action', 2) }}
        </AccordionTrigger>

        <!-- Agent, team, priority, and tags assignment -->
        <AccordionContent class="accordion-content--actions">
          <div>
            <SelectComboBox
              v-model="conversationStore.current.assigned_user_id"
              :items="[{ value: 'none', label: t('globals.terms.none') }, ...usersStore.options]"
              :placeholder="t('placeholders.selectAgent')"
              @select="selectAgent"
              type="user"
            />
          </div>

          <div>
            <SelectComboBox
              v-model="conversationStore.current.assigned_team_id"
              :items="[{ value: 'none', label: t('globals.terms.none') }, ...teamsStore.options]"
              :placeholder="t('placeholders.selectTeam')"
              @select="selectTeam"
              type="team"
            />
          </div>

          <div>
            <SelectComboBox
              v-model="conversationStore.current.priority_id"
              :items="priorityOptions"
              :placeholder="t('placeholders.selectPriority')"
              @select="selectPriority"
              type="priority"
            />
          </div>

          <div>
            <SelectTag
              v-if="conversationStore.current"
              v-model="conversationStore.current.tags"
              :items="tags.map((tag) => ({ label: tag, value: tag }))"
              :placeholder="t('placeholders.selectTags')"
            />
          </div>
        </AccordionContent>
      </AccordionItem>

      <!-- Information -->
      <AccordionItem value="information" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.information') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <ConversationInfo />
        </AccordionContent>
      </AccordionItem>

      <!-- Contact attributes -->
      <AccordionItem
        value="contact_attributes"
        class="accordion-item"
        v-if="customAttributeStore.contactAttributeOptions.length > 0"
      >
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.contactAttributes') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <CustomAttributes
            :loading="conversationStore.current.loading"
            :attributes="customAttributeStore.contactAttributeOptions"
            :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
            @update:setattributes="updateContactCustomAttributes"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Previous conversations -->
      <AccordionItem value="previous_conversations" class="accordion-item">
        <AccordionTrigger class="accordion-trigger">
          {{ $t('conversation.sidebar.previousConvo') }}
        </AccordionTrigger>
        <AccordionContent class="accordion-content">
          <PreviousConversations />
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { useToast } from '../../../composables/useToast'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '../../../stores/customAttributes'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import { TAG_ACTION } from '@/constants/conversation'
import api from '../../../api'

// Module-scoped cache: survives component remounts. The sidebar gets
// re-mounted many times per ticket switch in v2 (SplitterGroup layout
// cascade), so a per-instance guard is useless — each fresh mount would
// re-fetch. Sharing across mounts collapses 50 requests into 1.
let lastFetchedFollowersUuid = null

const customAttributeStore = useCustomAttributeStore()
const toast = useToast()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const tags = ref([])
// Step 7a: followers state (unused yet)
const followers = ref([])
const syncingFollowers = ref(false)
const accordionState = useStorage('conversation-sidebar-accordion', ['previous_conversations'])
const { t } = useI18n()
let isConversationChange = false
customAttributeStore.fetchCustomAttributes()

// Watch the uuid (string) rather than the current object — the store's
// `current` computed returns a fresh `{}` whenever data is briefly falsy
// during loading, so watching the object identity fires this watcher many
// times per conversation switch and produces a burst of fetchFollowers
// calls. Watching the uuid is identity-stable and only fires on real
// conversation transitions.
watch(
  () => conversationStore.current?.uuid,
  (newUuid, oldUuid) => {
    if (newUuid && newUuid !== oldUuid) {
      isConversationChange = true
      fetchFollowers()
    }
  },
  { immediate: true }
)

onMounted(async () => {
  await fetchTags()
})

// Watch for changes in the tags and upsert the tags
watch(
  () => conversationStore.current?.tags,
  (newTags, oldTags) => {
    if (isConversationChange) {
      isConversationChange = false
      return
    }

    if (!Array.isArray(newTags) || !Array.isArray(oldTags)) {
      return
    }

    if (
      newTags.length === oldTags.length &&
      newTags.every((item) => oldTags.includes(item))
    ) {
      return
    }

    // PR #286: store API changed from upsertTags({tags}) to
    // updateConversationTags(uuid, action, tags). Sidebar always SETs the
    // full tag list (multi-select picker emits the new full array).
    conversationStore.updateConversationTags(
      conversationStore.current.uuid,
      TAG_ACTION.SET,
      newTags
    )
  },
  { immediate: false }
)

const priorityOptions = computed(() => conversationStore.priorityOptions)

const fetchFollowers = async () => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  if (uuid === lastFetchedFollowersUuid) return
  lastFetchedFollowersUuid = uuid
  try {
    const res = await api.getConversationParticipants(uuid)
    followers.value = (res.data?.data || []).filter((f) => {
      const name = ((f.first_name || '') + ' ' + (f.last_name || '')).toLowerCase().trim()
      return name !== 'system' && name !== 'system user'
    })
  } catch {
    // Non-fatal — empty followers picker.
  }
}

const fetchTags = async () => {
  await tagStore.fetchTags()
  tags.value = tagStore.tags.map((item) => item.name)
}

const handleAssignedUserChange = (id) => {
  conversationStore.updateAssignee('user', {
    assignee_id: parseInt(id)
  })
}

const handleAssignedTeamChange = (id) => {
  conversationStore.updateAssignee('team', {
    assignee_id: parseInt(id)
  })
}

const handleRemoveAssignee = (type) => {
  conversationStore.removeAssignee(type)
}

const handlePriorityChange = (priority) => {
  conversationStore.updatePriority(priority)
}

const selectAgent = (agent) => {
  if (agent.value === 'none') {
    handleRemoveAssignee('user')
    return
  }
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  if (team.value === 'none') {
    handleRemoveAssignee('team')
    return
  }
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority.label
  conversationStore.current.priority_id = priority.value
  handlePriorityChange(priority.label)
}

const updateContactCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.contact.custom_attributes
  try {
    conversationStore.current.contact.custom_attributes = attributes
    await api.updateContactCustomAttribute(conversationStore.current.uuid, attributes)
    toast.success(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    toast.error(error)
    conversationStore.current.contact.custom_attributes = previousAttributes
  }
}
</script>

<style scoped>
:deep(.accordion-item) {
  @apply border-0 mb-2;
}

:deep(.accordion-trigger) {
  @apply bg-muted p-2 text-sm font-medium rounded mx-2;
}

:deep(.accordion-content) {
  @apply p-4;
}

:deep(.accordion-content--actions) {
  @apply space-y-3 p-4;
}
</style>
