<template>
  <Popover :open="showAudio" @update:open="showAudio = $event">
    <PopoverTrigger as-child>
      <div
        class="relative w-36 h-28 flex flex-col items-center justify-between rounded-lg border bg-muted/40 p-3 hover:bg-muted transition-colors cursor-pointer group text-left"
        @click="showAudio = true"
      >
        <div class="flex-1 flex items-center justify-center">
          <FileAudio class="w-10 h-10 text-purple-500" />
        </div>
        <div class="w-full text-center">
          <p class="text-xs font-medium text-foreground truncate" :title="attachment.name">
            {{ shortName(attachment.name) }}
          </p>
          <p class="text-xs text-muted-foreground">{{ formatBytes(attachment.size) }}</p>
        </div>
        <a
          :href="attachment.url"
          download
          class="absolute top-1.5 right-1.5 p-0.5 rounded hover:bg-background opacity-0 group-hover:opacity-100 transition-opacity"
          :title="t('imageLightbox.download')"
          :aria-label="t('imageLightbox.download')"
          @click.stop
        >
          <Download class="w-4 h-4 text-muted-foreground" />
        </a>
      </div>
    </PopoverTrigger>
    <PopoverContent class="w-80 p-3" @click.stop>
      <p class="text-xs font-medium truncate mb-2" :title="attachment.name">
        {{ attachment.name }}
      </p>
      <!-- autoplay: opening the popover is an explicit user action, so
           starting playback immediately matches the click intent. -->
      <audio :src="attachment.url" controls autoplay preload="auto" class="w-full h-8" />
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, FileAudio } from 'lucide-vue-next'
import { formatBytes } from '@shared-ui/utils/file'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'

defineProps({
  attachment: { type: Object, required: true }
})

const { t } = useI18n()
const showAudio = ref(false)
const shortName = (name) => (name || '').substring(0, 40)
</script>
