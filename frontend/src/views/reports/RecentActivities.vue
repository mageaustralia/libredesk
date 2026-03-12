<template>
  <div class="overflow-y-auto">
    <div class="p-6 w-[calc(100%-3rem)]">
      <div class="space-y-4">
        <!-- Activity list -->
        <div v-if="activities.length" class="space-y-0">
          <div
            v-for="activity in activities"
            :key="activity.id"
            class="flex items-start gap-3 py-3 border-b last:border-b-0"
          >
            <!-- Avatar -->
            <div
              class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium text-white shrink-0"
              :style="{ backgroundColor: avatarColor(activity.actor_first_name + activity.actor_last_name) }"
            >
              {{ initials(activity.actor_first_name, activity.actor_last_name) }}
            </div>

            <!-- Content -->
            <div class="min-w-0 flex-1">
              <p class="text-sm">
                <span v-if="activity.type === 'activity'" v-html="activity.content" />
                <span v-else>
                  <span class="font-medium">{{ activity.actor_first_name }} {{ activity.actor_last_name }}</span>
                  sent a response
                </span>
                <router-link
                  :to="conversationLink(activity)"
                  class="text-primary hover:underline ml-1"
                >
                  #{{ activity.reference_number }}
                </router-link>
                <span v-if="activity.subject" class="text-muted-foreground">
                  &mdash; {{ activity.subject }}
                </span>
              </p>
              <p class="text-xs text-muted-foreground mt-0.5">
                {{ formatTime(activity.created_at) }}
              </p>
            </div>
          </div>
        </div>

        <!-- Empty state -->
        <div v-if="!isLoading && !activities.length" class="text-center text-muted-foreground py-12">
          No recent activities
        </div>

        <!-- Loading -->
        <div v-if="isLoading" class="flex justify-center py-8">
          <Spinner />
        </div>

        <!-- Load more -->
        <div v-if="hasMore && !isLoading" class="flex justify-center pt-2 pb-4">
          <Button variant="outline" size="sm" @click="loadMore">
            Load more
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '@/api'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import { Button } from '@/components/ui/button'
import { formatMessageTimestamp } from '@/utils/datetime'

const activities = ref([])
const isLoading = ref(false)
const page = ref(1)
const pageSize = 20
const totalItems = ref(0)

const hasMore = computed(() => activities.value.length < totalItems.value)

const formatTime = (ts) => formatMessageTimestamp(ts)

const initials = (first, last) => {
  return ((first?.[0] || '') + (last?.[0] || '')).toUpperCase() || '?'
}

const avatarColor = (name) => {
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, 45%, 45%)`
}

const conversationLink = (activity) => {
  return `/inboxes/all/conversation/${activity.conversation_uuid}`
}

const fetchActivities = async (append = false) => {
  isLoading.value = true
  try {
    const res = await api.getRecentActivities({ page: page.value, per_page: pageSize })
    const data = res.data?.data
    if (data) {
      if (append) {
        activities.value = [...activities.value, ...data.results]
      } else {
        activities.value = data.results || []
      }
      totalItems.value = data.total || 0
    }
  } catch (err) {
    console.error('Failed to fetch activities', err)
  } finally {
    isLoading.value = false
  }
}

const loadMore = () => {
  page.value++
  fetchActivities(true)
}

onMounted(() => fetchActivities())
</script>
