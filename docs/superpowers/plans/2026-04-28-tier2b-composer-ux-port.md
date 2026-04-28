# Tier 2B — Email Composer UX Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Port the remaining Section 5.2 email-composer UX features from v1.0.3 onto v2.1.1-plus-enhancements, picking up where Tier 2A left off (`cfd085d6`).

**Architecture:** Same per-theme structure as Tier 2A. Each theme is one logical PR-equivalent commit (with follow-up fix commits if needed). Themes ordered by feature cohesion rather than strict source-commit chronology.

**Tech Stack:** Vue 3 (`frontend/apps/main/src/`), `@shared-ui/components`, Go backend, `i18n/en.json` ASCII-sorted.

**Source branch:** `v1.0.3-plus-enhancements` on `ubuntu@54.66.177.54:/home/ubuntu/libredesk` — inspect via `ssh ubuntu@54.66.177.54 'cd /home/ubuntu/libredesk && git show <hash>'`.

**Per-theme execution loop:** Same 13 steps as the Tier 2A plan (`docs/superpowers/plans/2026-04-28-tier2a-send-flow-port.md`). Audit → read prod → read v2 → adapt → build/test → review (controller) → fix-loop → i18n check → commit → spec status update → smoke (deferred) → mark complete.

---

## Task T2f: Composer recipient UX bundle (EC7 + EC8 + EC9 + EC10)

Cohesive bundle — all touch the recipient input row(s) and editor focus behaviour in the reply box.

**Source commits:**
- EC7 — `9bcae060` Gmail-style TO/CC/BCC composer + scroll-to-last
- EC8 — `17579887` Per-email remove buttons on TO/CC/BCC
- EC9 — `224c1668` Clear (X) button on CC/BCC fields (verify what's already in v2 from the inline-image work)
- EC10 — `0124d3a1` Cursor position to start of editor on new conv/reply

**Behaviour:**
- TO/CC/BCC inputs render added emails as Gmail-style chips
- Each chip has its own remove (X) icon
- CC/BCC fields have a clear-all (X) button when populated
- Adding an email auto-scrolls so the input cursor stays in view
- New conversation or reply puts the cursor at the START of the editor (not end)

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBoxContent.vue` — recipient inputs are here
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — editor focus on mount/conv-switch
- Modify: `i18n/en.json` — clear button labels, remove-email titles
- New (maybe): `frontend/apps/main/src/features/conversation/EmailChipsInput.vue` — extract chip-input if it grows

**Adaptation deltas:**
- v2 may already have part of EC9 (CC/BCC clear-X) from prior inline-image work — verify
- Check that the editor's existing focus logic from T2c (conv-switch flush + focus()) is compatible with EC10's cursor-to-start

- [ ] Step 1-13: Same execution loop pattern as Tier 2A

---

## Task T2g: TipTap editor fixes (EC11 + EC12)

**Source commits:**
- EC11 — `0934c57b` Paste/drop in TipTap editor fix
- EC12 — `93766088` Exit list on double-Enter + formatting toolbar

**Behaviour:**
- Pasting / dropping content into TipTap behaves correctly (no broken HTML, no duplicate content, no lost formatting)
- Double-Enter inside a bullet/numbered list exits the list (Word-like behaviour)
- Reply box gets a formatting toolbar (bold/italic/list/link) — verify whether v2 already has one and if so, what's missing

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/TextEditor.vue` (or wherever the TipTap instance lives)
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — toolbar surface

**Adaptation deltas:**
- v2's TipTap version may differ from v1.0.3 — check extension API compatibility
- Toolbar may already exist partially from upstream — port only the missing bits

- [ ] Step 1-13: Same execution loop

---

## Task T2h: Agent name in From header + From switcher (EC13 + EC14)

EC14 depends on per-inbox aliases (MP4 in spec section 5.5). If MP4 isn't ported yet, do EC13 alone and DEFER EC14 — note in commit body.

**Source commits:**
- EC13 — `fd21e08b` Agent name in email From header
- EC14 — subset of `97533510` From switcher in reply (per-inbox aliases)

**Behaviour:**
- EC13: Outgoing emails set From header to "Agent Name <inbox@domain>" instead of just "<inbox@domain>"
- EC14: Reply box has a From dropdown letting agent pick which inbox alias to send from (e.g. orders@ vs support@)

**Files (likely):**
- Modify: `internal/inbox/channel/email/smtp.go` — From header construction
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — From switcher (EC14 only)
- Modify: `internal/inbox/models/models.go` — alias model (EC14 only, depends on MP4)

**Adaptation deltas:**
- Verify whether v2 already plumbs sender_name through to SMTP From header (some upstream commits may have done this)
- Check MP4 status in spec; if pending, EC14 deferred with rationale

- [ ] Step 1-13: Same execution loop (skip EC14 sub-task if MP4 not ready)

---

## Task T2i: Macro Zap toolbar + smart new-conversation defaults (EC15 + EC16)

**Source commits:**
- EC15 — subset of `97533510` Macro toolbar (Zap) button for quick macro access
- EC16 — subset of `c7b60817` Smart new-conversation defaults

**Behaviour:**
- EC15: Reply toolbar gets a Zap (lightning) button that opens the macro picker without needing Ctrl+K
- EC16: Starting a new conversation pre-fills sensible defaults (assignee = current agent, status = Open, etc.)

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBoxMenuBar.vue` — Zap button
- Modify: `frontend/apps/main/src/features/conversation/NewConversationDialog.vue` (or equivalent) — defaults

**Adaptation deltas:**
- v2's macro picker may have a different invocation API than v1.0.3
- Verify v2 has a "new conversation" dialog and where its initial state comes from

- [ ] Step 1-13: Same execution loop

---

## Task T2j: Private note signature removal (EC19)

EC19 depends on MP1 (per-inbox signatures, spec section 5.5). If MP1 isn't ported yet, **DEFER** the entire theme.

**Source commit:** `a9f77c6d` — Remove signature from private notes, restore on switch to reply.

**Behaviour:** When the agent switches the reply mode to "Private note", any inbox signature in the editor is removed (private notes don't go to the customer, no need for the email signature). When switching back to "Reply", the signature is re-inserted.

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — watch on messageType, signature management

**Adaptation deltas:**
- Verify MP1 status in spec; if pending, defer entire T2j theme

- [ ] Step 1-13: Same execution loop (or skip if MP1 not ready)

---

## Stop / handoff protocol

After each theme commits cleanly, the controller can stop or continue. After all themes complete:
1. `git push origin v2.1.1-plus-enhancements`
2. Checkpoint to shared memory:
   ```bash
   curl -s -X POST https://mem.mageaustralia.com.au/memory \
     -H "Authorization: Bearer USW8pKBPUTu6Y21a7wSWajB6YgufSw2ARGqG1qDnzx4" \
     -H "Content-Type: application/json" \
     -d '{"content":"v103-port: Tier 2B complete (T2f composer recipient UX, T2g TipTap editor fixes, T2h From header+switcher, T2i Macro Zap+new-conv defaults, T2j private note signature). Next: Section 5.3 Email rendering & threading per spec.", "source":"checkpoint-libredesk-port", "author":"claude-code"}'
   ```
3. Notify Matthew that Tier 2B is done.

---

## Notes / risks

- **EC14 + EC19 are gated on Section 5.5 features (MP4, MP1)** — defer cleanly if dependencies aren't ready.
- **TipTap upgrades between v1.0.3 and v2** may mean some EC11/EC12 patches don't apply cleanly. Audit thoroughly.
- **EC9 partial-port risk** — verify what's already in v2 from the inline-image bundle before duplicating.
- **Cumulative complexity** — Tier 2A's final review flagged ReplyBox.vue (894 lines) and Conversation.vue (616 lines) as approaching the maintainability limit. Consider extracting composables (useUndoSendQueue, useSubjectInlineEdit, useCollisionGuard) BEFORE Tier 2B if any T2f-T2j theme would push them past 1000 lines.
