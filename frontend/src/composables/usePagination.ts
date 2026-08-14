// 通用列表分页 composable。
//
// 用法：
//   const source = computed(() => items.filter(...))
//   const { page, pageSize, pageSizes, pagedItems, total } = usePagination(source)
//   :items="pagedItems"
//   <NPagination v-if="total > pageSize" v-model:page="page" :page-size="pageSize"
//     :item-count="total" :page-sizes="pageSizes" show-size-picker />
//
// 行为：
//   - 每次 source 变化（过滤/新增/删除）自动回到第 1 页，避免空页
//   - 默认 pageSize=10，可选 [5, 10, 20, 50]

import {computed, ref, watch, type Ref, type ComputedRef} from 'vue'

export interface UsePaginationOptions {
  defaultPageSize?: number
  pageSizes?: number[]
}

export function usePagination<T>(
  source: Ref<T[]> | ComputedRef<T[]>,
  opts: UsePaginationOptions = {}
) {
  const defaultPageSize = opts.defaultPageSize ?? 10
  const pageSizes = opts.pageSizes ?? [5, 10, 20, 50]

  const page = ref(1)
  const pageSize = ref(defaultPageSize)

  const total = computed(() => source.value.length)

  const pagedItems = computed(() => {
    const start = (page.value - 1) * pageSize.value
    return source.value.slice(start, start + pageSize.value)
  })

  // source 变化 → 重置到第 1 页（避免过滤后停在原页变成空页）
  watch(source, () => {
    page.value = 1
  })

  return {
    page,
    pageSize,
    pageSizes,
    pagedItems,
    total,
  }
}