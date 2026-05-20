import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { permissions as p } from '@/constants/permissions'

// PR #286 — bulk-action permission gating composable. Each dropdown in the
// bulk-action toolbar is shown/hidden based on the corresponding
// per-conversation permission. A user with none of these still sees the
// selection count + clear button (they could already select rows; the
// toolbar just collapses to its read-only state).
//
// `canTrash` reuses `conversations:update_status` since moving to trash is
// a status transition under the hood — same permission that gates
// `bulkUpdateStatus`. Kept as a separate computed so the toolbar can hide
// the trash button independently if we ever split the permission.
export function useBulkActionPermissions () {
  const userStore = useUserStore()

  const canAssignAgent = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_USER_ASSIGNEE))
  const canAssignTeam = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE))
  const canUpdateStatus = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_STATUS))
  const canUpdateTags = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_TAGS))
  const canTrash = computed(() => userStore.can(p.CONVERSATIONS_UPDATE_STATUS))

  const canBulkAct = computed(
    () =>
      canAssignAgent.value ||
      canAssignTeam.value ||
      canUpdateStatus.value ||
      canUpdateTags.value ||
      canTrash.value
  )

  return { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags, canTrash, canBulkAct }
}
