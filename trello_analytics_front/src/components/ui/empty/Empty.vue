<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    :class="
      cn(
        'flex min-h-[12rem] flex-col items-center justify-center gap-2 rounded-xl border border-dashed p-8 text-center',
        props.class,
      )
    "
  >
    <slot />
  </div>
</template>
