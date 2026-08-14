/// <reference types="vite/client" />

declare module '*.vue' {
    import type {DefineComponent} from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}

// 由 vite.config.ts 的 define 注入：来自仓库根 wails.json 的 info.productVersion
declare const __APP_VERSION__: string
