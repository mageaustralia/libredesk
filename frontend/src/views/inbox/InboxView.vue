<template>
  <ConversationPlaceholder v-if="['inbox', 'team-inbox', 'view-inbox'].includes(route.name)" />
  <router-view />
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const type = computed(() => route.params.type)
const teamID = computed(() => route.params.teamID)
const viewID = computed(() => route.params.viewID)

const conversationStore = useConversationStore()

// Views that don't use status filtering (server-side filtered)
const NO_STATUS_VIEWS = ['spam', 'trash']

/**
 * Apply filters for a view: restore from localStorage or use sane defaults.
 * Spam/Trash/Views = no status filter. Everything else = ['Open'] default.
 */
function applyFiltersForView (listType, tID, vID) {
  // Save current view's filters before switching
  conversationStore.saveViewFilters()

  if (NO_STATUS_VIEWS.includes(listType)) {
    // Spam/Trash: clear status filter, clear ad-hoc filters
    conversationStore.conversations.status = []
    conversationStore.conversations.adHocFilters = []
    return
  }

  if (vID) {
    // Custom views: no status filter (server handles it)
    conversationStore.conversations.status = []
    conversationStore.conversations.adHocFilters = []
    return
  }

  // Try restoring saved filters for this view
  const restored = conversationStore.restoreViewFilters(listType, tID, vID)
  if (!restored) {
    // Sane defaults: Open status, newest sort, no ad-hoc filters
    conversationStore.conversations.status = [CONVERSATION_DEFAULT_STATUSES.OPEN]
    conversationStore.conversations.adHocFilters = []
  }
}

// Init conversations list based on route params
onMounted(() => {
  if (type.value) {
    applyFiltersForView(type.value, 0, 0)
    conversationStore.fetchConversationsList(true, type.value)
  }
  if (teamID.value) {
    applyFiltersForView(CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, teamID.value, 0)
    conversationStore.fetchConversationsList(
      true,
      CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
      teamID.value
    )
  }
  if (viewID.value) {
    applyFiltersForView(CONVERSATION_LIST_TYPE.VIEW, 0, viewID.value)
    conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], viewID.value)
  }
})

// Restore filters when returning from a conversation detail view
watch(
  () => route.name,
  (newName, oldName) => {
    // Returning from conversation to list view
    if (oldName && oldName.includes('conversation') && newName && !newName.includes('conversation')) {
      const listType = type.value || 'assigned'
      conversationStore.restoreViewFilters(listType, teamID.value || 0, viewID.value || 0)
    }
  }
)

// Refetch when route params change
watch(
  [type, teamID, viewID],
  ([newType, newTeamID, newViewID], [oldType, oldTeamID, oldViewID]) => {
    if (newType !== oldType && newType) {
      applyFiltersForView(newType, 0, 0)
      conversationStore.fetchConversationsList(true, newType)
    }
    if (newTeamID !== oldTeamID && newTeamID) {
      applyFiltersForView(CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED, newTeamID, 0)
      conversationStore.fetchConversationsList(
        true,
        CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED,
        newTeamID
      )
    }
    if (newViewID !== oldViewID && newViewID) {
      applyFiltersForView(CONVERSATION_LIST_TYPE.VIEW, 0, newViewID)
      conversationStore.fetchConversationsList(true, CONVERSATION_LIST_TYPE.VIEW, 0, [], newViewID)
    }
  }
)
</script>
