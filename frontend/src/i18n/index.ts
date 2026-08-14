import {createI18n} from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'
import type zhCNMessages from './locales/zh-CN'

export const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const
export type Language = (typeof SUPPORTED_LOCALES)[number]

export function isSupportedLanguage(v: unknown): v is Language {
  return typeof v === 'string' && (SUPPORTED_LOCALES as readonly string[]).includes(v)
}

export const i18n = createI18n({
  legacy: false, // Composition API —— <script setup> 里 useI18n() 必需
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {'zh-CN': zhCN, 'en-US': enUS}
})

/** 设置全局语言（非法值回退中文）。供启动加载、路由守卫、设置页调用。 */
export function applyLanguage(lang: string) {
  i18n.global.locale.value = isSupportedLanguage(lang) ? lang : 'zh-CN'
}

// 让 t('...') 的键在编译期校验（拼错直接报错），以 zh-CN 为 schema 基准。
export type MessageSchema = typeof zhCNMessages
declare module 'vue-i18n' {
  export interface DefineLocaleMessage extends MessageSchema {}
}
