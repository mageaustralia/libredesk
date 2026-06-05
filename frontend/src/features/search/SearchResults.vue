<template>
  <div class="max-w-5xl mx-auto p-6 min-h-screen">
    <p class="text-sm text-muted-foreground mb-4">{{ props.total }} results</p>
    <div class="bg-background rounded border overflow-hidden">
      <div class="divide-y divide-border">
        <div
          v-for="item in results"
          :key="item.uuid"
          class="p-6 hover:bg-accent/50 transition duration-200 ease-in-out group"
        >
          <router-link
            :to="{
              name: 'inbox-conversation',
              params: { uuid: item.uuid, type: 'assigned' }
            }"
            class="block"
          >
            <div class="flex justify-between items-start">
              <div class="flex-grow">
                <div
                  class="text-sm font-semibold mb-1 text-muted-foreground group-hover:text-primary transition duration-200"
                >
                  #{{ item.reference_number }}
                </div>
                <div
                  class="text-foreground font-medium mb-2 text-lg group-hover:text-primary transition duration-200"
                >
                  {{ item.subject || '(no subject)' }}
                </div>
                <div v-if="item.snippet" class="text-sm text-muted-foreground mb-2 line-clamp-2">
                  {{ truncateText(item.snippet, 200) }}
                </div>
                <div class="text-sm text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1">
                  <span class="flex items-center">
                    <CalendarIcon class="h-4 w-4 mr-1" />
                    Created: {{ formatDate(item.created_at) }}
                  </span>
                  <span
                    v-if="item.last_message_at && item.last_message_at !== item.created_at"
                    class="flex items-center"
                  >
                    <ClockIcon class="h-4 w-4 mr-1" />
                    Last activity: {{ formatDate(item.last_message_at) }}
                    <span
                      v-if="item.last_message_sender"
                      class="ml-1.5 text-xs"
                      :class="item.last_message_sender === 'agent' ? 'text-green-600' : 'text-foreground/70'"
                    >
                      ({{ item.last_message_sender === 'agent' ? 'agent' : 'customer' }})
                    </span>
                  </span>
                </div>
              </div>
              <div
                class="bg-secondary rounded-full p-2 group-hover:bg-primary transition duration-200"
              >
                <ChevronRightIcon
                  class="h-5 w-5 text-secondary-foreground group-hover:text-primary-foreground"
                  aria-hidden="true"
                />
              </div>
            </div>
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ChevronRightIcon, ClockIcon, CalendarIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'

const props = defineProps({
  results: {
    type: Array,
    required: true
  },
  total: {
    type: Number,
    default: 0
  }
})

const formatDate = (dateString) => {
  const date = parseISO(dateString)
  return format(date, 'MMM d, yyyy HH:mm')
}

const truncateText = (text, length) => {
  if (!text) return ''
  if (text.length <= length) return text
  return text.slice(0, length) + '...'
}
</script>
