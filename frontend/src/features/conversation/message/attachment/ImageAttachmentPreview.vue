<template>
  <div class="flex flex-wrap items-center group text-left">
    <div class="relative cursor-pointer" @click="showPreview = true">
      <img :src="getThumbFilepath(attachment.url)" class="w-36 h-28 flex items-center object-cover" />
      <div class="p-1 absolute inset-0 text-gray-50 opacity-0 group-hover:opacity-100 overlay text-wrap">
        <div class="flex flex-col justify-between h-full">
          <div>
            <p class="font-bold text-xs">{{ trimAttachmentName(attachment.name) }}</p>
            <p class="text-xs">{{ formatBytes(attachment.size) }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Eye :size="20" />
            <a :href="attachment.url" download @click.stop class="hover:text-white/80">
              <Download :size="20" />
            </a>
          </div>
        </div>
      </div>
    </div>

    <!-- Lightbox overlay -->
    <Teleport to="body">
      <div
        v-if="showPreview"
        class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
        @click.self="showPreview = false"
      >
        <button
          class="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
          @click="showPreview = false"
        >
          <X :size="28" />
        </button>
        <a
          :href="attachment.url"
          download
          class="absolute top-4 right-14 text-white hover:text-gray-300 z-10"
          title="Download"
        >
          <Download :size="24" />
        </a>
        <img
          :src="attachment.url"
          class="max-w-[90vw] max-h-[90vh] object-contain rounded shadow-2xl"
          :alt="attachment.name"
        />
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { formatBytes, getThumbFilepath } from '@/utils/file.js'
import { Download, Eye, X } from 'lucide-vue-next'

const props = defineProps({
  attachment: {
    type: Object,
    required: true
  }
})

const showPreview = ref(false)

const trimAttachmentName = (name) => {
  return name.substring(0, 40)
}
</script>
