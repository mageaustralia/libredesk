// useInboxSignature wires the New Conversation message body to the selected
// inbox's signature, swapping it cleanly whenever the inbox changes.
//
// Why this is non-trivial: the editor (TipTap StarterKit) doesn't include a
// <div> node, so a wrapper like <div class="email-signature"> gets stripped
// on insert — a marker-based replacement regex then misses on the next
// change. Instead this composable snapshots the editor's actual normalized
// output of the last-inserted signature (after TipTap has had a tick to
// process it) and uses that snapshot for find-and-replace next time. Falls
// back to the class="email-signature" regex (in case a Div extension was
// added later) and finally to "append" if no previous sig can be located.
//
// State is intentionally per-instance: a New Conversation dialog is a
// one-at-a-time composer, so the closure variable is fine.

import { watch, nextTick } from 'vue'

const SIG_WRAPPER_RE = /<div class="email-signature">[\s\S]*?<\/div>/
const SPACER_RE = /<p>\s*<br\s*\/?>\s*<\/p>/g

export function useInboxSignature ({
  form,                          // vee-validate form (required)
  api,                           // api object exposing getInboxSignature(inboxId, convUuid)
  contentField = 'content',
  inboxIdField = 'inbox_id'
}) {
  // The exact HTML the editor showed for the last-inserted signature, AFTER
  // TipTap normalization. Used as the authoritative substring for replacement
  // on the next inbox change.
  let lastSigSnapshot = ''

  async function applyForInbox (inboxId) {
    if (!inboxId) return
    let signature = ''
    try {
      const resp = await api.getInboxSignature(Number(inboxId), '')
      signature = resp.data?.data?.signature || ''
    } catch {
      signature = ''
    }

    const currentContent = form.values[contentField] || ''
    const sigBlock = signature ? '<p><br></p>' + signature : ''

    let newContent
    if (lastSigSnapshot && currentContent.includes(lastSigSnapshot)) {
      // Replace by snapshot (most reliable — uses editor's own output).
      newContent = currentContent.replace(lastSigSnapshot, sigBlock)
    } else if (SIG_WRAPPER_RE.test(currentContent)) {
      // Wrapper survived (e.g., a Div extension was added) — use it.
      newContent = currentContent.replace(SIG_WRAPPER_RE, sigBlock)
    } else {
      const stripped = currentContent.replace(/<[^>]*>/g, '').trim()
      if (!stripped) {
        // Empty editor — seed with the signature.
        newContent = sigBlock
      } else {
        // No previous sig located — append. The previous sig (if any) is left
        // as the user's own content; they can delete it manually.
        newContent = currentContent + sigBlock
      }
    }

    form.setFieldValue(contentField, newContent)

    // Wait for TipTap to normalize, then re-snapshot the actual rendered sig
    // block (find the last empty-paragraph spacer and take everything after).
    await nextTick()
    const finalContent = form.values[contentField] || ''
    if (signature) {
      const matches = [...finalContent.matchAll(SPACER_RE)]
      const lastSpacer = matches[matches.length - 1]
      if (lastSpacer && lastSpacer.index !== undefined) {
        lastSigSnapshot = finalContent.slice(lastSpacer.index)
      } else {
        // No spacer found — fall back to taking the whole content as the
        // snapshot (the prefix was empty, so the editor's output IS the sig).
        lastSigSnapshot = finalContent
      }
    } else {
      lastSigSnapshot = ''
    }
  }

  // Auto-swap whenever the form's inbox_id changes.
  watch(
    () => form.values[inboxIdField],
    (newInboxId) => {
      if (newInboxId) applyForInbox(newInboxId)
    }
  )

  return { applyForInbox }
}
