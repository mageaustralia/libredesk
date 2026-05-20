<template>
  <div
    role="toolbar"
    :aria-label="t('conversation.bulkActions.toolbar')"
    class="p-2 flex items-center gap-1 bg-muted/30"
  >
    <Checkbox
      :checked="conversationStore.allSelected"
      @update:checked="toggleSelectAll"
      :aria-label="t('conversation.bulkActions.selectAll')"
      class="ml-1 mr-1"
    />
    <span
      class="text-xs font-medium whitespace-nowrap tabular-nums inline-block min-w-20 mr-1"
      aria-live="polite"
    >
      {{
        t(
          'conversation.bulkActions.selected',
          conversationStore.selectedCount,
          { count: conversationStore.selectedCount }
        )
      }}
    </span>

    <!-- Assign Agent -->
    <SelectComboBox
      v-if="canAssignAgent"
      :items="agentItems"
      :placeholder="t('placeholders.selectAgent')"
      type="user"
      align="start"
      @select="(item) => onAssigneeSelect('user', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.assignAgent')"
          :aria-label="t('actions.assignAgent')"
        >
          <UserPlus class="w-4 h-4" />
        </Button>
      </template>
    </SelectComboBox>

    <!-- Assign Team -->
    <SelectComboBox
      v-if="canAssignTeam"
      :items="teamItems"
      :placeholder="t('placeholders.selectTeam')"
      type="team"
      align="start"
      @select="(item) => onAssigneeSelect('team', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.assignTeam')"
          :aria-label="t('actions.assignTeam')"
        >
          <Users class="w-4 h-4" />
        </Button>
      </template>
    </SelectComboBox>

    <!-- Add Tag -->
    <SelectComboBox
      v-if="canUpdateTags"
      :items="tagItems"
      :placeholder="t('placeholders.selectTags')"
      align="start"
      @select="(item) => onTagSelect(TAG_ACTION.ADD, item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.addTags')"
          :aria-label="t('actions.addTags')"
        >
          <Tag class="w-4 h-4" />
        </Button>
      </template>
    </SelectComboBox>

    <!-- Status -->
    <DropdownMenu v-if="canUpdateStatus">
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.setStatus')"
          :aria-label="t('actions.setStatus')"
        >
          <CircleDot class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem
          v-for="status in conversationStore.statusOptionsNoSnooze"
          :key="status.value"
          @click="bulkUpdateStatus(status.label)"
        >
          {{ status.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <!-- Priority — kept from our v2 tree; not in upstream PR #286. -->
    <DropdownMenu v-if="canUpdateStatus">
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="t('actions.setPriority')"
          :aria-label="t('actions.setPriority')"
        >
          <Flag class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem
          v-for="priority in conversationStore.priorityOptions"
          :key="priority.value"
          @click="bulkUpdatePriority(priority.label)"
        >
          {{ priority.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <!-- Move to Trash — hidden in Trash and Spam views. -->
    <Button
      v-if="canTrash && !isTrashView && !isSpamView"
      variant="ghost"
      size="icon"
      :disabled="bulkLoading"
      :title="t('conversation.trash')"
      :aria-label="t('conversation.trash')"
      @click="bulkMoveToTrash"
    >
      <Trash2 class="w-4 h-4" />
    </Button>

    <!-- Permanent delete — Trash view only. FS13: opens an AlertDialog so
         the agent must actively confirm the destructive action; deleted
         rows have no recovery path. -->
    <Button
      v-if="canTrash && isTrashView"
      variant="ghost"
      size="icon"
      :disabled="bulkLoading"
      :title="t('conversation.bulkActions.deletePermanently')"
      :aria-label="t('conversation.bulkActions.deletePermanently')"
      class="text-destructive hover:text-destructive"
      @click="deleteConfirmOpen = true"
    >
      <Trash2 class="w-4 h-4" />
    </Button>

    <Loader2 v-if="bulkLoading" class="w-4 h-4 animate-spin text-muted-foreground ml-2" />

    <Button
      variant="ghost"
      size="icon"
      class="ml-auto"
      :aria-label="t('conversation.bulkActions.clearSelection')"
      @click="conversationStore.clearSelection()"
    >
      <X class="w-4 h-4" />
    </Button>

    <AlertDialog v-model:open="deleteConfirmOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ t('conversation.bulkActions.deletePermanently') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{
              t(
                'conversation.bulkActions.deletePermanentlyConfirmation',
                conversationStore.selectedCount,
                { count: conversationStore.selectedCount }
              )
            }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-destructive-foreground" @click="bulkDeletePermanently">
            {{ t('conversation.bulkActions.deletePermanently') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { UserPlus, Users, Tag, CircleDot, Flag, Trash2, Loader2, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import { TAG_ACTION, CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import api from '@/api'

const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const route = useRoute()
const { t } = useI18n()
const emitter = useEmitter()
const bulkLoading = ref(false)
const deleteConfirmOpen = ref(false)

const { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags, canTrash } =
  useBulkActionPermissions()

const isTrashView = computed(() => route.params.type === CONVERSATION_LIST_TYPE.TRASH)
const isSpamView = computed(() => route.params.type === CONVERSATION_LIST_TYPE.SPAM)

onMounted(() => {
  if (canAssignAgent.value) usersStore.fetchUsers()
  if (canAssignTeam.value) teamsStore.fetchTeams()
  if (canUpdateTags.value) tagStore.fetchTags()
})

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

const withNoneOption = (options) => [
  { value: 'none', label: t('globals.terms.none') },
  ...options
]

const agentItems = computed(() => withNoneOption(usersStore.options))
const teamItems = computed(() => withNoneOption(teamsStore.options))
const tagItems = computed(() => tagStore.tagNames.map((name) => ({ label: name, value: name })))

// runBulkAction — fans out per-conversation API calls in parallel via
// Promise.allSettled. PR #286's pattern, augmented with our richer toast
// behaviour: success toast shows the count, failure toast shows the split.
const runBulkAction = async (actionFn, { resetBeforeFetch = false } = {}) => {
  const uuids = [...conversationStore.selectedUUIDs]
  const total = uuids.length
  bulkLoading.value = true
  const results = await Promise.allSettled(uuids.map((uuid) => actionFn(uuid)))
  bulkLoading.value = false

  const successCount = results.filter((r) => r.status === 'fulfilled').length
  const failed = total - successCount

  if (failed > 0) {
    const failures = results
      .map((r, i) => ({ uuid: uuids[i], reason: r.reason }))
      .filter((f) => f.reason)
    if (failures.length) console.warn('Bulk action failures:', failures)
  }

  conversationStore.clearSelection()
  // FS13: permanent-delete needs a reset-then-refetch because the list
  // merger appends rather than prunes. Other actions just refetch page 1.
  if (resetBeforeFetch) conversationStore.resetConversations()
  await conversationStore.fetchFirstPageConversations()

  if (failed > 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      title: t('globals.terms.error', 1),
      description: t('conversation.bulkActions.failedToast', {
        success: successCount,
        failed,
        total
      })
    })
  } else {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.bulkActions.successToast', successCount, {
        count: successCount
      })
    })
  }
}

const onAssigneeSelect = (assigneeType, item) => {
  if (item.value === 'none') {
    runBulkAction((uuid) => api.removeAssignee(uuid, assigneeType))
    return
  }
  const assigneeId = parseInt(item.value, 10)
  runBulkAction((uuid) => api.updateAssignee(uuid, assigneeType, { assignee_id: assigneeId }))
}

const onTagSelect = (action, item) => {
  runBulkAction((uuid) => conversationStore.updateConversationTags(uuid, action, [item.value]))
}

const bulkUpdateStatus = (status) => {
  runBulkAction((uuid) => api.updateConversationStatus(uuid, { status }))
}

const bulkUpdatePriority = (priority) => {
  runBulkAction((uuid) => api.updateConversationPriority(uuid, { priority }))
}

const bulkMoveToTrash = () => {
  runBulkAction((uuid) => api.moveToTrash(uuid))
}

const bulkDeletePermanently = async () => {
  deleteConfirmOpen.value = false
  await runBulkAction((uuid) => api.deleteConversationPermanently(uuid), {
    resetBeforeFetch: true
  })
}
</script>
