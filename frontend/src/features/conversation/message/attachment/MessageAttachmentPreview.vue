<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <ImageAttachmentPreview v-if="isImage(attachment)" :attachment="attachment" />
        <div v-else-if="isAudio(attachment)" class="flex items-center gap-2 rounded-lg border bg-gray-50 dark:bg-gray-800 px-3 py-2">
          <audio controls preload="auto" class="h-8 max-w-[260px]">
            <source :src="attachment.url" />
          </audio>
          <a :href="attachment.url" download @click.stop class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 shrink-0" title="Download">
            <Download class="w-4 h-4 text-gray-500" />
          </a>
        </div>
        <FileAttachmentPreview v-else :attachment="attachment" />
      </div>
    </div>
  </div>
</template>

<script setup>
import ImageAttachmentPreview from '@/features/conversation/message/attachment/ImageAttachmentPreview.vue'
import FileAttachmentPreview from '@/features/conversation/message/attachment/FileAttachmentPreview.vue'
import { Download } from 'lucide-vue-next'

defineProps({
  attachments: {
    type: Array,
    required: true
  }
})

const isImage = (attachment) => {
  return attachment.content_type.includes('image')
}

const isAudio = (attachment) => {
  return attachment.content_type.startsWith('audio/')
}
</script>
