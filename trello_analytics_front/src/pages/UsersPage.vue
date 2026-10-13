<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createUser, fetchUsers, patchUser } from '@/api/users'
import { messageFromUnknown } from '@/api/errors'
import type { User } from '@/types/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const loading = ref(true)
const error = ref<string | null>(null)
const users = ref<User[]>([])
const formError = ref<string | null>(null)
const formOk = ref<string | null>(null)

const form = ref({
  email: '',
  password: '',
  display_name: '',
  role: 'member',
})

async function load() {
  loading.value = true
  error.value = null
  try {
    const res = await fetchUsers()
    users.value = res.users ?? []
  } catch (err) {
    users.value = []
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

async function onCreate() {
  formError.value = null
  formOk.value = null
  try {
    await createUser({
      email: form.value.email.trim(),
      password: form.value.password,
      display_name: form.value.display_name.trim(),
      role: form.value.role,
    })
    form.value = { email: '', password: '', display_name: '', role: 'member' }
    formOk.value = 'Пользователь создан'
    await load()
  } catch (err) {
    formError.value = messageFromUnknown(err)
  }
}

async function deactivate(u: User) {
  formError.value = null
  try {
    await patchUser(u.id, { is_active: false })
    await load()
  } catch (err) {
    formError.value = messageFromUnknown(err)
  }
}

async function activate(u: User) {
  formError.value = null
  try {
    await patchUser(u.id, { is_active: true })
    await load()
  } catch (err) {
    formError.value = messageFromUnknown(err)
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Пользователи</h1>
      <p class="text-sm text-muted-foreground">Создание, список и деактивация (удаления нет).</p>
    </div>

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>
    <Alert v-if="formError" variant="destructive">
      <AlertTitle>Ошибка формы</AlertTitle>
      <AlertDescription>{{ formError }}</AlertDescription>
    </Alert>
    <Alert v-if="formOk">
      <AlertTitle>Готово</AlertTitle>
      <AlertDescription>{{ formOk }}</AlertDescription>
    </Alert>

    <Card>
      <CardHeader>
        <CardTitle class="text-base">Создать пользователя</CardTitle>
        <CardDescription>Пароль ≥ 12 символов.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="grid gap-3 sm:grid-cols-2" @submit.prevent="onCreate">
          <div class="space-y-1">
            <Label for="u-email">Email</Label>
            <Input id="u-email" v-model="form.email" type="email" required />
          </div>
          <div class="space-y-1">
            <Label for="u-name">Отображаемое имя</Label>
            <Input id="u-name" v-model="form.display_name" required />
          </div>
          <div class="space-y-1">
            <Label for="u-pass">Пароль</Label>
            <Input id="u-pass" v-model="form.password" type="password" minlength="12" required />
          </div>
          <div class="space-y-1">
            <Label for="u-role">Роль</Label>
            <select
              id="u-role"
              v-model="form.role"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
            >
              <option value="member">Участник</option>
              <option value="team_lead">Тимлид</option>
              <option value="admin">Админ</option>
            </select>
          </div>
          <div class="sm:col-span-2">
            <Button type="submit">Создать</Button>
          </div>
        </form>
      </CardContent>
    </Card>

    <Skeleton v-if="loading" class="h-32" />

    <Empty v-else-if="!error && users.length === 0">
      <EmptyHeader>
        <EmptyTitle>Нет пользователей</EmptyTitle>
        <EmptyDescription>Список пуст.</EmptyDescription>
      </EmptyHeader>
    </Empty>

    <Card v-else-if="users.length">
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Email</TableHead>
              <TableHead>Имя</TableHead>
              <TableHead>Роль</TableHead>
              <TableHead>Активен</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="u in users" :key="u.id">
              <TableCell>{{ u.email }}</TableCell>
              <TableCell>{{ u.display_name }}</TableCell>
              <TableCell>
                <Badge variant="outline">
                  {{
                    u.role === 'admin'
                      ? 'Админ'
                      : u.role === 'team_lead'
                        ? 'Тимлид'
                        : 'Участник'
                  }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge :variant="u.is_active ? 'success' : 'secondary'">
                  {{ u.is_active ? 'да' : 'нет' }}
                </Badge>
              </TableCell>
              <TableCell class="text-right">
                <Button
                  v-if="u.is_active"
                  size="sm"
                  variant="outline"
                  @click="deactivate(u)"
                >
                  Деактивировать
                </Button>
                <Button v-else size="sm" variant="secondary" @click="activate(u)">
                  Activate
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </div>
</template>
