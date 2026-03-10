<template>
  <div class="space-y-2">
    <div class="flex justify-between items-start">
      <Avatar class="size-20">
        <AvatarImage :src="conversation?.contact?.avatar_url || ''" />
        <AvatarFallback>
          {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7"
        @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
      >
        <ViewVerticalIcon />
      </Button>
    </div>

    <div class="flex items-center gap-2">
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-24 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </span>
      <ExternalLink
        v-if="!conversationStore.conversation.loading && userStore.can('contacts:read')"
        size="16"
        class="text-muted-foreground cursor-pointer flex-shrink-0"
        @click="$router.push({ name: 'contact-detail', params: { id: conversation?.contact_id } })"
      />
      <button
        v-if="!conversationStore.conversation.loading"
        class="text-muted-foreground hover:text-foreground flex-shrink-0"
        @click="showContactSearch = true"
        title="Change contact"
      >
        <ArrowRightLeft :size="14" />
      </button>
    </div>

    <!-- Contact change search -->
    <div v-if="showContactSearch" class="mt-2 space-y-2">
      <div class="relative">
        <input
          ref="contactSearchInput"
          v-model="contactQuery"
          @input="searchContacts"
          placeholder="Search by name or email..."
          class="w-full text-sm border rounded px-2 py-1.5 pr-12"
        />
        <div class="absolute right-1 top-1/2 -translate-y-1/2 flex items-center gap-0.5">
          <button
            class="text-muted-foreground hover:text-foreground p-0.5"
            @click="showCreateForm = true"
            title="Create new contact"
          >
            <Plus :size="14" />
          </button>
          <button
            class="text-muted-foreground hover:text-foreground p-0.5"
            @click="closeSearch"
          >
            <X :size="14" />
          </button>
        </div>
      </div>
      <div v-if="contactResults.length" class="border rounded max-h-40 overflow-y-auto">
        <button
          v-for="c in contactResults"
          :key="c.id"
          class="w-full text-left px-2 py-1.5 text-sm hover:bg-muted flex flex-col"
          @click="changeContact(c)"
        >
          <span>{{ c.first_name }} {{ c.last_name }}</span>
          <span class="text-xs text-muted-foreground">{{ c.email }}</span>
        </button>
      </div>
      <p v-if="contactQuery.length >= 2 && !contactResults.length && !searching && !showCreateForm" class="text-xs text-muted-foreground px-1">
        No contacts found — click <Plus :size="12" class="inline" /> to create
      </p>

      <!-- Quick create contact form -->
      <div v-if="showCreateForm" class="border rounded p-2 space-y-2 bg-muted/30">
        <p class="text-xs font-medium">Create new contact</p>
        <input
          v-model="newContact.email"
          placeholder="Email *"
          class="w-full text-sm border rounded px-2 py-1"
          @keyup.escape="showCreateForm = false"
        />
        <div class="flex gap-1">
          <input
            v-model="newContact.first_name"
            placeholder="First name"
            class="w-1/2 text-sm border rounded px-2 py-1"
          />
          <input
            v-model="newContact.last_name"
            placeholder="Last name"
            class="w-1/2 text-sm border rounded px-2 py-1"
          />
        </div>
        <div class="flex gap-1">
          <Button size="sm" class="h-7 text-xs" @click="createAndAssign" :disabled="!newContact.email">
            Create & set as contact
          </Button>
          <Button size="sm" variant="ghost" class="h-7 text-xs" @click="showCreateForm = false">
            Cancel
          </Button>
        </div>
      </div>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Mail size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else class="break-all">
        {{ conversation?.contact?.email }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Phone size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ phoneNumber }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, nextTick, watch } from 'vue'
import { ViewVerticalIcon } from '@radix-icons/vue'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Mail, Phone, ExternalLink, ArrowRightLeft, X, Plus } from 'lucide-vue-next'
import countries from '@/constants/countries.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@/utils/http'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()
const userStore = useUserStore()

const showContactSearch = ref(false)
const contactQuery = ref('')
const contactResults = ref([])
const contactSearchInput = ref(null)
const searching = ref(false)
let searchTimeout = null
const showCreateForm = ref(false)
const newContact = ref({ email: '', first_name: '', last_name: '' })

const closeSearch = () => {
  showContactSearch.value = false
  showCreateForm.value = false
  contactQuery.value = ''
  contactResults.value = []
  newContact.value = { email: '', first_name: '', last_name: '' }
}

const createAndAssign = async () => {
  if (!newContact.value.email) return
  try {
    const res = await api.quickCreateContact(newContact.value)
    const created = res.data?.data
    if (created) {
      await changeContact(created)
      showCreateForm.value = false
      newContact.value = { email: '', first_name: '', last_name: '' }
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

watch(showContactSearch, (val) => {
  if (val) nextTick(() => contactSearchInput.value?.focus())
})

const searchContacts = () => {
  clearTimeout(searchTimeout)
  if (contactQuery.value.length < 2) {
    contactResults.value = []
    return
  }
  searching.value = true
  searchTimeout = setTimeout(async () => {
    try {
      const res = await api.searchContacts({ query: contactQuery.value })
      contactResults.value = res.data?.data || []
    } catch (e) {
      contactResults.value = []
    }
    searching.value = false
  }, 300)
}

const changeContact = async (contact) => {
  try {
    await api.updateConversationContact(conversation.value.uuid, contact.id)
    // Update local state
    conversationStore.current.contact_id = contact.id
    conversationStore.current.contact = {
      ...conversationStore.current.contact,
      id: contact.id,
      first_name: contact.first_name,
      last_name: contact.last_name,
      email: contact.email,
      avatar_url: contact.avatar_url,
      phone_number: contact.phone_number,
      phone_number_country_code: contact.phone_number_country_code,
    }
    showContactSearch.value = false
    contactQuery.value = ''
    contactResults.value = []
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Contact updated' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const phoneNumber = computed(() => {
  const countryCodeValue = conversation.value?.contact?.phone_number_country_code || ''
  const number = conversation.value?.contact?.phone_number || t('conversation.sidebar.notAvailable')
  if (!countryCodeValue) return number

  // Lookup calling code
  const country = countries.find((c) => c.iso_2 === countryCodeValue)
  const callingCode = country ? country.calling_code : countryCodeValue
  return `${callingCode} ${number}`
})
</script>
