<script setup lang="ts">
import {computed} from 'vue'
import {useRouter} from 'vue-router'
import {NEmpty, NPagination, useMessage} from 'naive-ui'
import TaskCard from '@/components/TaskCard.vue'
import {useTasksStore} from '@/stores/tasks'
import {usePagination} from '@/composables/usePagination'

const tasks = useTasksStore()
const router = useRouter()
const message = useMessage()

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
  message.info('取消请求已发送')
}

async function removeTask(id: string) {
  await tasks.remove(id)
}
</script>

<template>
  <NEmpty v-if="runningTasks.length === 0" description="暂无运行中的任务" style="margin-top: 60px" />
  <template v-else>
    <TaskCard
      v-for="t in pagedItems"
      :key="t.id"
      :task="t"
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