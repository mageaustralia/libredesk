<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <ImageAttachmentPreview v-if="isImage(attachment)" :attachment="attachment" @preview="openLightbox" />
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

  <!-- Shared lightbox with prev/next -->
  <Teleport to="body">
    <div
      v-if="lightboxOpen"
      class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
      @click.self="lightboxOpen = false"
      @keydown.escape="lightboxOpen = false"
      @keydown.left="prevImage"
      @keydown.right="nextImage"
      tabindex="0"
      ref="lightboxEl"
    >
      <!-- Close -->
      <button
        class="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
        @click="lightboxOpen = false"
      >
        <X :size="28" />
      </button>
      <!-- Download -->
      <a
        :href="currentImage?.url"
        download
        class="absolute top-4 right-14 text-white hover:text-gray-300 z-10"
        title="Download"
      >
        <Download :size="24" />
      </a>
      <!-- Counter -->
      <div v-if="imageAttachments.length > 1" class="absolute top-4 left-4 text-white/70 text-sm z-10">
        {{ lightboxIndex + 1 }} / {{ imageAttachments.length }}
      </div>
      <!-- Prev -->
      <button
        v-if="imageAttachments.length > 1"
        class="absolute left-4 top-1/2 -translate-y-1/2 text-white hover:text-gray-300 z-10 p-2"
        @click.stop="prevImage"
      >
        <ChevronLeft :size="32" />
      </button>
      <!-- Next -->
      <button
        v-if="imageAttachments.length > 1"
        class="absolute right-4 top-1/2 -translate-y-1/2 text-white hover:text-gray-300 z-10 p-2"
        @click.stop="nextImage"
      >
        <ChevronRight :size="32" />
      </button>
      <!-- Loading spinner -->
      <div v-if="imageLoading" class="absolute inset-0 flex items-center justify-center pointer-events-none">
        <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
      </div>
      <!-- Image -->
      <img
        :key="currentImage?.uuid"
        :src="currentImage?.url"
        class="max-w-[90vw] max-h-[90vh] object-contain rounded shadow-2xl transition-opacity duration-150"
        :class="imageLoading ? 'opacity-0' : 'opacity-100'"
        :alt="currentImage?.name"
        @load="imageLoading = false"
      />
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import ImageAttachmentPreview from '@/features/conversation/message/attachment/ImageAttachmentPreview.vue'
import FileAttachmentPreview from '@/features/conversation/message/attachment/FileAttachmentPreview.vue'
import { Download, X, ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps({
  attachments: {
    type: Array,
    required: true
  }
})

const isImage = (attachment) => attachment.content_type.includes('image')
const isAudio = (attachment) => attachment.content_type.startsWith('audio/')

const imageAttachments = computed(() => (props.attachments || []).filter(isImage))

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)
const lightboxEl = ref(null)
const imageLoading = ref(false)

const currentImage = computed(() => imageAttachments.value[lightboxIndex.value])

// Preload adjacent images when index changes
watch(lightboxIndex, () => {
  const imgs = imageAttachments.value
  if (imgs.length <= 1) return
  const prev = (lightboxIndex.value - 1 + imgs.length) % imgs.length
  const next = (lightboxIndex.value + 1) % imgs.length
  new Image().src = imgs[prev].url
  new Image().src = imgs[next].url
})

function openLightbox(attachment) {
  const idx = imageAttachments.value.findIndex(a => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  imageLoading.value = false
  lightboxOpen.value = true
  nextTick(() => {
    lightboxEl.value?.focus()
    // Preload all images when lightbox opens
    imageAttachments.value.forEach(a => { new Image().src = a.url })
  })
}

function prevImage() {
  if (imageAttachments.value.length <= 1) return
  imageLoading.value = true
  lightboxIndex.value = (lightboxIndex.value - 1 + imageAttachments.value.length) % imageAttachments.value.length
}

function nextImage() {
  if (imageAttachments.value.length <= 1) return
  imageLoading.value = true
  lightboxIndex.value = (lightboxIndex.value + 1) % imageAttachments.value.length
}
</script>
