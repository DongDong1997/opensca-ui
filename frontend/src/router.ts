import {createRouter, createWebHashHistory, type RouteRecordRaw} from 'vue-router'

const routes: RouteRecordRaw[] = [
  // 应用入口 → 首页
  {path: '/', redirect: '/home'},
  {
    path: '/home',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: {titleKey: 'router.home'}
  },
  {
    path: '/welcome',
    name: 'welcome',
    component: () => import('@/views/WelcomeView.vue'),
    meta: {layout: 'plain', titleKey: 'router.welcome'}
  },
  {
    path: '/scan',
    name: 'scan',
    component: () => import('@/views/ScanView.vue'),
    meta: {titleKey: 'router.scan'}
  },
  {
    // 任务管理：布局壳子，只挂运行中/已完成两个子页面；
    // 持久化的全部扫描记录（按项目分组）由"历史记录"页面承担。
    path: '/tasks',
    component: () => import('@/views/TasksView.vue'),
    meta: {titleKey: 'router.tasks'},
    children: [
      // 默认直接渲染"运行中"，避免 redirect 链路 / 守卫时序问题
      {path: '', name: 'tasks', component: () => import('@/views/TasksRunningView.vue')},
      {path: 'running', name: 'tasks-running', component: () => import('@/views/TasksRunningView.vue')},
      {path: 'finished', name: 'tasks-finished', component: () => import('@/views/TasksFinishedView.vue')}
    ]
  },
  {
    path: '/report/:id',
    name: 'report',
    component: () => import('@/views/ReportView.vue'),
    props: true,
    meta: {titleKey: 'router.report'}
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: {titleKey: 'router.settings'}
  },
  {
    // 顶栏"历史记录"菜单项：展示所有文件夹的历史任务（每文件夹排除最新一条），
    // 顶部没有 按项目/运行中/已完成 tab —— 整个页面就是"历史"语义。
    path: '/history',
    name: 'history',
    component: () => import('@/views/HistoryView.vue'),
    meta: {titleKey: 'router.history'}
  },
  {
    // 单文件夹的历史记录子页面（点击 /history 里的"历史 N 条"标签进入）——
    // 与任务管理是独立的两套页面，没有"按项目/运行中/已完成"tab，
    // 顶部显示"历史记录"标题 + 文件夹路径。
    path: '/history/:path',
    name: 'history-folder',
    component: () => import('@/views/TasksHistoryView.vue'),
    props: true,
    meta: {titleKey: 'router.history'}
  },
  {path: '/:pathMatch(.*)*', redirect: '/home'}
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// 未配置 CLI 时的页面重定向（document.title 由 App.vue 的 locale watcher 负责）
import {useConfigStore} from '@/stores/config'
import {applyLanguage} from '@/i18n'

router.beforeEach(async (to, _from, next) => {
  const cfg = useConfigStore()
  if (!cfg.loaded) {
    await cfg.load()
  }
  // 在首个路由组件挂载前应用持久化语言，避免闪一帧中文
  applyLanguage(cfg.language || 'zh-CN')
  // 只有真正需要 CLI 的页面才拦；首页、欢迎、设置、报告（已经扫完）都放行
  const requiresCli = to.name !== 'home' && to.name !== 'welcome' && to.name !== 'settings'
  if (requiresCli && !cfg.cliValid && cfg.cliPath !== '') {
    // 配置了路径但验证失败 → 去设置页修
    return next({name: 'settings'})
  }
  if (requiresCli && !cfg.cliValid) {
    return next({name: 'welcome'})
  }
  next()
})