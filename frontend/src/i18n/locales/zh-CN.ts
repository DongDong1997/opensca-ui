export default {
  common: {
    browse: '浏览',
    verify: '验证',
    update: '更新',
    cancel: '取消',
    clear: '清空',
    remove: '移除',
    open: '打开',
    close: '关闭',
    unknown: '未知',
    loading: '加载中…',
    scanning: '扫描中…',
    saveSuccess: '设置已保存',
    saveFailed: '保存失败: {msg}',
    openFailed: '打开失败: {msg}',
    pickDirFailed: '选择目录失败: {msg}',
    pickFileFailed: '选择文件失败: {msg}',
    selectFailed: '选择失败: {msg}',
    startFailed: '启动失败: {msg}',
    checkUpdateFailed: '检查更新失败: {msg}',
    updateFailed: '更新失败: {msg}'
  },
  severity: {
    critical: '严重',
    high: '高危',
    medium: '中危',
    low: '低危',
    info: '提示',
    unknown: '未知'
  },
  task: {
    status: {
      pending: '等待中',
      running: '运行中',
      success: '完成',
      failed: '失败',
      canceled: '已取消'
    }
  },
  router: {
    home: '首页',
    welcome: '欢迎使用 OpenSCA UI',
    scan: '新建扫描',
    tasks: '任务管理',
    report: '报告详情',
    settings: '设置',
    history: '历史记录'
  },
  shell: {
    nav: {
      home: '首页',
      scan: '新建扫描',
      tasks: '任务管理',
      history: '历史记录',
      settings: '设置'
    },
    cli: {
      notConfigured: '未配置 CLI',
      valid: 'CLI {version}',
      invalid: 'CLI 无效 · {reason}',
      checkPath: '请检查路径'
    },
    tip: {
      hasUpdate: '有更新，最新版本 v{version}，点击更新',
      hasUpdateTitle: '点击前往设置页更新到 v{version}',
      latest: '已是最新版',
      latestTitle: '已是最新版本，点击查看设置',
      unset: '未配置 CLI，点击前往设置',
      unsetTitle: '尚未配置 opensca-cli，点击前往设置'
    }
  },
  settings: {
    nav: {
      cli: 'CLI',
      general: '通用',
      vuln: '漏洞库',
      runtime: '运行',
      about: '关于'
    },
    card: {
      cli: 'CLI 设置',
      general: '通用',
      vuln: '漏洞库',
      runtime: '运行设置',
      about: '关于'
    },
    saveBar: '保存设置',
    language: {
      label: '界面语言',
      desc: '切换后立即生效，无需重启'
    },
    cli: {
      path: 'opensca-cli 路径',
      verifyPassed: '✓ 验证通过',
      verifyFailed: '✗ 验证失败',
      version: '版本: {v}'
    },
    general: {
      folderTitle: '生成报告位置（文件夹扫描）',
      zipTitle: '生成报告位置（压缩包扫描）',
      useDefault: '是否使用默认位置',
      customPathPlaceholder: '自定义报告目录路径',
      autoSubdirs1: '路径下会自动创建',
      autoSubdirs2: '与',
      autoSubdirs3: '子目录。',
      folderHint1: '选择文件夹扫描时，默认生成在文件夹根目录的',
      folderHint2: '中。无法生成',
      folderHint3: '时，默认位置在',
      folderHint4: '。',
      zipHint: '默认位置就是',
      defaultPathExample: 'C:\\Users\\<用户名>\\AppData\\Roaming\\opensca-ui\\reports'
    },
    vuln: {
      token: '云漏洞库 Token',
      tokenPlaceholder: '可选，留空时仅使用本地漏洞库',
      verifyBtn: '验证',
      valid: '✓ 有效',
      invalid: '✗ 无效',
      verifyEndpoint: '验证端点: {source}',
      applyCloudBefore: '去',
      applyCloudAfter: '免费申请',
      tokenPassHint1: 'opensca-ui 会作为',
      tokenPassHint2: '参数传给 CLI（v2.x/v3.x 均支持），无需手动配 config.json。',
      localDB: '本地漏洞库（db.json）',
      localDBPlaceholder: '可选，db.json 完整路径',
      localDBHint1: 'v3.x 通过',
      localDBHint2: '注入；opensca-ui 会自动生成临时 config.json（只填 origin.json）传给 CLI。',
      tokenEmpty: 'token 为空',
      tokenFillFirst: '请先填写 token'
    },
    runtime: {
      maxConcurrent: '最大并发扫描数',
      ioHint: '建议 2-4，过高会竞争 IO',
      theme: '界面主题',
      themeLight: '浅色',
      themeDark: '深色'
    },
    about: {
      version: 'OpenSCA UI v{v}',
      builtWith: '本软件是 opensca-cli 的桌面图形界面，基于 Wails + Vue 3 + Naive UI 构建。',
      configPath: '配置文件位置：',
      loadFailed: '获取失败',
      loading: '加载中…'
    },
    updateModal: {
      title: '检查 opensca-cli 更新',
      current: '当前',
      latest: '最新',
      unknown: '未知',
      hasUpdate: '有新版本',
      isLatest: '已是最新',
      changelog: '更新日志',
      downloadAsset: '下载资产：{name}',
      openRelease: '打开 release 页面',
      downloadReplace: '下载并替换',
      close: '关闭',
      backupHint: '提示：替换前会自动备份原文件为 opensca-cli.exe.bak，失败可手动恢复。',
      updatedTo: '已更新到 v{version}',
      backedUp: '（旧文件已备份到 {path}）',
      configCliFirst: '请先配置 opensca-cli 路径',
      noDownloadAsset: '没有可用的下载资产，请前往 release 页面手动下载',
      noReleaseLink: '没有 release 链接',
      missingTargetPath: '缺少目标路径'
    }
  },
  home: {
    subtitle: '一站式开源软件成分分析与漏洞检测工具',
    newScan: '新建扫描',
    history: '历史记录',
    settings: '设置',
    recentProjects: '最近打开的项目',
    clearAll: '清空',
    clearAllConfirm: '确认清空所有最近项目？此操作不可撤销。',
    emptyRecent: '还没有扫描过的项目',
    emptyRecentHint: '开始一次扫描后，最近的项目会自动出现在这里',
    scannedTimes: '扫描过 {n} 次',
    archived: '已存档 {n} 条',
    openProjectFolder: '打开项目目录',
    removeFromRecent: '从最近列表移除',
    removeConfirm: '从最近列表移除该路径？',
    removed: '已从最近列表移除',
    listCleared: '最近列表已清空',
    justNow: '刚刚',
    minAgo: '{n} 分钟前',
    hourAgo: '{n} 小时前',
    dayAgo: '{n} 天前'
  },
  scan: {
    title: '新建扫描',
    projectName: '项目名',
    projectNamePlaceholder: '选择路径后自动从文件夹名生成',
    projectNameTooltip: '项目名绑定该扫描所属项目，作为历史记录分组的依据。取自所选路径的最末段文件夹名（或压缩包去扩展名），无法编辑。',
    taskLabel: '任务标签（备注）',
    labelPlaceholder: '可选，例如：重构前扫描 / 回归验证',
    targetPath: '目标路径',
    start: '开始扫描',
    viewTasks: '查看任务',
    tokenHint: 'Token / 漏洞库 在设置中',
    recentTasks: '最近任务',
    noTasks: '还没有任务，启动一次扫描试试看',
    selectFirst: '请先选择项目目录',
    taskCreated: '扫描任务已创建'
  },
  tasks: {
    runningTab: '运行中 ({n})',
    finishedTab: '已完成 ({n})'
  },
  tasksRunning: {
    noRunning: '暂无运行中的任务',
    cancelSent: '取消请求已发送'
  },
  tasksFinished: {
    noFinished: '暂无已完成任务'
  },
  history: {
    noHistory: '暂无历史记录',
    entryCount: '历史 {n} 条'
  },
  tasksHistory: {
    back: '返回历史记录',
    open: '打开',
    noFolderHistory: '该文件夹暂无历史记录'
  },
  report: {
    refresh: '刷新',
    openReportDir: '打开报告目录',
    viewInBrowser: '在浏览器中查看',
    noReportPath: '无对应生成报告',
    htmlNotGenerated: 'HTML 报告未生成',
    parseFailed: '报告解析失败',
    cliWarningTitle: 'opensca-cli 报告提示',
    cliWarningHint: '建议：到设置页配置云 Token 或本地漏洞库，让 CLI 能匹配出真实漏洞数据。',
    tabVulns: '漏洞列表',
    tabComponents: '组件依赖',
    tabLogs: '实时日志',
    tabRaw: '原始 JSON',
    noReport: '暂无报告（任务可能尚未完成）',
    noReportSimple: '暂无报告',
    totalComponents: '共 {n} 个组件',
    vulnCountShort: '{n} 个漏洞',
    backHistory: '← 返回历史记录',
    backRunning: '← 返回运行中',
    backFinished: '← 返回已完成',
    backTasks: '← 返回任务列表'
  },
  welcome: {
    title: '欢迎使用 OpenSCA UI',
    subtitle: '在开始之前，请指定 opensca-cli 可执行文件的路径',
    cliPath: 'CLI 路径',
    browse: '浏览',
    verifyEnter: '验证并进入',
    verifyPassed: '验证通过',
    verifyFailed: '验证失败',
    version: '版本: {version}',
    cliOutput: 'CLI 输出',
    cliSuccess: 'CLI 验证成功，进入主界面',
    selectOrInputPath: '请先选择或输入路径',
    helpBefore: '还没下载 opensca-cli？去',
    helpAfter: '下载对应系统的版本。'
  },
  dropzone: {
    hint: '拖入项目目录或压缩包',
    or: '或者点击下方按钮选择',
    pickDir: '选择目录',
    pickZip: '选择压缩包',
    pastePath: '或粘贴路径：',
    use: '使用'
  },
  stattiles: {
    components: '组件',
    vulns: '漏洞'
  },
  logviewer: {
    autoScroll: '自动滚动',
    lines: '{n} 行',
    empty: '暂无日志'
  },
  taskcard: {
    cancel: '取消',
    viewReport: '查看报告',
    delete: '删除',
    finishedAt: '完成于 {time}',
    startedAt: '开始于 {time}',
    duration: '用时 {s}s'
  },
  vuln: {
    severityCol: '严重度',
    riskLevel: '风险等级',
    vulnId: '漏洞编号',
    cve: 'CVE',
    title: '标题',
    affectedComponent: '受影响组件',
    solution: '解决方案',
    name: '漏洞名称',
    releaseDate: '发布日期',
    exploitLevel: '利用难度',
    attackType: '攻击类型',
    description: '漏洞描述',
    suggestion: '修复建议',
    searchSimplePlaceholder: '搜索 漏洞编号 / 标题 / 组件',
    severityFilter: '按严重度筛选',
    searchPlaceholder: '搜索 漏洞名称 / 编号 / 描述 / 建议',
    riskFilter: '按风险等级筛选',
    counts: '共 {components} 个组件 / {vulns} 条漏洞',
    matchCount: '共 {n} 条',
    noMatch: '没有匹配的漏洞',
    direct: '直接依赖',
    indirect: '间接依赖',
    unknownLicense: '未知',
    sevCountCritical: '{n} 严重',
    sevCountHigh: '{n} 高危',
    sevCountMedium: '{n} 中危',
    sevCountLow: '{n} 低危',
    sevCountInfo: '{n} 提示',
    exploit: {
      easy: '容易',
      medium: '中等',
      hard: '困难',
      unknown: '未知'
    }
  },
  vulnDetail: {
    vulnId: '漏洞编号',
    severity: '严重度',
    cve: 'CVE 编号',
    cwe: 'CWE 编号',
    source: '漏洞来源',
    purl: 'PURL',
    desc: '漏洞描述',
    suggestion: '修复建议',
    refs: '参考链接',
    noRefs: '无参考链接',
    noDesc: '（无描述）',
    noSuggestion: '（无建议）'
  }
}
