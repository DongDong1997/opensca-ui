import {onBeforeUnmount} from 'vue'
import {EventsOn, EventsOff, EventsOnce} from '../../wailsjs/runtime/runtime'

/**
 * 订阅 Wails 后端事件，组件卸载时自动取消订阅。
 *
 * @example
 * const { off } = useWailsEvent('scan:log', (p) => console.log(p))
 * // 手动: off()
 */
export function useWailsEvent<T = unknown>(
  name: string,
  handler: (payload: T) => void
): {off: () => void} {
  const wrapped = handler as unknown as (...args: unknown[]) => void
  EventsOn(name, wrapped)
  const off = () => EventsOff(name)
  onBeforeUnmount(off)
  return {off}
}

/**
 * 一次性事件订阅
 */
export function useWailsEventOnce<T = unknown>(
  name: string,
  handler: (payload: T) => void
): {off: () => void} {
  const wrapped = handler as unknown as (...args: unknown[]) => void
  EventsOnce(name, wrapped)
  const off = () => EventsOff(name)
  onBeforeUnmount(off)
  return {off}
}