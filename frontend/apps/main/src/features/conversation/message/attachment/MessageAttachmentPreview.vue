<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <ImageAttachmentPreview
          v-if="isImage(attachment)"
          :attachment="attachment"
          @preview="openLightbox"
        />
        <AudioAttachmentPreview
          v-else-if="isAudio(attachment)"
          :attachment="attachment"
        />
        <FileAttachmentPreview v-else :attachment="attachment" />
      </div>
    </div>
  </div>

  <ImageLightbox
    v-model="lightboxOpen"
    :images="imageAttachments"
    :start-index="lightboxIndex"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import ImageAttachmentPreview from '@/features/conversation/message/attachment/ImageAttachmentPreview.vue'
import AudioAttachmentPreview from '@/features/conversation/message/attachment/AudioAttachmentPreview.vue'
import FileAttachmentPreview from '@/features/conversation/message/attachment/FileAttachmentPreview.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'

const props = defineProps({
  attachments: { type: Array, required: true }
})

const isImage = (attachment) => (attachment.content_type || '').startsWith('image/')
const isAudio = (attachment) => (attachment.content_type || '').startsWith('audio/')

const imageAttachments = computed(() =>
  (props.attachments || []).filter(isImage)
)

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const openLightbox = (attachment) => {
  const idx = imageAttachments.value.findIndex((a) => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  lightboxOpen.value = true
}
</script>
