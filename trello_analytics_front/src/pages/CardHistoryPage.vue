<script setup lang="ts">
/**
 * Deep-link compatibility: /cards/:id/history → /cards?history=:id (modal on CardsPage).
 */
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

onMounted(() => {
  const id = String(route.params.id ?? '')
  void router.replace({
    name: 'cards',
    query: id ? { history: id } : {},
  })
})
</script>

<template>
  <div class="text-sm text-muted-foreground">Открываем историю…</div>
</template>
