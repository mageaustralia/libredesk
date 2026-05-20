export const CONVERSATION_LIST_TYPE = {
  ASSIGNED: 'assigned',
  UNASSIGNED: 'unassigned',
  TEAM_UNASSIGNED: 'team_unassigned',
  VIEW: 'view',
  ALL: 'all',
  MENTIONED: 'mentioned',
  SPAM: 'spam',
  TRASH: 'trash'
}

export const CONVERSATION_DEFAULT_STATUSES = {
  OPEN: 'Open',
  SNOOZED: 'Snoozed',
  RESOLVED: 'Resolved',
  CLOSED: 'Closed',
}

export const CONVERSATION_DEFAULT_STATUSES_LIST = Object.values(CONVERSATION_DEFAULT_STATUSES);

export const MACRO_CONTEXT = {
  REPLY: 'reply',
  NEW_CONVERSATION: 'new-conversation'
}

// PR #286 — tag bulk-actions. Backend tagsUpdateReq.Action accepts these
// three values. `set_tags` is the back-compat default when the action
// field is omitted (matches handleUpdateConversationtags's switch).
export const TAG_ACTION = {
  ADD: 'add_tags',
  SET: 'set_tags',
  REMOVE: 'remove_tags'
}