# Custom Patches for Libredesk

## Patch Order
Apply patches in numerical order after checking out a new version.

## Current Patches

### 0001-feat-Custom-TW-enhancements.patch (OUTDATED)
**Status:** Needs rebase for v1.0.1+
Contains:
- RAG firstname/lastname placeholders (cmd/rag.go - file structure changed)
- Subject in AI context (ReplyBox.vue)
- Ticket ID in sidebar (superseded by patch 0002)

### 0002-ticket-id-display-fix.patch
**Status:** Partially applied
Contains:
- docker-compose.yml: Use local Dockerfile build
- ConversationSideBarContact.vue: Show only contact name in sidebar

## Manual Changes (Not in Patches)
After applying patches, manually run:
```bash
# Add ticket ID to main header
sed -i "s/return conversation.data?.contact.first_name + ' ' + conversation.data?.contact.last_name$/return conversation.data?.contact.first_name + ' ' + conversation.data?.contact.last_name + (conversation.data?.reference_number ? ' #' + conversation.data.reference_number : '') + (conversation.data?.subject ? ' - ' + conversation.data.subject : '')/" frontend/src/stores/conversation.js
```

## Pending Customizations (Need Rebase)
1. **OpenRouter Support** - Branch: trabulium/feature/openrouter-support
2. **RAG Placeholders** - firstname, lastname, subject in AI prompts

## Upgrade Workflow
1. ./upgrade.sh <version>
2. Apply manual changes from above
3. ./deploy.sh
