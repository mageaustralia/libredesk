#!/usr/bin/env node
// check-i18n.mjs — scan the frontend for t()/$t() / .t() lookups whose key is
// a string literal, and confirm each key exists in i18n/en.json. Catches the
// silent-failure case where a missing key renders as the bare key string at
// runtime.
//
// Limitations:
// - Dynamic keys (variables) are skipped; we can't statically resolve them.
// - We only check existence in en.json. Other locales come via Crowdin so a
//   key missing there is a translation gap, not a code bug, and is out of
//   scope here.
//
// Exit code: 0 if all literal-key references resolve, 1 otherwise.

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..')
const I18N_PATH = path.join(ROOT, 'i18n', 'en.json')
// v1.0.3 has a flat frontend layout (no monorepo).
const SCAN_DIRS = ['frontend/src']
const SCAN_EXTS = new Set(['.vue', '.js', '.ts', '.mjs'])
const SKIP_DIRS = new Set(['node_modules', 'dist', '.git'])

// Matches: t('key'), $t("key"), i18n.t('key'), foo.t('key'). The key char
// class is deliberately conservative — letters/digits/dots/underscores/
// dashes — so hardcoded sentences ("Hello world!") never look like keys.
const CALL_RE = /(?<![A-Za-z0-9_])\$?t\(\s*['"]([A-Za-z][A-Za-z0-9_.-]*)['"]/g

function walk (dir, out = []) {
  let entries
  try { entries = fs.readdirSync(dir, { withFileTypes: true }) } catch { return out }
  for (const e of entries) {
    if (SKIP_DIRS.has(e.name)) continue
    const full = path.join(dir, e.name)
    if (e.isDirectory()) walk(full, out)
    else if (SCAN_EXTS.has(path.extname(e.name))) out.push(full)
  }
  return out
}

function lineOf (source, idx) {
  let line = 1
  for (let i = 0; i < idx && i < source.length; i++) if (source.charCodeAt(i) === 10) line++
  return line
}

function main () {
  if (!fs.existsSync(I18N_PATH)) {
    console.error(`✖ i18n file not found at ${I18N_PATH}`)
    process.exit(2)
  }
  let dict
  try { dict = JSON.parse(fs.readFileSync(I18N_PATH, 'utf8')) }
  catch (err) { console.error(`✖ ${I18N_PATH} is not valid JSON:`, err.message); process.exit(2) }

  const files = SCAN_DIRS.flatMap(d => walk(path.join(ROOT, d)))
  const missing = new Map()

  for (const file of files) {
    const src = fs.readFileSync(file, 'utf8')
    let m
    CALL_RE.lastIndex = 0
    while ((m = CALL_RE.exec(src)) !== null) {
      const key = m[1]
      // Trailing dot means the literal is a prefix being string-concatenated
      // with a variable, e.g. t('list.column.' + col) — the actual lookup
      // key is dynamic and out of scope for this check.
      if (key.endsWith('.')) continue
      if (!(key in dict)) {
        const arr = missing.get(key) || []
        arr.push({ file: path.relative(ROOT, file), line: lineOf(src, m.index) })
        missing.set(key, arr)
      }
    }
  }

  if (missing.size === 0) {
    console.log(`✓ All literal i18n keys resolve (${files.length} files scanned).`)
    process.exit(0)
  }

  console.error(`✖ ${missing.size} i18n key(s) referenced but missing from ${path.relative(ROOT, I18N_PATH)}:\n`)
  for (const [key, locs] of missing) {
    console.error(`  • ${key}`)
    for (const { file, line } of locs.slice(0, 5)) console.error(`      ${file}:${line}`)
    if (locs.length > 5) console.error(`      …and ${locs.length - 5} more`)
  }
  process.exit(1)
}

main()
