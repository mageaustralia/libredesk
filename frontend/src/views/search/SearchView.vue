<template>
  <div class="flex flex-col h-screen">
    <SearchHeader v-model="searchQuery" @search="handleSearch" />

    <div v-if="searchPerformed || searchQuery.length >= MIN_SEARCH_LENGTH" class="px-6 py-2 border-b space-y-2">
      <div class="flex items-center gap-2 flex-wrap">
        <Button
          v-for="s in SCOPES"
          :key="s"
          :variant="scope === s ? 'default' : 'outline'"
          size="sm"
          class="rounded-full h-7 px-3"
          @click="setScope(s)"
        >
          {{ $t(`search.scope.${s}`) }}
        </Button>
        <div class="flex-1"></div>
        <Select v-model="filters.statusId" @update:modelValue="onFilterChange">
          <SelectTrigger class="h-7 w-40 text-xs">
            <SelectValue :placeholder="$t('search.filter.anyStatus')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="0">{{ $t('search.filter.anyStatus') }}</SelectItem>
            <SelectItem v-for="st in statuses" :key="st.id" :value="String(st.id)">
              {{ st.name }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="filters.inboxId" @update:modelValue="onFilterChange">
          <SelectTrigger class="h-7 w-44 text-xs">
            <SelectValue :placeholder="$t('search.filter.anyInbox')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="0">{{ $t('search.filter.anyInbox') }}</SelectItem>
            <SelectItem v-for="ib in inboxes" :key="ib.id" :value="String(ib.id)">
              {{ ib.name }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="filters.days" @update:modelValue="onFilterChange">
          <SelectTrigger class="h-7 w-36 text-xs">
            <SelectValue :placeholder="$t('search.filter.anyTime')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="0">{{ $t('search.filter.anyTime') }}</SelectItem>
            <SelectItem value="7">{{ $t('search.filter.last7Days') }}</SelectItem>
            <SelectItem value="30">{{ $t('search.filter.last30Days') }}</SelectItem>
            <SelectItem value="90">{{ $t('search.filter.last90Days') }}</SelectItem>
          </SelectContent>
        </Select>
        <Button v-if="hasActiveFilters" variant="ghost" size="sm" class="h-7 text-xs" @click="clearFilters">
          {{ $t('search.clearFilters') }}
        </Button>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto">
      <div v-if="loading && page === 1" class="flex justify-center items-center h-64">
        <Spinner />
      </div>
      <div v-else-if="error" class="mt-8 text-center space-y-4">
        <p class="text-lg text-destructive">{{ error }}</p>
        <Button @click="handleSearch"> {{ $t('globals.terms.tryAgain') }} </Button>
      </div>

      <div v-else>
        <p v-if="searchPerformed && totalAll === 0" class="mt-8 text-center text-muted-foreground">
          {{ $t('search.noResultsForQuery', { query: searchQuery }) }}
        </p>
        <template v-else-if="searchPerformed">
          <SearchResults :grouped="grouped" :scope="scope" @set-scope="setScope" class="h-full" />
          <div v-if="hasMore" class="flex justify-center py-6">
            <Button variant="outline" @click="loadMore" :disabled="loading">
              <Spinner v-if="loading" class="mr-2 h-4 w-4" />
              Load more ({{ scopedShown }} of {{ scopedTotal }})
            </Button>
          </div>
        </template>

        <p
          v-else-if="searchQuery.length > 0 && searchQuery.length < MIN_SEARCH_LENGTH"
          class="mt-8 text-center text-muted-foreground"
        >
          {{ $t('search.minQueryLength', { length: MIN_SEARCH_LENGTH }) }}
        </p>
        <div v-else class="mt-16 text-center">
          <h2 class="text-2xl font-semibold text-primary mb-4">
            {{ $t('globals.messages.search', { name: $t('globals.terms.conversation', 2).toLowerCase() }) }}
          </h2>
          <p class="text-lg text-muted-foreground">
            {{ $t('search.searchBy') }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import SearchHeader from '@/features/search/SearchHeader.vue'
import SearchResults from '@/features/search/SearchResults.vue'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import api from '@/api'

const MIN_SEARCH_LENGTH = 3
const DEBOUNCE_DELAY = 300
const PAGE_SIZE = 30
const SCOPES = ['all', 'contacts', 'conversations', 'messages']

const emptyGrouped = () => ({
  contacts: { results: [], total: 0 },
  conversations: { results: [], total: 0 },
  messages: { results: [], total: 0 }
})

// Restore session state; old flat-array format from the previous search UI is
// discarded so mid-upgrade tabs don't break.
const restoreGrouped = () => {
  try {
    const stored = JSON.parse(sessionStorage.getItem('searchResults') || 'null')
    if (stored && !Array.isArray(stored) && stored.contacts && stored.conversations && stored.messages) {
      return stored
    }
  } catch {
    /* fall through */
  }
  return emptyGrouped()
}

const searchQuery = ref(sessionStorage.getItem('searchQuery') || '')
const grouped = ref(restoreGrouped())
const scope = ref(sessionStorage.getItem('searchScope') || 'all')
const filters = reactive(
  JSON.parse(sessionStorage.getItem('searchFilters') || '{"statusId":"0","inboxId":"0","days":"0"}')
)
const statuses = ref([])
const inboxes = ref([])
const page = ref(1)
const loading = ref(false)
const error = ref(null)
const totalAll = computed(
  () =>
    grouped.value.contacts.total +
    grouped.value.conversations.total +
    grouped.value.messages.total
)
const searchPerformed = ref(searchQuery.value.length >= MIN_SEARCH_LENGTH && totalAll.value > 0)
let debounceTimer = null

const route = useRoute()
watch(
  () => route.fullPath,
  () => {
    if (route.name === 'search' && !sessionStorage.getItem('searchQuery')) {
      searchQuery.value = ''
      grouped.value = emptyGrouped()
      scope.value = 'all'
      searchPerformed.value = false
      page.value = 1
    }
  }
)

onMounted(async () => {
  try {
    const [stResp, ibResp] = await Promise.all([api.getStatuses(), api.getInboxes()])
    statuses.value = stResp.data.data || []
    inboxes.value = ibResp.data.data || []
  } catch {
    // Filter dropdowns degrade to "Any" — search still works.
  }
})

const hasActiveFilters = computed(
  () => filters.statusId !== '0' || filters.inboxId !== '0' || filters.days !== '0'
)

// Pagination only applies to a specific scope; "all" is a capped overview.
const scopedTotal = computed(() => (scope.value === 'all' ? 0 : grouped.value[scope.value].total))
const scopedShown = computed(() =>
  scope.value === 'all' ? 0 : grouped.value[scope.value].results.length
)
const hasMore = computed(() => scope.value !== 'all' && scopedShown.value < scopedTotal.value)

const buildParams = (pageNum) => {
  const params = {
    query: searchQuery.value,
    scope: scope.value,
    page: pageNum,
    page_size: PAGE_SIZE
  }
  if (filters.statusId !== '0') params.status_id = filters.statusId
  if (filters.inboxId !== '0') params.inbox_id = filters.inboxId
  const days = Number(filters.days)
  if (days > 0) params.from_date = new Date(Date.now() - days * 86400000).toISOString()
  return params
}

const persistSession = () => {
  sessionStorage.removeItem('searchResults')
  sessionStorage.setItem('searchQuery', searchQuery.value)
  sessionStorage.setItem('searchScope', scope.value)
  sessionStorage.setItem('searchFilters', JSON.stringify(filters))
  try {
    sessionStorage.setItem('searchResults', JSON.stringify(grouped.value))
  } catch {
    // Results too large for sessionStorage — skip caching.
  }
}

const fetchResults = async (pageNum) => {
  loading.value = true
  error.value = null
  try {
    const resp = await api.searchUnified(buildParams(pageNum))
    const data = resp.data.data || {}
    const incoming = {
      contacts: data.contacts || { results: [], total: 0 },
      conversations: data.conversations || { results: [], total: 0 },
      messages: data.messages || { results: [], total: 0 }
    }
    if (pageNum === 1) {
      grouped.value = incoming
    } else if (scope.value !== 'all') {
      // Append into the active scope's group only.
      grouped.value[scope.value] = {
        results: [...grouped.value[scope.value].results, ...incoming[scope.value].results],
        total: incoming[scope.value].total
      }
    }
    page.value = pageNum
    persistSession()
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}

const handleSearch = async () => {
  if (searchQuery.value.length < MIN_SEARCH_LENGTH) {
    grouped.value = emptyGrouped()
    searchPerformed.value = false
    return
  }
  searchPerformed.value = true
  page.value = 1
  await fetchResults(1)
}

const setScope = (s) => {
  if (scope.value === s) return
  scope.value = s
  page.value = 1
  if (searchQuery.value.length >= MIN_SEARCH_LENGTH) fetchResults(1)
}

const onFilterChange = () => {
  page.value = 1
  if (searchQuery.value.length >= MIN_SEARCH_LENGTH) fetchResults(1)
}

const clearFilters = () => {
  filters.statusId = '0'
  filters.inboxId = '0'
  filters.days = '0'
  onFilterChange()
}

const loadMore = () => {
  fetchResults(page.value + 1)
}

const debouncedSearch = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(handleSearch, DEBOUNCE_DELAY)
}

watch(searchQuery, (newValue) => {
  if (newValue.length >= MIN_SEARCH_LENGTH) {
    debouncedSearch()
  } else {
    clearTimeout(debounceTimer)
    grouped.value = emptyGrouped()
    searchPerformed.value = false
  }
})

onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
})
</script>
