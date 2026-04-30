# Overrides

Magento-style template overrides for the Vue frontend.

## How it works

The `overrideResolver` Vite plugin (see `frontend/vite-plugins/override-resolver.js`)
intercepts every resolved import. For roots configured in `vite.config.js`
(`apps/main/src`, `apps/widget/src`, `shared-ui`), if the resolved path is
`<root>/<x>` and a same-path file exists at `<root>/overrides/<x>`, the
plugin redirects the import to the override.

## Layout

Mirror the upstream tree under `overrides/`. Example:

```
apps/main/src/features/conversation/Conversation.vue          ← upstream
apps/main/src/overrides/features/conversation/Conversation.vue ← override
```

Any import of `@/features/conversation/Conversation.vue` (from anywhere
in the app) silently resolves to the override.

## When to override (vs. editing upstream)

**Override when:**
- The change is large or structural (layout, component composition,
  significant behavioural divergence).
- The file has a high churn rate upstream and merging our edits in via
  rebase has been painful.
- We're going to ship a meaningfully different UX from upstream.

**Just edit upstream when:**
- The change is a tiny bug fix (a few lines, easy to merge).
- The change is something we'd want to send upstream as a PR anyway.
- Overriding the whole file would mean we have to manually port every
  upstream change to it forever — not worth it for trivial diffs.

## Importing the upstream original from inside an override

Append `?upstream` to the import to bypass the override redirect for
that specific import:

```js
// inside overrides/features/foo/Bar.vue
import { somethingFromUpstream } from '@/features/foo/Bar.vue?upstream'
```

This is rarely needed — most overrides simply replace the upstream file
wholesale.

## Tradeoff

Overriding a file means we no longer **automatically** receive upstream
changes to it. When upstream evolves an overridden file we must diff
their new version against ours and merge the relevant bits in. The
trade is: structural diffs survive `git pull` cleanly, at the cost of
manual reconciliation on the few files we override.

Override sparingly.
