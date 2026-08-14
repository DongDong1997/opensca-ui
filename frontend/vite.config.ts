import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import {fileURLToPath, URL} from 'node:url'
import {readFileSync} from 'node:fs'

// 应用版本唯一来源：仓库根 wails.json 的 info.productVersion（与 build.ps1 同源，
// 发布 = 改 wails.json）。构建/开发时注入为全局常量 __APP_VERSION__，界面显示即时跟随。
function loadAppVersion(): string {
  try {
    const p = fileURLToPath(new URL('../wails.json', import.meta.url))
    const cfg = JSON.parse(readFileSync(p, 'utf8'))
    return cfg?.info?.productVersion || '0.0.0'
  } catch {
    return '0.0.0'
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  define: {
    __APP_VERSION__: JSON.stringify(loadAppVersion())
  }
})