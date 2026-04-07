<template>
  <Sheet :open="open" @update:open="$emit('update:open', $event)">
    <SheetContent side="right" class="w-80 sm:max-w-80 p-0 flex flex-col filter-panel-sheet">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b shrink-0">
        <h3 class="text-sm font-semibold">Filters</h3>
        <Button variant="ghost" size="sm" class="h-7 text-xs mr-6" @click="clearAll">
          Clear
        </Button>
      </div>

      <!-- Scrollable body -->
      <div class="flex-1 overflow-y-auto px-4 py-3 space-y-4">

        <!-- Contact Email -->
        <div>
          <label class="text-xs font-medium text-muted-foreground mb-1 block">Contact Email</label>
          <input
            v-model="contactEmail"
            type="text"
            placeholder="Search by email..."
            class="w-full h-9 px-3 text-sm border rounded-md bg-transparent outline-none focus:ring-1 focus:ring-ring hover:border-ring transition-colors"
            @keydown.enter.prevent="applyAdHocFilters()"
            @blur="applyAdHocFilters()"
          />
        </div>

        <!-- Status (hidden for spam/trash — server-filtered) -->
        <FilterDropdown v-if="!isServerFilteredView" ref="statusRef" label="Status" :summary="statusSummary" placeholder="Any status"
          :open="statusDropOpen" @toggle="statusDropOpen = !statusDropOpen">
          <label class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer hover:bg-accent border-b">
            <Checkbox :checked="isAllUnresolved" @update:checked="toggleAllUnresolved" />
            <span class="font-medium">All Unresolved</span>
          </label>
          <label
            v-for="status in conversationStore.statusOptions"
            :key="status.value"
            class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer hover:bg-accent"
          >
            <Checkbox
              :checked="conversationStore.conversations.status.includes(status.label)"
              @update:checked="handleStatusToggle(status.label)"
            />
            {{ status.label }}
          </label>
        </FilterDropdown>

        <!-- Agent -->
        <FilterDropdown ref="agentRef" :label="'Agents ' + (agentMode === 'include' ? 'Include' : 'Exclude')"
          :summary="agentSummary" placeholder="Any agent" :toggleLabel="true"
          :open="agentDropOpen" @toggle="agentDropOpen = !agentDropOpen" @toggleMode="toggleAgentMode">
          <div class="p-2 border-b">
            <input v-model="agentSearch" type="text" placeholder="Search..."
              class="w-full h-7 px-2 text-sm border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring"
              ref="agentSearchInput" />
          </div>
          <div class="overflow-y-auto">
            <button v-if="!agentSearch"
              @click="toggleAgent('unassigned')"
              class="w-full text-left px-3 py-2 text-sm hover:bg-accent flex items-center justify-between cursor-pointer border-b"
              :class="{ 'bg-accent/50': selectedAgents.includes('unassigned') }">
              <span class="truncate text-muted-foreground italic">Unassigned</span>
              <Check v-if="selectedAgents.includes('unassigned')" class="w-3.5 h-3.5 text-primary shrink-0" />
            </button>
            <button v-for="agent in filteredAgents" :key="agent.value"
              @click="toggleAgent(String(agent.value))"
              class="w-full text-left px-3 py-2 text-sm hover:bg-accent flex items-center justify-between cursor-pointer"
              :class="{ 'bg-accent/50': selectedAgents.includes(String(agent.value)) }">
              <span class="truncate">{{ agent.label }}</span>
              <Check v-if="selectedAgents.includes(String(agent.value))" class="w-3.5 h-3.5 text-primary shrink-0" />
            </button>
            <div v-if="filteredAgents.length === 0" class="py-3 text-center text-xs text-muted-foreground">No agents found</div>
          </div>
        </FilterDropdown>

        <!-- Team -->
        <FilterDropdown ref="teamRef" :label="'Groups ' + (teamMode === 'include' ? 'Include' : 'Exclude')"
          :summary="teamSummary" placeholder="Any group" :toggleLabel="true"
          :open="teamDropOpen" @toggle="teamDropOpen = !teamDropOpen" @toggleMode="toggleTeamMode">
          <div class="p-2 border-b">
            <input v-model="teamSearch" type="text" placeholder="Search..."
              class="w-full h-7 px-2 text-sm border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring" />
          </div>
          <div class="overflow-y-auto">
            <button v-for="team in filteredTeams" :key="team.value"
              @click="toggleTeam(String(team.value))"
              class="w-full text-left px-3 py-2 text-sm hover:bg-accent flex items-center justify-between cursor-pointer"
              :class="{ 'bg-accent/50': selectedTeams.includes(String(team.value)) }">
              <span class="truncate">{{ team.label }}</span>
              <Check v-if="selectedTeams.includes(String(team.value))" class="w-3.5 h-3.5 text-primary shrink-0" />
            </button>
            <div v-if="filteredTeams.length === 0" class="py-3 text-center text-xs text-muted-foreground">No groups found</div>
          </div>
        </FilterDropdown>

        <!-- Priority -->
        <FilterDropdown ref="priorityRef" label="Priority" :summary="prioritySummary" placeholder="Any priority"
          :open="priorityDropOpen" @toggle="priorityDropOpen = !priorityDropOpen">
          <button v-for="priority in conversationStore.priorityOptions" :key="priority.value"
            @click="togglePriority(String(priority.value))"
            class="w-full text-left px-3 py-2 text-sm hover:bg-accent flex items-center justify-between cursor-pointer"
            :class="{ 'bg-accent/50': selectedPriorities.includes(String(priority.value)) }">
            {{ priority.label }}
            <Check v-if="selectedPriorities.includes(String(priority.value))" class="w-3.5 h-3.5 text-primary shrink-0" />
          </button>
        </FilterDropdown>

        <!-- Tags -->
        <FilterDropdown ref="tagsRef" label="Tags" :summary="tagsSummary" placeholder="Any tag"
          :open="tagsDropOpen" @toggle="tagsDropOpen = !tagsDropOpen">
          <div class="p-2 border-b">
            <input v-model="tagsSearch" type="text" placeholder="Search..."
              class="w-full h-7 px-2 text-sm border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring" />
          </div>
          <div class="overflow-y-auto">
            <button v-for="tag in filteredTags" :key="tag.value"
              @click="toggleTag(String(tag.value))"
              class="w-full text-left px-3 py-2 text-sm hover:bg-accent flex items-center justify-between cursor-pointer"
              :class="{ 'bg-accent/50': selectedTags.includes(String(tag.value)) }">
              <span class="truncate">{{ tag.label }}</span>
              <Check v-if="selectedTags.includes(String(tag.value))" class="w-3.5 h-3.5 text-primary shrink-0" />
            </button>
            <div v-if="filteredTags.length === 0" class="py-3 text-center text-xs text-muted-foreground">No tags found</div>
          </div>
        </FilterDropdown>

        <!-- Date filters -->
        <FilterDateDropdown ref="createdRef" label="Created" v-model="dateCreated"
          :open="createdDropOpen" @toggle="createdDropOpen = !createdDropOpen" @change="applyAdHocFilters()" />

        <FilterDateDropdown ref="lastActivityRef" label="Last activity" v-model="dateLastActivity"
          :open="lastActivityDropOpen" @toggle="lastActivityDropOpen = !lastActivityDropOpen" @change="applyAdHocFilters()" />

        <FilterDateDropdown ref="closedRef" label="Closed at" v-model="dateClosedAt"
          :open="closedDropOpen" @toggle="closedDropOpen = !closedDropOpen" @change="applyAdHocFilters()" />

        <FilterDateDropdown ref="resolvedRef" label="Resolved at" v-model="dateResolvedAt"
          :open="resolvedDropOpen" @toggle="resolvedDropOpen = !resolvedDropOpen" @change="applyAdHocFilters()" />

        <FilterDateDropdown ref="slaRef" label="SLA deadline" v-model="dateSLA"
          :open="slaDropOpen" @toggle="slaDropOpen = !slaDropOpen" @change="applyAdHocFilters()" />

      </div>

      <!-- Active filter count footer -->
      <div v-if="activeFilterCount > 0" class="px-4 py-2 border-t shrink-0 bg-muted/30">
        <p class="text-xs text-muted-foreground">
          {{ activeFilterCount }} filter{{ activeFilterCount === 1 ? '' : 's' }} active
        </p>
      </div>
    </SheetContent>
  </Sheet>
</template>

<!-- Inline sub-components to keep things in one file -->
<script>
import { h, defineComponent } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

// Reusable combobox-style dropdown
const FilterDropdown = defineComponent({
  name: 'FilterDropdown',
  props: {
    label: String,
    summary: String,
    placeholder: String,
    open: Boolean,
    toggleLabel: Boolean
  },
  emits: ['toggle', 'toggleMode'],
  setup(props, { slots, emit }) {
    return () => h('div', {}, [
      // Label
      props.toggleLabel
        ? h('button', {
            class: 'text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-0.5 mb-1',
            onClick: () => emit('toggleMode')
          }, [props.label, h(ChevronDown, { class: 'w-3 h-3 opacity-50' })])
        : h('label', { class: 'text-xs font-medium text-muted-foreground mb-1 block' }, props.label),
      // Input button + dropdown
      h('div', { class: 'relative' }, [
        h('button', {
          class: 'w-full h-9 px-3 text-sm border rounded-md bg-transparent flex items-center justify-between cursor-pointer hover:border-ring transition-colors',
          onClick: () => emit('toggle')
        }, [
          h('span', { class: props.summary === props.placeholder ? 'text-muted-foreground' : '' }, props.summary),
          h(ChevronDown, { class: ['w-3.5 h-3.5 opacity-50 shrink-0 transition-transform', props.open ? 'rotate-180' : ''] })
        ]),
        props.open
          ? h('div', { class: 'absolute z-50 w-full mt-1 border rounded-md bg-popover shadow-md max-h-64 overflow-y-auto' },
              slots.default?.())
          : null
      ])
    ])
  }
})

// Date preset dropdown
const datePresets = [
  { label: 'Any time', value: '' },
  { label: 'Today', value: 'today' },
  { label: 'Yesterday', value: 'yesterday' },
  { label: 'Last 7 days', value: 'last_7_days' },
  { label: 'Last 30 days', value: 'last_30_days' },
  { label: 'This month', value: 'this_month' },
]

const FilterDateDropdown = defineComponent({
  name: 'FilterDateDropdown',
  props: {
    label: String,
    modelValue: String,
    open: Boolean
  },
  emits: ['update:modelValue', 'toggle', 'change'],
  setup(props, { emit }) {
    const summary = () => {
      if (!props.modelValue) return 'Any time'
      const preset = datePresets.find(p => p.value === props.modelValue)
      return preset ? preset.label : props.modelValue
    }

    return () => h('div', {}, [
      h('label', { class: 'text-xs font-medium text-muted-foreground mb-1 block' }, props.label),
      h('div', { class: 'relative' }, [
        h('button', {
          class: 'w-full h-9 px-3 text-sm border rounded-md bg-transparent flex items-center justify-between cursor-pointer hover:border-ring transition-colors',
          onClick: () => emit('toggle')
        }, [
          h('span', { class: !props.modelValue ? 'text-muted-foreground' : '' }, summary()),
          h(ChevronDown, { class: ['w-3.5 h-3.5 opacity-50 shrink-0 transition-transform', props.open ? 'rotate-180' : ''] })
        ]),
        props.open
          ? h('div', { class: 'absolute z-50 w-full mt-1 border rounded-md bg-popover shadow-md overflow-y-auto' },
              datePresets.map(preset =>
                h('button', {
                  key: preset.value,
                  class: ['w-full text-left px-3 py-2 text-sm hover:bg-accent cursor-pointer',
                    props.modelValue === preset.value ? 'bg-accent/50 font-medium' : ''].join(' '),
                  onClick: () => {
                    emit('update:modelValue', preset.value)
                    emit('toggle')
                    emit('change')
                  }
                }, preset.label)
              )
            )
          : null
      ])
    ])
  }
})

export { FilterDropdown, FilterDateDropdown }
</script>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { Checkbox } from '@/components/ui/checkbox'
import { Button } from '@/components/ui/button'
import { ChevronDown, Check } from 'lucide-vue-next'

const props = defineProps({
  open: Boolean,
  viewType: { type: String, default: '' }
})
const emit = defineEmits(['update:open'])

const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const route = useRoute()
const emitter = useEmitter()

// Views where status is server-controlled
const NO_STATUS_VIEWS = ['spam', 'trash']
const isServerFilteredView = computed(() => NO_STATUS_VIEWS.includes(props.viewType))

// Fetch tags on mount
onMounted(() => { tagStore.fetchTags() })

// --- Dropdown open states ---
const statusDropOpen = ref(false)
const agentDropOpen = ref(false)
const teamDropOpen = ref(false)
const priorityDropOpen = ref(false)
const tagsDropOpen = ref(false)
const createdDropOpen = ref(false)
const lastActivityDropOpen = ref(false)
const closedDropOpen = ref(false)
const resolvedDropOpen = ref(false)
const slaDropOpen = ref(false)

// --- Refs for click-outside ---
const statusRef = ref(null)
const agentRef = ref(null)
const teamRef = ref(null)
const priorityRef = ref(null)
const tagsRef = ref(null)
const createdRef = ref(null)
const lastActivityRef = ref(null)
const closedRef = ref(null)
const resolvedRef = ref(null)
const slaRef = ref(null)
const agentSearchInput = ref(null)

function handleClickOutside(e) {
  const refs = [
    [statusRef, statusDropOpen], [agentRef, agentDropOpen], [teamRef, teamDropOpen],
    [priorityRef, priorityDropOpen], [tagsRef, tagsDropOpen],
    [createdRef, createdDropOpen], [lastActivityRef, lastActivityDropOpen],
    [closedRef, closedDropOpen], [resolvedRef, resolvedDropOpen], [slaRef, slaDropOpen]
  ]
  for (const [refEl, openState] of refs) {
    const el = refEl.value?.$el || refEl.value
    if (el && !el.contains(e.target)) openState.value = false
  }
}

onMounted(() => document.addEventListener('mousedown', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('mousedown', handleClickOutside))

watch(agentDropOpen, (open) => {
  if (open) nextTick(() => agentSearchInput.value?.focus())
})

// --- Status ---
const statusSummary = computed(() => {
  const s = conversationStore.conversations.status
  if (s.length === 0) return 'Any status'
  if (s.length === 1) return s[0]
  const resolvedNames = ['Resolved', 'Closed', 'Trashed', 'Spam']
  if (!s.some(n => resolvedNames.includes(n))) return 'All Unresolved'
  return s.length + ' statuses'
})

const isAllUnresolved = computed(() => {
  const current = conversationStore.conversations.status
  const resolvedNames = ['Resolved', 'Closed', 'Trashed', 'Spam']
  return current.length > 1 && !current.some(s => resolvedNames.includes(s))
})

function toggleAllUnresolved(checked) {
  if (checked) {
    const resolvedNames = ['Resolved', 'Closed', 'Trashed', 'Spam']
    const all = conversationStore.statusOptions.map(s => s.label).filter(name => !resolvedNames.includes(name))
    conversationStore.setListStatus(all)
  } else {
    conversationStore.setListStatus(['Open'])
  }
}

function handleStatusToggle(statusName) {
  conversationStore.toggleListStatus(statusName)
}

// --- Agent ---
const agentSearch = ref('')
const agentMode = ref('include')
const selectedAgents = ref([])

function toggleAgentMode() {
  agentMode.value = agentMode.value === 'include' ? 'exclude' : 'include'
  if (selectedAgents.value.length > 0) applyAdHocFilters()
}

const agentSummary = computed(() => {
  if (selectedAgents.value.length === 0) return 'Any agent'
  if (selectedAgents.value.length === 1) {
    if (selectedAgents.value[0] === 'unassigned') return 'Unassigned'
    const opt = (usersStore.options || []).find(o => String(o.value) === selectedAgents.value[0])
    return opt ? opt.label : selectedAgents.value[0]
  }
  return selectedAgents.value.length + ' agents'
})

const filteredAgents = computed(() => {
  const options = usersStore.options || []
  if (!agentSearch.value) return options
  const q = agentSearch.value.toLowerCase()
  return options.filter(a => a.label.toLowerCase().includes(q))
})

function toggleAgent(id) {
  const idx = selectedAgents.value.indexOf(id)
  if (idx >= 0) selectedAgents.value.splice(idx, 1)
  else selectedAgents.value.push(id)
  applyAdHocFilters()
}

// --- Team ---
const teamSearch = ref('')
const teamMode = ref('include')
const selectedTeams = ref([])

function toggleTeamMode() {
  teamMode.value = teamMode.value === 'include' ? 'exclude' : 'include'
  if (selectedTeams.value.length > 0) applyAdHocFilters()
}

const teamSummary = computed(() => {
  if (selectedTeams.value.length === 0) return 'Any group'
  if (selectedTeams.value.length === 1) {
    const opt = (teamsStore.options || []).find(o => String(o.value) === selectedTeams.value[0])
    return opt ? opt.label : selectedTeams.value[0]
  }
  return selectedTeams.value.length + ' groups'
})

const filteredTeams = computed(() => {
  const options = teamsStore.options || []
  if (!teamSearch.value) return options
  const q = teamSearch.value.toLowerCase()
  return options.filter(t => t.label.toLowerCase().includes(q))
})

function toggleTeam(id) {
  const idx = selectedTeams.value.indexOf(id)
  if (idx >= 0) selectedTeams.value.splice(idx, 1)
  else selectedTeams.value.push(id)
  applyAdHocFilters()
}

// --- Priority ---
const selectedPriorities = ref([])

const prioritySummary = computed(() => {
  if (selectedPriorities.value.length === 0) return 'Any priority'
  if (selectedPriorities.value.length === 1) {
    const opt = conversationStore.priorityOptions.find(o => String(o.value) === selectedPriorities.value[0])
    return opt ? opt.label : selectedPriorities.value[0]
  }
  return selectedPriorities.value.length + ' priorities'
})

function togglePriority(id) {
  const idx = selectedPriorities.value.indexOf(id)
  if (idx >= 0) selectedPriorities.value.splice(idx, 1)
  else selectedPriorities.value.push(id)
  applyAdHocFilters()
}

// --- Tags ---
const tagsSearch = ref('')
const selectedTags = ref([])

const tagsSummary = computed(() => {
  if (selectedTags.value.length === 0) return 'Any tag'
  if (selectedTags.value.length === 1) {
    const opt = (tagStore.tagOptions || []).find(o => String(o.value) === selectedTags.value[0])
    return opt ? opt.label : selectedTags.value[0]
  }
  return selectedTags.value.length + ' tags'
})

const filteredTags = computed(() => {
  const options = tagStore.tagOptions || []
  if (!tagsSearch.value) return options
  const q = tagsSearch.value.toLowerCase()
  return options.filter(t => t.label.toLowerCase().includes(q))
})

function toggleTag(id) {
  const idx = selectedTags.value.indexOf(id)
  if (idx >= 0) selectedTags.value.splice(idx, 1)
  else selectedTags.value.push(id)
  applyAdHocFilters()
}

// --- Contact email ---
const contactEmail = ref('')

// --- Date filters ---
const dateCreated = ref('')
const dateLastActivity = ref('')
const dateClosedAt = ref('')
const dateResolvedAt = ref('')
const dateSLA = ref('')

// --- Build and apply ad-hoc filters ---
function applyAdHocFilters() {
  const filters = []

  if (selectedAgents.value.length > 0) {
    const hasUnassigned = selectedAgents.value.includes('unassigned')
    const agentIds = selectedAgents.value.filter(id => id !== 'unassigned')
    if (hasUnassigned && agentIds.length === 0) {
      // Only "Unassigned" selected
      filters.push({ field: 'assigned_user_id', operator: agentMode.value === 'include' ? 'not set' : 'set',
        value: '', model: 'conversations' })
    } else if (hasUnassigned && agentIds.length > 0) {
      // "Unassigned" + specific agents
      filters.push({ field: 'assigned_user_id', operator: agentMode.value === 'include' ? 'in_or_null' : 'not_in',
        value: JSON.stringify(agentIds), model: 'conversations' })
    } else {
      // Only specific agents
      filters.push({ field: 'assigned_user_id', operator: agentMode.value === 'include' ? 'in' : 'not_in',
        value: JSON.stringify(agentIds), model: 'conversations' })
    }
  }
  if (selectedTeams.value.length > 0) {
    filters.push({ field: 'assigned_team_id', operator: teamMode.value === 'include' ? 'in' : 'not_in',
      value: JSON.stringify(selectedTeams.value), model: 'conversations' })
  }
  if (selectedPriorities.value.length > 0) {
    filters.push({ field: 'priority_id', operator: 'in',
      value: JSON.stringify(selectedPriorities.value), model: 'conversations' })
  }
  if (selectedTags.value.length > 0) {
    filters.push({ field: 'tags', operator: 'contains',
      value: JSON.stringify(selectedTags.value), model: 'conversations' })
  }
  if (contactEmail.value.trim()) {
    filters.push({ field: 'email', operator: 'ilike', value: contactEmail.value.trim(), model: 'users' })
  }

  // Date filters
  const dateFields = [
    { ref: dateCreated, field: 'created_at' },
    { ref: dateLastActivity, field: 'last_message_at' },
    { ref: dateClosedAt, field: 'closed_at' },
    { ref: dateResolvedAt, field: 'resolved_at' },
    { ref: dateSLA, field: 'next_sla_deadline_at' },
  ]
  for (const d of dateFields) {
    if (d.ref.value) {
      filters.push({ field: d.field, operator: 'relative_date', value: d.ref.value, model: 'conversations' })
    }
  }

  conversationStore.setAdHocFilters(filters)
}

// --- Active filter count ---
const activeFilterCount = computed(() => {
  let count = 0
  if (selectedAgents.value.length > 0) count++
  if (selectedTeams.value.length > 0) count++
  if (selectedPriorities.value.length > 0) count++
  if (selectedTags.value.length > 0) count++
  if (contactEmail.value.trim()) count++
  if (dateCreated.value) count++
  if (dateLastActivity.value) count++
  if (dateClosedAt.value) count++
  if (dateResolvedAt.value) count++
  if (dateSLA.value) count++
  if (!isServerFilteredView.value) {
    const s = conversationStore.conversations.status
    const resolvedNames = ['Resolved', 'Closed', 'Trashed', 'Spam']
    const isDefault = s.length === 1 && s[0] === 'Open'
    const isAllUnresolved = s.length > 1 && !s.some(n => resolvedNames.includes(n))
    if (!isDefault && !isAllUnresolved) count++
  }
  return count
})

// --- Sync panel state from store ---
function syncFromStore() {
  const adHoc = conversationStore.conversations.adHocFilters || []

  const agentFilter = adHoc.find(f => f.field === 'assigned_user_id')
  if (agentFilter) {
    if (agentFilter.operator === 'not set') {
      agentMode.value = 'include'
      selectedAgents.value = ['unassigned']
    } else if (agentFilter.operator === 'set') {
      agentMode.value = 'exclude'
      selectedAgents.value = ['unassigned']
    } else {
      agentMode.value = agentFilter.operator === 'not_in' ? 'exclude' : 'include'
      const ids = (() => { try { return JSON.parse(agentFilter.value) } catch { return [] } })()
      selectedAgents.value = agentFilter.operator === 'in_or_null' ? [...ids, 'unassigned'] : ids
    }
  } else { agentMode.value = 'include'; selectedAgents.value = [] }

  const teamFilter = adHoc.find(f => f.field === 'assigned_team_id')
  if (teamFilter) {
    teamMode.value = teamFilter.operator === 'not_in' ? 'exclude' : 'include'
    try { selectedTeams.value = JSON.parse(teamFilter.value) } catch { selectedTeams.value = [] }
  } else { teamMode.value = 'include'; selectedTeams.value = [] }

  const priorityFilter = adHoc.find(f => f.field === 'priority_id')
  if (priorityFilter) {
    try { selectedPriorities.value = JSON.parse(priorityFilter.value) } catch { selectedPriorities.value = [] }
  } else { selectedPriorities.value = [] }

  const tagsFilter = adHoc.find(f => f.field === 'tags')
  if (tagsFilter) {
    try { selectedTags.value = JSON.parse(tagsFilter.value) } catch { selectedTags.value = [] }
  } else { selectedTags.value = [] }

  const emailFilter = adHoc.find(f => f.field === 'email')
  contactEmail.value = emailFilter ? emailFilter.value : ''

  // Date filters
  dateCreated.value = adHoc.find(f => f.field === 'created_at')?.value || ''
  dateLastActivity.value = adHoc.find(f => f.field === 'last_message_at')?.value || ''
  dateClosedAt.value = adHoc.find(f => f.field === 'closed_at')?.value || ''
  dateResolvedAt.value = adHoc.find(f => f.field === 'resolved_at')?.value || ''
  dateSLA.value = adHoc.find(f => f.field === 'next_sla_deadline_at')?.value || ''

  // Close all dropdowns
  statusDropOpen.value = false
  agentDropOpen.value = false
  teamDropOpen.value = false
  priorityDropOpen.value = false
  tagsDropOpen.value = false
  createdDropOpen.value = false
  lastActivityDropOpen.value = false
  closedDropOpen.value = false
  resolvedDropOpen.value = false
  slaDropOpen.value = false
  agentSearch.value = ''
  teamSearch.value = ''
  tagsSearch.value = ''
}

watch(() => props.open, (isOpen) => {
  if (isOpen) syncFromStore()
})

// --- Clear all ---
function clearAll() {
  selectedAgents.value = []
  selectedTeams.value = []
  selectedPriorities.value = []
  selectedTags.value = []
  contactEmail.value = ''
  dateCreated.value = ''
  dateLastActivity.value = ''
  dateClosedAt.value = ''
  dateResolvedAt.value = ''
  dateSLA.value = ''
  agentMode.value = 'include'
  teamMode.value = 'include'
  agentSearch.value = ''
  teamSearch.value = ''
  tagsSearch.value = ''
  const resolvedNames = ['Resolved', 'Closed', 'Trashed', 'Spam']
  const allUnresolved = conversationStore.statusOptions.map(s => s.label).filter(name => !resolvedNames.includes(name))
  conversationStore.setListStatus(allUnresolved.length > 0 ? allUnresolved : ['Open'])
  conversationStore.setAdHocFilters([])
}
</script>

<style>
.filter-panel-sheet + [data-radix-overlay],
.filter-panel-sheet ~ [data-radix-overlay] {
  background: transparent !important;
}
</style>
