<template>
  <table class="conversation-table w-full text-sm table-fixed">
    <colgroup>
      <col :style="{ width: columnWidths.checkbox + 'px' }">
      <col :style="{ width: columnWidths.contact + 'px' }">
      <col :style="{ width: columnWidths.subject + 'px' }">
      <col :style="{ width: columnWidths.state + 'px' }">
      <col :style="{ width: columnWidths.group + 'px' }">
      <col :style="{ width: columnWidths.agent + 'px' }">
      <col :style="{ width: columnWidths.priority + 'px' }">
      <col :style="{ width: columnWidths.status + 'px' }">
      <col :style="{ width: columnWidths.updated + 'px' }">
    </colgroup>
    <thead class="sticky top-0 bg-background z-10 border-b">
      <tr class="text-left text-xs text-muted-foreground">
        <th class="px-2 py-2">
          <Checkbox
            :checked="conversationStore.allSelected"
            @update:checked="toggleSelectAll"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Contact
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'contact')"
            @dblclick="resetColumn('contact')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Subject
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'subject')"
            @dblclick="resetColumn('subject')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          State
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'state')"
            @dblclick="resetColumn('state')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Group
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'group')"
            @dblclick="resetColumn('group')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Agent
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'agent')"
            @dblclick="resetColumn('agent')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Priority
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'priority')"
            @dblclick="resetColumn('priority')"
          />
        </th>
        <th class="px-2 py-2 font-medium relative select-none">
          Status
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'status')"
            @dblclick="resetColumn('status')"
          />
        </th>
        <th class="px-2 py-2 font-medium text-right relative select-none">
          Updated
          <div
            class="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 z-20"
            @mousedown.prevent="startResize($event, 'updated')"
            @dblclick="resetColumn('updated')"
          />
        </th>
      </tr>
    </thead>
    <tbody class="divide-y">
      <tr
        v-for="conversation in conversations"
        :key="conversation.uuid"
        class="conversation-table-row cursor-pointer transition-colors hover:bg-accent/20"
        :class="{
          'bg-accent/60': conversation.uuid === conversationStore.current?.uuid,
          'bg-primary/5': conversationStore.isSelected(conversation.uuid) && conversation.uuid !== conversationStore.current?.uuid
        }"
        @click="navigateToConversation(conversation)"
      >
        <!-- Checkbox -->
        <td class="px-2 py-2" @click.stop>
          <Checkbox
            :checked="conversationStore.isSelected(conversation.uuid)"
            @update:checked="() => conversationStore.toggleSelect(conversation.uuid)"
          />
        </td>

        <!-- Contact -->
        <td class="px-2 py-2">
          <div class="flex items-center gap-2 min-w-0">
            <Avatar class="w-6 h-6 rounded-full shrink-0">
              <AvatarImage
                :src="conversation.contact.avatar_url || ''"
                v-if="conversation.contact.avatar_url"
              />
              <AvatarFallback class="text-[10px]">
                {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
              </AvatarFallback>
            </Avatar>
            <span class="truncate text-xs font-medium">
              {{ conversationStore.getContactFullName(conversation.uuid) }}
            </span>
          </div>
        </td>

        <!-- Subject -->
        <td class="px-2 py-2">
          <div class="flex items-center gap-1.5 min-w-0">
            <span class="text-[10px] text-muted-foreground whitespace-nowrap" v-if="conversation.reference_number">
              #{{ conversation.reference_number }}
            </span>
            <span class="truncate font-medium text-xs">
              {{ conversation.subject || 'No subject' }}
            </span>
            <div
              v-if="conversation.unread_message_count > 0"
              class="shrink-0 w-4 h-4 flex items-center justify-center bg-green-600 text-white text-[9px] font-medium rounded-full"
            >
              {{ conversation.unread_message_count }}
            </div>
          </div>
        </td>

        <!-- State (last_message_sender indicator) -->
        <td class="px-2 py-2">
          <span
            class="text-[10px] font-medium px-1.5 py-0.5 rounded-full whitespace-nowrap"
            :class="getStatusClass(conversation)"
          >{{ conversation.status }}</span>
        </td>

        <!-- Group (Team) -->
        <td class="px-2 py-2" @click.stop>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                class="text-xs flex items-center gap-1 hover:text-foreground transition-colors cursor-pointer truncate max-w-full"
                :class="conversation.assigned_team_name ? 'text-muted-foreground' : 'text-muted-foreground/50'"
              >
                {{ conversation.assigned_team_name || '—' }}
                <ChevronDown class="w-2.5 h-2.5 opacity-50 shrink-0" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" class="max-h-60 overflow-y-auto">
              <DropdownMenuItem
                v-if="conversation.assigned_team_name"
                @click="unassignTeam(conversation)"
                class="text-xs text-muted-foreground"
              >None</DropdownMenuItem>
              <DropdownMenuSeparator v-if="conversation.assigned_team_name" />
              <DropdownMenuItem
                v-for="team in teamsStore.options"
                :key="'team-' + team.value"
                @click="assignTeam(conversation, team)"
                class="text-xs"
              >{{ team.label }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </td>

        <!-- Agent -->
        <td class="px-2 py-2" @click.stop>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                class="text-xs flex items-center gap-1 hover:text-foreground transition-colors cursor-pointer truncate max-w-full"
                :class="conversation.assigned_user_name ? 'text-muted-foreground' : 'text-orange-500 dark:text-orange-400'"
              >
                {{ conversation.assigned_user_name || 'Unassigned' }}
                <ChevronDown class="w-2.5 h-2.5 opacity-50 shrink-0" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" class="max-h-60 overflow-y-auto">
              <DropdownMenuItem
                v-if="conversation.assigned_user_name"
                @click="unassignAgent(conversation)"
                class="text-xs text-muted-foreground"
              >None</DropdownMenuItem>
              <DropdownMenuSeparator v-if="conversation.assigned_user_name" />
              <DropdownMenuItem
                v-for="agent in usersStore.options"
                :key="'agent-' + agent.value"
                @click="assignAgent(conversation, agent)"
                class="text-xs"
              >{{ agent.label }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </td>

        <!-- Priority -->
        <td class="px-2 py-2" @click.stop>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button class="text-xs flex items-center gap-1 hover:text-foreground transition-colors cursor-pointer whitespace-nowrap">
                <span
                  class="w-2 h-2 rounded-full shrink-0"
                  :class="getPriorityDotClass(conversation)"
                ></span>
                {{ conversation.priority || 'None' }}
                <ChevronDown class="w-2.5 h-2.5 opacity-50 shrink-0" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem
                v-for="priority in conversationStore.priorityOptions"
                :key="priority.value"
                @click="updatePriority(conversation, priority.label)"
                class="text-xs"
              >{{ priority.label }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </td>

        <!-- Status -->
        <td class="px-2 py-2" @click.stop>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button class="text-xs flex items-center gap-1 hover:text-foreground transition-colors cursor-pointer text-muted-foreground whitespace-nowrap">
                {{ conversation.status }}
                <ChevronDown class="w-2.5 h-2.5 opacity-50 shrink-0" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem
                v-for="status in conversationStore.statusOptionsNoSnooze"
                :key="status.value"
                @click="updateStatus(conversation, status.label)"
                class="text-xs"
              >{{ status.label }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </td>

        <!-- Updated -->
        <td class="px-2 py-2 text-right">
          <span class="text-xs text-muted-foreground whitespace-nowrap">
            {{ getRelativeTime(conversation.last_message_at, now) }}
          </span>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ChevronDown } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { getRelativeTime } from '@/utils/datetime'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const router = useRouter()
const route = useRoute()
const now = ref(new Date())
let timer = null

defineProps({
  conversations: Array
})

// ── Column resize logic ──────────────────────────────────────────────
const STORAGE_KEY = 'libredesk-table-col-widths'

const defaultWidths = {
  checkbox: 36,
  contact: 140,
  subject: 400,
  state: 80,
  group: 120,
  agent: 120,
  priority: 80,
  status: 80,
  updated: 90,
}

const MIN_WIDTH = 60

const loadWidths = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const parsed = JSON.parse(saved)
      return { ...defaultWidths, ...parsed }
    }
  } catch {
    // ignore parse errors
  }
  return { ...defaultWidths }
}

const columnWidths = ref(loadWidths())

const saveWidths = () => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(columnWidths.value))
}

let resizingCol = null
let startX = 0
let startWidth = 0

const startResize = (event, col) => {
  resizingCol = col
  startX = event.clientX
  startWidth = columnWidths.value[col]
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

const onMouseMove = (event) => {
  if (!resizingCol) return
  const delta = event.clientX - startX
  const newWidth = Math.max(MIN_WIDTH, startWidth + delta)
  columnWidths.value[resizingCol] = newWidth
}

const onMouseUp = () => {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  saveWidths()
  resizingCol = null
}

const resetColumn = (col) => {
  columnWidths.value[col] = defaultWidths[col]
  saveWidths()
}
// ── End column resize logic ──────────────────────────────────────────

onMounted(() => {
  timer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  // Clean up resize listeners in case unmount happens mid-drag
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
})

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

function navigateToConversation(conversation) {
  const baseRoute = route.name.includes('team')
    ? 'team-inbox-conversation'
    : route.name.includes('view')
      ? 'view-inbox-conversation'
      : 'inbox-conversation'
  router.push({
    name: baseRoute,
    params: {
      uuid: conversation.uuid,
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
    },
    query: conversation.mentioned_message_uuid
      ? { scrollTo: conversation.mentioned_message_uuid }
      : {}
  })
}

function getStatusClass(conversation) {
  const s = (conversation.status || '').toLowerCase()
  switch (s) {
    case 'open': return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'replied': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'resolved': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    case 'closed': return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
    case 'snoozed': return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
}

function getPriorityDotClass(conversation) {
  const p = (conversation.priority || '').toLowerCase()
  switch (p) {
    case 'urgent': return 'bg-red-500'
    case 'high': return 'bg-orange-500'
    case 'medium': return 'bg-yellow-500'
    case 'low': return 'bg-blue-500'
    default: return 'bg-gray-300'
  }
}

async function assignAgent(conversation, agent) {
  try {
    await api.updateAssignee(conversation.uuid, 'user', { assignee_id: parseInt(agent.value) })
    conversationStore.updateConversationField(conversation.uuid, 'assigned_user_name', agent.label)
  } catch (error) { handleHTTPError(error) }
}

async function unassignAgent(conversation) {
  try {
    await api.removeAssignee(conversation.uuid, 'user')
    conversationStore.updateConversationField(conversation.uuid, 'assigned_user_name', null)
  } catch (error) { handleHTTPError(error) }
}

async function assignTeam(conversation, team) {
  try {
    await api.updateAssignee(conversation.uuid, 'team', { assignee_id: parseInt(team.value) })
    conversationStore.updateConversationField(conversation.uuid, 'assigned_team_name', team.label)
  } catch (error) { handleHTTPError(error) }
}

async function unassignTeam(conversation) {
  try {
    await api.removeAssignee(conversation.uuid, 'team')
    conversationStore.updateConversationField(conversation.uuid, 'assigned_team_name', null)
  } catch (error) { handleHTTPError(error) }
}

async function updatePriority(conversation, priority) {
  try {
    await api.updateConversationPriority(conversation.uuid, { priority })
    conversationStore.updateConversationField(conversation.uuid, 'priority', priority)
  } catch (error) { handleHTTPError(error) }
}

async function updateStatus(conversation, status) {
  try {
    await api.updateConversationStatus(conversation.uuid, { status })
    conversationStore.updateConversationField(conversation.uuid, 'status', status)
  } catch (error) { handleHTTPError(error) }
}
</script>
