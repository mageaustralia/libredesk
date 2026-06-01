// useNewConversationDraft persists the in-progress New Conversation form to
// localStorage so a session-expiry redirect (or accidental tab close) doesn't
// wipe an agent's work. Restores on dialog open, debounced-saves on changes,
// cleared on a successful send.
//
// localStorage scope: per-browser, single draft (the New Conversation dialog
// is a one-at-a-time composer). Multi-user-per-browser would collide; not
// worth a per-user key for v1 — easy to add later if needed.
//
// Attachments are deliberately NOT restored — files are uploaded server-side
// at pick time and may have been reaped by the orphan-cleaner during the
// outage. Storing only the text fields keeps the recovery boundary clear.

import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

const STORAGE_KEY = 'libredesk:newConversationDraft:v1'
const TTL_MS = 14 * 24 * 60 * 60 * 1000 // 14 days — long enough to survive
                                        // a multi-hour session expiry, short
                                        // enough that stale drafts don't accumulate.
const DEBOUNCE_MS = 500

function safeJSON (raw) {
  try { return JSON.parse(raw) } catch { return null }
}

function readStored () {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const data = safeJSON(raw)
    if (!data || !data.savedAt) return null
    if (Date.now() - data.savedAt > TTL_MS) {
      localStorage.removeItem(STORAGE_KEY)
      return null
    }
    return data
  } catch {
    return null
  }
}

function isMeaningful (s) {
  if (!s) return false
  // Strip the editor's "empty" sentinels so an untouched TipTap doc doesn't
  // count as content worth persisting.
  const v = String(s).trim().replace(/<p>\s*<\/p>/gi, '').replace(/<br\s*\/?>/gi, '')
  return v !== ''
}

export function useNewConversationDraft ({
  form,         // vee-validate form (useForm) — required
  dialogOpen,   // ref<boolean> of the dialog open state (restore on open)
  emailQuery,   // ref<string> of the TO chip-input model
  ccEmails,     // ref<string>
  bccEmails,    // ref<string>
  showCc,       // ref<boolean>
  showBcc       // ref<boolean>
}) {
  const hasDraft = ref(false)
  let saveTimer = null
  let restored = false

  function save () {
    const v = form.values || {}
    const meaningful =
      isMeaningful(v.subject) ||
      isMeaningful(v.content) ||
      isMeaningful(emailQuery?.value) ||
      isMeaningful(ccEmails?.value) ||
      isMeaningful(bccEmails?.value)
    if (!meaningful) {
      try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
      hasDraft.value = false
      return
    }
    const payload = {
      savedAt: Date.now(),
      form: {
        contact_email: v.contact_email || '',
        first_name: v.first_name || '',
        last_name: v.last_name || '',
        subject: v.subject || '',
        content: v.content || '',
        inbox_id: v.inbox_id || '',
        team_id: v.team_id || '',
        agent_id: v.agent_id || ''
      },
      emailQuery: emailQuery?.value || '',
      cc: ccEmails?.value || '',
      bcc: bccEmails?.value || '',
      showCc: !!showCc?.value,
      showBcc: !!showBcc?.value
    }
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
      hasDraft.value = true
    } catch { /* quota / disabled / private mode — silently skip */ }
  }

  function scheduleSave () {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(save, DEBOUNCE_MS)
  }

  function restore () {
    const data = readStored()
    if (!data) return false
    const f = data.form || {}
    for (const [k, val] of Object.entries(f)) {
      if (val !== '' && val != null) form.setFieldValue(k, val)
    }
    if (emailQuery && data.emailQuery) emailQuery.value = data.emailQuery
    if (ccEmails && data.cc) ccEmails.value = data.cc
    if (bccEmails && data.bcc) bccEmails.value = data.bcc
    if (showCc) showCc.value = data.showCc || !!data.cc
    if (showBcc) showBcc.value = data.showBcc || !!data.bcc
    hasDraft.value = true
    return true
  }

  function clear () {
    if (saveTimer) { clearTimeout(saveTimer); saveTimer = null }
    try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
    hasDraft.value = false
  }

  // Save on any form/field change (debounced).
  watch(() => form.values, scheduleSave, { deep: true })
  if (emailQuery) watch(emailQuery, scheduleSave)
  if (ccEmails) watch(ccEmails, scheduleSave)
  if (bccEmails) watch(bccEmails, scheduleSave)
  if (showCc) watch(showCc, scheduleSave)
  if (showBcc) watch(showBcc, scheduleSave)

  // Restore each time the dialog opens. Guard so we don't re-restore during a
  // single open session (would overwrite the user's edits with stale values
  // if they had emptied a field — empty values aren't re-applied by restore,
  // but the flag avoids the no-op work entirely).
  if (dialogOpen) {
    watch(dialogOpen, (open) => {
      if (open && !restored) {
        restored = restore()
      } else if (!open) {
        restored = false
      }
    }, { immediate: true })
  } else {
    onMounted(() => { restored = restore() })
  }

  onBeforeUnmount(() => { if (saveTimer) clearTimeout(saveTimer) })

  return { hasDraft, clear, save, restore }
}
