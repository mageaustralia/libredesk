<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>Keyboard Shortcuts</DialogTitle>
      </DialogHeader>
      <div class="grid grid-cols-2 gap-x-8 gap-y-1 text-sm">
        <template v-for="section in sections" :key="section.title">
          <div class="col-span-2 font-semibold text-xs uppercase tracking-wide text-muted-foreground mt-4 first:mt-2 mb-1 border-b pb-1">
            {{ section.title }}
          </div>
          <div
            v-for="(item, i) in section.items"
            :key="i"
            class="flex justify-between items-center py-1"
          >
            <span>{{ item.desc }}</span>
            <span class="flex gap-1 ml-4 shrink-0">
              <kbd
                v-for="(k, ki) in item.keys"
                :key="ki"
                class="px-1.5 py-0.5 text-xs font-mono bg-muted border rounded shadow-sm"
              >{{ k }}</kbd>
            </span>
          </div>
        </template>
      </div>
      <p class="text-xs text-muted-foreground mt-4">
        <strong>Note:</strong> {{ metaKey }}+B toggles the sidebar when focus is outside the editor. Inside the editor it toggles bold.
        Single-key shortcuts (R, N, /) only work when no input or editor is focused.
      </p>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'

defineProps({
  open: { type: Boolean, default: false }
})
defineEmits(['update:open'])

const isMac = navigator.platform?.toLowerCase().includes('mac')
const metaKey = computed(() => isMac ? 'Cmd' : 'Ctrl')

const sections = computed(() => [
  {
    title: 'Navigation',
    items: [
      { desc: 'Toggle sidebar', keys: [metaKey.value, 'B'] },
      { desc: 'Command palette / macros', keys: [metaKey.value, 'K'] },
      { desc: 'Search', keys: ['/'] },
    ]
  },
  {
    title: 'Conversation',
    items: [
      { desc: 'Reply to ticket', keys: ['R'] },
      { desc: 'Add a note', keys: ['N'] },
      { desc: 'Send message / Add note', keys: [metaKey.value, 'Enter'] },
      { desc: 'Discard draft / collapse reply', keys: ['Esc'] },
    ]
  },
  {
    title: 'Text Formatting (in editor)',
    items: [
      { desc: 'Bold', keys: [metaKey.value, 'B'] },
      { desc: 'Italic', keys: [metaKey.value, 'I'] },
      { desc: 'Underline', keys: [metaKey.value, 'U'] },
      { desc: 'Strikethrough', keys: [metaKey.value, 'Shift', 'X'] },
      { desc: 'Code (inline)', keys: [metaKey.value, 'E'] },
      { desc: 'Bullet list', keys: [metaKey.value, 'Shift', '8'] },
      { desc: 'Ordered list', keys: [metaKey.value, 'Shift', '7'] },
      { desc: 'Blockquote', keys: [metaKey.value, 'Shift', 'B'] },
      { desc: 'Undo', keys: [metaKey.value, 'Z'] },
      { desc: 'Redo', keys: [metaKey.value, 'Shift', 'Z'] },
    ]
  },
  {
    title: 'Other',
    items: [
      { desc: 'Select range of tickets', keys: ['Shift', 'Click'] },
      { desc: 'Show this dialog', keys: ['?'] },
    ]
  }
])
</script>
