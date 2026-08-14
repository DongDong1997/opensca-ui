<script setup lang="ts">
import {computed} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {NEmpty, NPagination, useMessage} from 'naive-ui'
import TaskCard from '@/components/TaskCard.vue'
import {useTasksStore} from '@/stores/tasks'
import {usePagination} from '@/composables/usePagination'

const tasks = useTasksStore()
const router = useRouter()
const message = useMessage()
const {t} = useI18n()

const runningTasks = computed(() =>
  tasks.currentSessionList.filter((t) => t.status === 'running' || t.status === 'pending')
)

const {page, pageSize, pageSizes, pagedItems, total} = usePagination(runningTasks)

function viewTask(id: string) {
  // from=tasks-running 让 ReportView 返回按钮回到"运行中"tab
  router.push({name: 'report', params: {id}, query: {from: 'tasks-running'}})
}

async function cancelTask(id: string) {
  await tasks.cancel(id)
  message.info(t('tasksRunning.cancelSent'))
}

async function removeTask(id: string) {
  await tasks.remove(id)
}
</script>

<template>
  <NEmpty v-if="runningTasks.length === 0" :description="t('tasksRunning.noRunning')" style="margin-top: 60px" />
  <template v-else>
    <TaskCard
      v-for="task in pagedItems"
      :key="task.id"
      :task="task"
      @view="viewTask"
      @report="viewTask"
      @cancel-task="cancelTask"
      @removed="removeTask"
    />
    <div v-if="total > pageSize" class="list-pager">
      <NPagination
        v-model:page="page"
        :page-size="pageSize"
        :item-count="total"
        :page-sizes="pageSizes"
        show-size-picker
        size="small"
      />
    </div>
  </template>
</template>

<style scoped>
.list-pager {
  display: flex;
  justify-content: center;
  padding: 16px 0 8px;
}
</style>
