<script setup lang="ts">
import { onMounted, ref } from 'vue'

const apiStatus = ref<'loading' | 'ok' | 'error'>('loading')
const apiDetail = ref('')

onMounted(async () => {
  try {
    const res = await fetch('/api/v1/health')
    const data = (await res.json()) as { status?: string; database?: string }
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`)
    }
    apiStatus.value = 'ok'
    apiDetail.value = `status=${data.status}, database=${data.database ?? 'n/a'}`
  } catch (err) {
    apiStatus.value = 'error'
    apiDetail.value = err instanceof Error ? err.message : String(err)
  }
})
</script>

<template>
  <main class="page">
    <h1>Trello Team Analytics</h1>
    <p class="lead">Локальный каркас frontend (Vue 3 + Vite). Бизнес-логика не реализована.</p>
    <p>
      API health:
      <strong :data-status="apiStatus">{{ apiStatus }}</strong>
      <span v-if="apiDetail"> — {{ apiDetail }}</span>
    </p>
  </main>
</template>
