# Tier 2A — Send-flow Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the send-flow cluster of v1.0.3 ReplyBox enhancements onto v2.1.1-plus-enhancements: Send & Set Status dropdown, Send-fail toast + restore, Undo send with countdown, Sticky subject header + inline edit, Fullscreen reply editor.

**Architecture:** Each theme is one feature unit (one logical PR), bundling its initial commit with any later fixes. Themes are ordered roughly chronologically per upstream commit history. Each lands on `v2.1.1-plus-enhancements`.

**Tech Stack:** Vue 3 (`frontend/apps/main/src/`), `@shared-ui/components`, Go backend (`internal/...`), `i18n/en.json` ASCII-sorted.

**Source branch:** `v1.0.3-plus-enhancements` on `ubuntu@54.66.177.54:/home/ubuntu/libredesk` — inspect commits via `ssh ubuntu@54.66.177.54 'cd /home/ubuntu/libredesk && git show <hash>'`.

**Per-theme execution loop (apply to every task):**

1. **Audit source** — `ssh ubuntu@54.66.177.54 'cd /home/ubuntu/libredesk && git log --all --grep="<keywords>"'` to find the initial commit + all follow-up fix commits for this feature. List the bundle.
2. **Read prod source** — `git show <hash>` on each commit in the bundle. Note diff shape and intent.
3. **Read v2 destination files** — identify adaptation deltas: file path differences (`frontend/src/` → `frontend/apps/main/src/`), import path differences (`@/components/...` → `@shared-ui/components/...`), API surface differences (`handleHTTPError(err).message` from `@shared-ui/utils/http.js`, `emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {...})`).
4. **Apply changes** — port each commit in the bundle, adapting per v2 conventions.
5. **Build + test** — `go build ./...` (must succeed), `go test ./<changed packages>/...`, `pnpm --dir frontend type-check` if frontend touched.
6. **Spec reviewer** — dispatch a code-reviewer subagent reviewing the diff against spec/plan. Flag any drift.
7. **Code quality reviewer** — second pass focusing on code quality, test coverage, edge cases.
8. **Apply must-fix items inline** — Critical = always fix before commit, Important = fix unless deferred-with-rationale in commit message.
9. **i18n alphabetisation** — if `i18n/en.json` was touched, verify ASCII order using `LC_ALL=C sort -c <(jq -r 'keys[]' i18n/en.json)`. ASCII order means uppercase before lowercase.
10. **Commit** — full why-context message. No Co-Authored-By trailers (per memory).
11. **Update spec** — change theme status from `pending` to `done <commit-hash>` in `docs/superpowers/specs/2026-04-27-v103-port-design.md` Section 5.2.
12. **Smoke test** — start dev server (`pnpm --dir frontend dev`), exercise the feature path in browser, confirm no regression in unaffected paths.
13. **Mark task complete** — TaskUpdate, then proceed to next theme.

---

## Task T2a: Send & Set Status dropdown (EC1 + EC17)

**Source commits:** `693974ee` (initial), plus EC17 dedup-guard from `fdfcf50c`. Run audit step 1 to confirm bundle.

**Behaviour:** Reply box gets a chevron button next to Send that opens a dropdown of status options ("Send & Resolve", "Send & Close", "Send & Snooze"). Picking one sends the reply AND transitions the conversation status in a single action. The "delete draft" button is also surfaced. EC17 ensures the dedup guard at `cmd/messages.go:checkMessageDedup` doesn't fire if the second click was via a different status path (so one Send and one Send & Resolve aren't conflated).

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue`
- Modify: `frontend/apps/main/src/features/conversation/ReplyBoxMenuBar.vue` (or equivalent — verify in v2)
- Modify: `frontend/apps/main/src/api/index.js` — add status query param to message send call
- Modify: `cmd/messages.go` — accept optional `set_status` field on message POST, transition status post-send
- Modify: `internal/conversation/conversation.go` (or models) — single-action status transition method if needed
- Modify: `i18n/en.json`

**Key adaptation deltas (v2):**
- TipTap reply box state lives in v2 `ReplyBox.vue` differently — verify the editor ref pattern
- Status enum in v2 uses category-based lookup (post-2.2.0); ensure status names map to the right backend status_id
- Dedup map in `cmd/messages.go:checkMessageDedup` — the dedup key is `(user_id, conv_uuid, content)`. EC17's fix is to include the action variant in the key

- [ ] **Step 1:** Audit source bundle (loop step 1)
- [ ] **Step 2:** Read prod source for each commit (loop step 2)
- [ ] **Step 3:** Read v2 destination files; list adaptation deltas (loop step 3)
- [ ] **Step 4:** Port the changes (loop step 4)
- [ ] **Step 5:** Build + test (loop step 5)
- [ ] **Step 6:** Spec reviewer (loop step 6)
- [ ] **Step 7:** Code quality reviewer (loop step 7)
- [ ] **Step 8:** Apply must-fix items (loop step 8)
- [ ] **Step 9:** i18n ASCII order check if applicable (loop step 9)
- [ ] **Step 10:** Commit (loop step 10)
- [ ] **Step 11:** Update spec status (loop step 11)
- [ ] **Step 12:** Smoke test (loop step 12)
- [ ] **Step 13:** Mark task complete (loop step 13)

---

## Task T2b: Send-fail error toast + restore editor content (EC2)

**Source commit:** `d94cd572`.

**Behaviour:** When the message send POST fails (HTTP 5xx, timeout, network error), show a destructive toast with the error message AND restore the editor content the user typed (currently it gets cleared on submit and the user loses their work). Critical UX for the "failed to send" case we already partially addressed at the timeout layer.

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — wrap send call in try/catch, snapshot editor content before send, restore on catch
- Modify: `i18n/en.json` — error toast string

**Key adaptation deltas (v2):**
- v2 uses `handleHTTPError(err).message` from `@shared-ui/utils/http.js` for HTTP error message extraction
- Toast surface via `emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description })`

- [ ] **Step 1-13:** Same loop pattern as T2a

---

## Task T2c: Undo send with countdown + draft clearing on conv switch (EC3 + EC18)

**Source commits:** subset of `fdfcf50c` (initial undo), plus EC18 draft-clearing fix.

**Behaviour:** After clicking Send, instead of immediately dispatching the SMTP send, show a "Sending in 5s — Undo" countdown banner. If the user clicks Undo within the window, the message stays as a draft and the editor restores. If the countdown expires, the actual send fires. Implementation lives entirely client-side — backend POST only happens after the countdown completes. EC18 is the related fix for draft state leaking when user switches conversations during the countdown.

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — countdown timer state, banner component, dispatch deferral
- Modify: `frontend/apps/main/src/stores/conversation.js` — draft state isolation per conv
- New (maybe): `frontend/apps/main/src/features/conversation/UndoSendBanner.vue` — extract banner if it grows

**Key adaptation deltas (v2):**
- Pinia store pattern in v2 — verify `conversation.js` store has draft slice
- Watch for `currentConvUUID` changes to cancel countdown (EC18's bug was countdown firing on stale conv after switch)

- [ ] **Step 1-13:** Same loop pattern

---

## Task T2d: Sticky subject header + inline subject editing (EC4 + EC5)

**Source commits:** subset of `fdfcf50c` (sticky header), `3de2f5bd` (inline editing).

**Behaviour:** Subject line stays visible at the top of the conversation as you scroll the message list (sticky positioning). Click the subject to edit it inline — pressing Enter or clicking outside saves; pressing Escape cancels. Backend gets a PATCH endpoint or extends an existing one to update the conversation subject.

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/Conversation.vue` (or sticky-header component) — add sticky positioning + click-to-edit
- New (maybe): `frontend/apps/main/src/features/conversation/SubjectHeader.vue` — extract if standalone
- Modify: `frontend/apps/main/src/api/index.js` — add `updateConversationSubject(uuid, subject)`
- Modify: `cmd/conversation.go` — handler for subject update
- Modify: `internal/conversation/queries.sql` + `conversation.go` — `UpdateConversationSubject` method
- Modify: `i18n/en.json`

**Key adaptation deltas (v2):**
- v2 may already have sticky header markup — verify before adding
- Conversation update broadcasts via WS — include subject change

- [ ] **Step 1-13:** Same loop pattern

---

## Task T2e: Fullscreen reply editor + sidebar toggle (EC6)

**Source commit:** `5097e66e`.

**Behaviour:** Reply box gets a "fullscreen" toggle button. When activated, the editor expands to fill the viewport (overlay or full-area resize), hiding the conversation list and sidebar for distraction-free composition. Toggle off restores normal layout. State should NOT persist across reloads (per-session only).

**Files (likely):**
- Modify: `frontend/apps/main/src/features/conversation/ReplyBox.vue` — fullscreen ref + toggle button
- Modify: `frontend/apps/main/src/features/conversation/Conversation.vue` (or layout parent) — react to fullscreen state, hide sidebar/list
- Modify: `i18n/en.json` — toggle button label

**Key adaptation deltas (v2):**
- v2 layout uses different parent component for sidebar/list — verify the right place to gate visibility
- TipTap editor may need `class="prose-lg"` or similar at fullscreen for readable typography

- [ ] **Step 1-13:** Same loop pattern

---

## Stop / handoff protocol

After T2e (or any earlier theme that completes a session boundary):
1. `git push origin v2.1.1-plus-enhancements`
2. Checkpoint to shared memory:
   ```bash
   curl -s -X POST https://mem.mageaustralia.com.au/memory \
     -H "Authorization: Bearer USW8pKBPUTu6Y21a7wSWajB6YgufSw2ARGqG1qDnzx4" \
     -H "Content-Type: application/json" \
     -d '{"content":"v103-port: Tier 2A complete (T2a-T2e send-flow). Next: Tier 2B (composer recipient UX EC7-EC10, editor fixes EC11-EC12, From switcher EC13-EC14, macro/defaults EC15-EC16, private note signature EC19).", "source":"checkpoint-libredesk-port", "author":"claude-code"}'
   ```
3. Notify Matthew that Tier 2A is done and ask whether to continue with Tier 2B.

---

## Notes / risks

- **EC1 + EC17 interaction with our recent dedup-window bump (10s → 60s):** the longer window makes dedup-bypass for status variants MORE important. Without EC17 a user clicking Send then Send&Resolve within 60s gets the second click rejected.
- **EC3 (Undo send) interaction with backend dedup:** the countdown-then-send pattern means the actual POST fires `5s + processing` after the click. Backend dedup window should still cover this — verify in audit step.
- **EC4 / EC5 sticky header:** v2 already has different conversation header markup vs v1.0.3 — may not be a clean port. Audit thoroughly.
