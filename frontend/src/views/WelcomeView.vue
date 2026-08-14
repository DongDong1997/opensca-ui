<script setup lang="ts">
import {ref} from 'vue'
import {useRouter} from 'vue-router'
import {NButton, NCard, NIcon, NInput, NResult, NSpace, NText, useMessage} from 'naive-ui'
import {RocketOutline, FolderOpenOutline, CheckmarkCircleOutline, CloseCircleOutline} from '@vicons/ionicons5'
import {useConfigStore} from '@/stores/config'
import {api} from '@/api'

const router = useRouter()
const cfg = useConfigStore()
const message = useMessage()

const cliPath = ref(cfg.cliPath)
const verifying = ref(false)
const lastResult = ref<{valid: boolean; version: string; message: string} | null>(null)

async function pick() {
  try {
    const p = await api.PickExecutable()
    if (p) {
      cliPath.value = p
      lastResult.value = null
    }
  } catch (e) {
    message.error(`选择失败: ${String(e)}`)
  }
}

async function verify() {
  if (!cliPath.value.trim()) {
    message.warning('请先选择或输入路径')
    return
  }
  verifying.value = true
  try {
    // 必须先 setCliPath（写入后端 cfg.CLIPath 并触发 CheckCli），否则后端 Submit 会拒绝
    const info = await cfg.setCliPath(cliPath.value.trim())
    lastResult.value = {valid: info.valid, version: info.version, message: info.message}
    if (info.valid) {
      message.success('CLI 验证成功，进入主界面')
      await new Promise((r) => setTimeout(r, 400))
      router.push('/scan')
    }
  } catch (e) {
    lastResult.value = {valid: false, version: '', message: String(e)}
    message.error('验证失败')
  } finally {
    verifying.value = false
  }
}
</script>

<template>
  <div class="welcome">
    <NCard class="welcome-card">
      <div class="welcome-hero">
        <NIcon :size="48" color="#18a058" :component="RocketOutline" />
        <h1>欢迎使用 OpenSCA UI</h1>
        <NText depth="3">在开始之前，请指定 opensca-cli 可执行文件的路径</NText>
      </div>

      <div class="welcome-form">
        <NText strong>CLI 路径</NText>
        <NSpace :size="8" style="margin-top: 8px">
          <NInput v-model:value="cliPath" placeholder="C:\\path\\to\\opensca-cli.exe" style="width: 480px" />
          <NButton @click="pick">
            <template #icon><NIcon :component="FolderOpenOutline" /></template>
            浏览
          </NButton>
        </NSpace>
        <NSpace style="margin-top: 16px">
          <NButton type="primary" :loading="verifying" :disabled="!cliPath.trim()" @click="verify">
            验证并进入
          </NButton>
        </NSpace>

        <div v-if="lastResult" class="result">
          <NResult
            v-if="lastResult.valid"
            status="success"
            title="验证通过"
            :description="`版本: ${lastResult.version || 'unknown'}`"
          >
            <template #icon>
              <NIcon :size="48" color="#18a058" :component="CheckmarkCircleOutline" />
            </template>
          </NResult>
          <NResult
            v-else
            status="error"
            title="验证失败"
            :description="lastResult.message"
          >
            <template #icon>
              <NIcon :size="48" color="#d03050" :component="CloseCircleOutline" />
            </template>
          </NResult>
          <!-- 调试：把 CLI 的真实 stdout 暴露出来，便于排错版本识别问题 -->
          <NInput
            v-if="lastResult && (lastResult as any).rawOutput"
            type="textarea"
            readonly
            :value="(lastResult as any).rawOutput"
            placeholder="CLI 输出"
            :autosize="{minRows: 3, maxRows: 8}"
            style="margin-top: 12px; font-family: var(--n-font-family-mono); font-size: 12px"
          />
        </div>
      </div>

      <div class="welcome-help">
        <NText depth="3" style="font-size: 12px">
          还没下载 opensca-cli？去
          <a href="https://github.com/XmirrorSecurity/OpenSCA-cli/releases" target="_blank" rel="noopener">GitHub Releases</a>
          下载对应系统的版本。
        </NText>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.welcome {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, var(--n-primary-color-hover) 0%, var(--n-card-color) 100%);
}
.welcome-card {
  max-width: 720px;
  width: 100%;
}
.welcome-hero {
  text-align: center;
  margin-bottom: 32px;
}
.welcome-hero h1 {
  margin: 12px 0 8px;
  font-size: 24px;
}
.welcome-form {
  margin-bottom: 24px;
}
.result {
  margin-top: 24px;
}
.welcome-help {
  border-top: 1px solid var(--n-border-color);
  padding-top: 16px;
  text-align: center;
}
.welcome-help a {
  color: var(--n-primary-color);
}
</style>