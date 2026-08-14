import {onBeforeUnmount} from 'vue'
import {useWailsEvent} from './useWailsEvent'
import {useTasksStore} from '@/stores/tasks'
import type {ScanLogEvent, ScanProgressEvent, ScanStatusEvent, ScanDoneEvent, TaskStatus} from '@/api/types'

/**
 * 订阅某个任务的事件流，把数据 dispatch 到 tasks store。
 * 用法：进入任务详情时调用 useTaskStream(taskID)，离开时自动取消订阅。
 */
export function useTaskStream(taskID: () => string) {
  const tasks = useTasksStore()

  const offLog = useWailsEvent<ScanLogEvent>('scan:log', (p) => {
    if (p.taskID === taskID()) tasks.appendLog(p.taskID, p.line, p.ts)
  })
  const offProgress = useWailsEvent<ScanProgressEvent>('scan:progress', (p) => {
    if (p.taskID === taskID()) tasks.setProgress(p.taskID, p.percent, p.stage)
  })
  const offUpdate = useWailsEvent<ScanStatusEvent>('scan:update', (p) => {
    if (p.taskID === taskID()) tasks.setStatus(p.taskID, p.status as TaskStatus)
  })
  const offDone = useWailsEvent<ScanDoneEvent>('scan:done', (p) => {
    if (p.taskID === taskID()) {
      tasks.setStatus(p.taskID, p.status as TaskStatus)
      tasks.setFinished(p.taskID, p.durationMs, p.reportPath)
    }
  })

  // useWailsEvent 已经注册 onBeforeUnmount，但这里我们也冗余卸载以便测试/手动控制
  onBeforeUnmount(() => {
    offLog.off()
    offProgress.off()
    offUpdate.off()
    offDone.off()
  })
}