<template>
  <div class="max-w-5xl mx-auto p-6 min-h-screen flex flex-col gap-6">
    <!-- Contacts -->
    <section
      v-if="showSection('contacts') && grouped.contacts.results.length"
      :style="{ order: sectionOrder('contacts') }"
    >
      <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-2">
        {{ $t('search.scope.contacts') }} ({{ grouped.contacts.total }})
      </h3>
      <div class="bg-background rounded border overflow-hidden divide-y divide-border">
        <router-link
          v-for="c in grouped.contacts.results"
          :key="'c' + c.id"
          :to="{ name: 'contact-detail', params: { id: c.id } }"
          class="flex items-center gap-3 p-4 hover:bg-accent/50 transition group"
        >
          <div class="bg-secondary rounded-full p-2">
            <UserIcon class="h-4 w-4 text-secondary-foreground" />
          </div>
          <div>
            <div class="font-medium group-hover:text-primary transition">
              {{ (c.first_name + ' ' + (c.last_name || '')).trim() }}
            </div>
            <div class="text-sm text-muted-foreground">{{ c.email }}</div>
          </div>
        </router-link>
      </div>
      <ViewAllButton
        v-if="scope === 'all' && grouped.contacts.total > grouped.contacts.results.length"
        :count="grouped.contacts.total"
        @click="$emit('set-scope', 'contacts')"
      />
    </section>

    <!-- Conversations -->
    <section
      v-if="showSection('conversations') && grouped.conversations.results.length"
      :style="{ order: sectionOrder('conversations') }"
    >
      <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-2">
        {{ $t('search.scope.conversations') }} ({{ grouped.conversations.total }})
      </h3>
      <div class="bg-background rounded border overflow-hidden divide-y divide-border">
        <router-link
          v-for="item in grouped.conversations.results"
          :key="item.uuid"
          :to="{ name: 'inbox-conversation', params: { uuid: item.uuid, type: 'assigned' } }"
          class="block p-4 hover:bg-accent/50 transition group"
        >
          <div class="text-sm font-semibold mb-0.5 text-muted-foreground group-hover:text-primary transition">
            #{{ item.reference_number }}
            <span class="font-normal">— {{ item.contact_name }}</span>
            <span v-if="item.contact_email" class="font-normal text-xs">&lt;{{ item.contact_email }}&gt;</span>
          </div>
          <div class="text-foreground font-medium group-hover:text-primary transition">
            {{ item.subject || '(no subject)' }}
          </div>
          <div class="text-sm text-muted-foreground mt-1 flex flex-wrap items-center gap-x-4 gap-y-1">
            <span class="flex items-center">
              <CalendarIcon class="h-4 w-4 mr-1" /> Created: {{ formatDate(item.created_at) }}
            </span>
            <span v-if="item.last_message_at && item.last_message_at !== item.created_at" class="flex items-center">
              <ClockIcon class="h-4 w-4 mr-1" /> Last activity: {{ formatDate(item.last_message_at) }}
              <span
                v-if="item.last_message_sender"
                class="ml-1.5 text-xs"
                :class="item.last_message_sender === 'agent' ? 'text-green-600' : 'text-foreground/70'"
              >
                ({{ item.last_message_sender === 'agent' ? 'agent' : 'customer' }})
              </span>
            </span>
          </div>
        </router-link>
      </div>
      <ViewAllButton
        v-if="scope === 'all' && grouped.conversations.total > grouped.conversations.results.length"
        :count="grouped.conversations.total"
        @click="$emit('set-scope', 'conversations')"
      />
    </section>

    <!-- Messages -->
    <section
      v-if="showSection('messages') && grouped.messages.results.length"
      :style="{ order: sectionOrder('messages') }"
    >
      <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-2">
        {{ $t('search.scope.messages') }} ({{ grouped.messages.total }})
      </h3>
      <div class="bg-background rounded border overflow-hidden divide-y divide-border">
        <router-link
          v-for="item in grouped.messages.results"
          :key="'m' + item.uuid"
          :to="{ name: 'inbox-conversation', params: { uuid: item.uuid, type: 'assigned' } }"
          class="block p-4 hover:bg-accent/50 transition group"
        >
          <div class="text-sm font-semibold mb-0.5 text-muted-foreground group-hover:text-primary transition">
            #{{ item.reference_number }}
          </div>
          <div class="text-foreground font-medium group-hover:text-primary transition">
            {{ item.subject || '(no subject)' }}
          </div>
          <div class="text-sm text-muted-foreground mt-1 line-clamp-2">
            <template v-for="(part, i) in snippetParts(item.snippet)" :key="i">
              <mark v-if="part.hit" class="bg-yellow-200 dark:bg-yellow-700/60 rounded px-0.5 text-inherit">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
          <div class="text-xs text-muted-foreground mt-1 flex items-center">
            <CalendarIcon class="h-3.5 w-3.5 mr-1" /> {{ formatDate(item.created_at) }}
          </div>
        </router-link>
      </div>
      <ViewAllButton
        v-if="scope === 'all' && grouped.messages.total > grouped.messages.results.length"
        :count="grouped.messages.total"
        @click="$emit('set-scope', 'messages')"
      />
    </section>
  </div>
</template>

<script setup>
import { h } from 'vue'
import { useI18n } from 'vue-i18n'
import { ClockIcon, CalendarIcon, UserIcon, ChevronDownIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'
import { Button } from '@/components/ui/button'

const props = defineProps({
  grouped: { type: Object, required: true },
  scope: { type: String, default: 'all' },
  query: { type: String, default: '' }
})
defineEmits(['set-scope'])

const { t } = useI18n()

const showSection = (key) => props.scope === 'all' || props.scope === key

// For email-like queries, sections whose results contain the query literally
// jump above sections that only fuzzy-matched. An email pasted into search is
// an exact-lookup intent: a message that actually contains the address beats
// contacts/conversations surfaced by name similarity. Name queries keep the
// fixed contacts > conversations > messages order — fuzzy name matches are
// usually the person being looked for.
const BASE_ORDER = { contacts: 1, conversations: 2, messages: 3 }

const sectionHasLiteralHit = (key) => {
  const q = props.query.trim().toLowerCase()
  if (!q) return false
  const results = props.grouped[key]?.results || []
  if (key === 'contacts')
    return results.some(
      (c) =>
        (c.email || '').toLowerCase().includes(q) ||
        `${c.first_name} ${c.last_name || ''}`.toLowerCase().includes(q)
    )
  if (key === 'conversations')
    return results.some(
      (c) =>
        String(c.reference_number) === q ||
        (c.contact_email || '').toLowerCase().includes(q) ||
        (c.contact_name || '').toLowerCase().includes(q) ||
        (c.subject || '').toLowerCase().includes(q)
    )
  // Messages are full-text hits; a highlight marker means the query text was
  // genuinely found in the body.
  return results.some((m) => (m.snippet || '').includes('[[['))
}

const sectionOrder = (key) => {
  if (props.scope !== 'all' || !props.query.includes('@')) return BASE_ORDER[key]
  return sectionHasLiteralHit(key) ? BASE_ORDER[key] : BASE_ORDER[key] + 10
}

const formatDate = (dateString) => format(parseISO(dateString), 'MMM d, yyyy HH:mm')

// Snippets arrive with [[[ ]]] highlight delimiters from ts_headline.
// Split-and-render keeps everything as text nodes — no v-html, no XSS surface.
const snippetParts = (snippet) =>
  (snippet || '')
    .split(/\[\[\[|\]\]\]/)
    .map((text, i) => ({ text, hit: i % 2 === 1 }))
    .filter((p) => p.text.length > 0)

// Small inline functional component for the per-section "View all N" affordance.
const ViewAllButton = (viewProps, { emit }) =>
  h(
    Button,
    { variant: 'ghost', size: 'sm', class: 'mt-1 text-xs', onClick: () => emit('click') },
    () => [t('search.viewAll', { count: viewProps.count }), h(ChevronDownIcon, { class: 'h-3.5 w-3.5 ml-1' })]
  )
ViewAllButton.props = { count: { type: Number, required: true } }
ViewAllButton.emits = ['click']
</script>
