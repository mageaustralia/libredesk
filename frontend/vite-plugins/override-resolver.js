// Magento-style theme overrides for Vite.
//
// Works on RESOLVED paths (not import strings) so every import form —
// `@/foo`, `../foo`, `@shared-ui/foo`, with or without extension —
// flows through the same redirect.
//
// For each configured root <R>, any resolved import under <R>/<x> is
// redirected to <R>/overrides/<x> if that file exists. Files already
// inside <R>/overrides/ are passed through unchanged so they can
// import siblings the normal way without recursion.
//
// Layout:
//   apps/main/src/features/foo.vue          ← upstream, untouched
//   apps/main/src/overrides/features/foo.vue ← our override, takes precedence
//
// To deliberately import the upstream original from inside an override,
// suffix `?upstream` on the import. The plugin recognises this query
// flag and short-circuits the redirect.

import fs from 'node:fs'
import path from 'node:path'

const UPSTREAM_QUERY = '?upstream'

export function overrideResolver({ roots }) {
  const sep = path.sep
  // Normalise once so prefix checks below are simple string comparisons.
  const normalised = roots.map((basePath) => {
    const abs = path.resolve(basePath)
    return {
      basePath: abs,
      overridesDir: path.join(abs, 'overrides'),
    }
  })

  return {
    name: 'libredesk-override-resolver',
    enforce: 'pre',
    async resolveId(source, importer, options) {
      // Honour the explicit upstream escape hatch.
      if (source.endsWith(UPSTREAM_QUERY)) {
        const stripped = source.slice(0, -UPSTREAM_QUERY.length)
        const resolved = await this.resolve(stripped, importer, {
          ...options,
          skipSelf: true,
        })
        return resolved ? resolved.id : null
      }

      // Defer to Vite's own resolver to get an absolute on-disk path.
      const resolved = await this.resolve(source, importer, {
        ...options,
        skipSelf: true,
      })
      if (!resolved || resolved.external) return null

      const id = resolved.id
      for (const { basePath, overridesDir } of normalised) {
        if (!id.startsWith(basePath + sep)) continue
        // Already inside an overrides/ tree — don't recurse on ourselves.
        if (id.startsWith(overridesDir + sep)) return null

        const relPath = path.relative(basePath, id)
        const overridePath = path.join(overridesDir, relPath)
        if (fs.existsSync(overridePath)) {
          return overridePath
        }
      }
      return null
    },
  }
}
