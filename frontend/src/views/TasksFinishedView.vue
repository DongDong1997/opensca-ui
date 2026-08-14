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

const finishedTasks = computed(() =>
  tasks.currentSessionList.filter((t) => t.status === 'success' || t.status === 'failed' || t.status === 'canceled')
)

const {page, pageSize, pageSizes, pagedItems, total} = usePagination(finishedTasks)

function viewTask(id: string) {
  // from=tasks-finished 让 ReportView 返回按钮回到"已完成"tab
  router.push({name: 'report', params: {id}, query: {from: 'tasks-finished'}})
}

async function removeTask(id: string) {
  await tasks.remove(id)
}
</script>

<template>
  <NEmpty v-if="finishedTasks.length === 0" :description="t('tasksFinished.noFinished')" style="margin-top: 60px" />
  <template v-else>
    <TaskCard
      v-for="task in pagedItems"
      :key="task.id"
      :task="task"
      @view="viewTask"
      @report="viewTask"
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
