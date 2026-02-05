<template>
  <div v-if="!isSearchRoute" class="h-screen w-full flex flex-col">
    <!-- Back button when list is hidden -->
    <div
      v-if="hideListOnTicketOpen && hasConversationOpen"
      class="flex items-center px-3 py-1.5 border-b bg-background shrink-0"
    >
      <button
        @click="goBack"
        class="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft class="h-4 w-4" />
        Back to conversations
      </button>
    </div>

    <ResizablePanelGroup
      direction="horizontal"
      class="flex-1"
      @layout="onLayoutChange"
    >
      <!-- Conversation List Panel -->
      <ResizablePanel
        v-show="showList"
        :default-size="panelSizes[0]"
        :min-size="25"
        :max-size="35"
        class="overflow-y-auto"
      >
        <ConversationList />
      </ResizablePanel>

      <ResizableHandle v-show="showList" />

      <!-- Conversation Detail Panel -->
      <ResizablePanel :default-size="showList ? panelSizes[1] : 100" :min-size="30">
        <router-view v-slot="{ Component }">
          <component :is="Component" />
        </router-view>
      </ResizablePanel>
    </ResizablePanelGroup>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import { useTheme } from '@/composables/useTheme'
import ConversationList from '@/features/conversation/list/ConversationList.vue'
import { ArrowLeft } from 'lucide-vue-next'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@/components/ui/resizable'

const route = useRoute()
const router = useRouter()
const { hideListOnTicketOpen } = useTheme()

const isSearchRoute = computed(() => route.name === 'search')
const hasConversationOpen = computed(() => !!route.params.uuid)
const showList = computed(() => !hideListOnTicketOpen.value || !hasConversationOpen.value)

// Persist panel sizes: [conversationList, conversationDetail]
const panelSizes = useStorage('inboxPanelSizes', [25, 75])

const onLayoutChange = (sizes) => {
  panelSizes.value = sizes
}

function goBack() {
  // Navigate to the parent inbox list (same inbox type, no uuid)
  const routeName = route.name
  if (routeName === 'team-inbox-conversation') {
    router.push({ name: 'team-inbox', params: { teamID: route.params.teamID } })
  } else if (routeName === 'view-inbox-conversation') {
    router.push({ name: 'view-inbox', params: { viewID: route.params.viewID } })
  } else {
    router.push({ name: 'inbox', params: { type: route.params.type || 'assigned' } })
  }
}
</script>
