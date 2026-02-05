<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <SidebarMenuButton>
        <Palette />
      </SidebarMenuButton>
    </DropdownMenuTrigger>
    <DropdownMenuContent side="right" align="end">
      <DropdownMenuItem
        v-for="t in THEMES"
        :key="t"
        @click="setTheme(t)"
        :class="{ 'bg-accent': currentTheme === t }"
      >
        <Check v-if="currentTheme === t" class="mr-2 h-4 w-4" />
        <span v-else class="mr-2 h-4 w-4" />
        {{ t.charAt(0).toUpperCase() + t.slice(1) }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { useTheme } from '@/composables/useTheme'
import { Palette, Check } from 'lucide-vue-next'
import { SidebarMenuButton } from '@/components/ui/sidebar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'

const { currentTheme, setTheme, THEMES } = useTheme()
</script>
