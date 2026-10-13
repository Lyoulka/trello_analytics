<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { alertVariants, type AlertVariants } from '.'

const props = defineProps<{
  variant?: AlertVariants['variant']
  class?: HTMLAttributes['class']
  role?: string
}>()
</script>

<template>
  <div :role="role ?? 'alert'" :class="cn(alertVariants({ variant }), props.class)">
    <slot />
  </div>
</template>
