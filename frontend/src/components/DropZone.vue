<script setup lang="ts">
import {ref} from 'vue'
import {NButton, NIcon, NText, NSpace, NUpload, NUploadDragger, useMessage} from 'naive-ui'
import {FolderOpenOutline, CloudUploadOutline, FileTrayFullOutline} from '@vicons/ionicons5'
import {api} from '@/api'

const emit = defineEmits<{
  (e: 'selected', path: string): void
}>()

const message = useMessage()
const dragOver = ref(false)
const manualPath = ref('')

async function onPickDir() {
  try {
    const p = await api.PickDirectory()
    if (p) emit('selected', p)
  } catch (e) {
    message.error(`选择目录失败: ${String(e)}`)
  }
}

async function onPickZip() {
  try {
    const p = await api.PickZip()
    if (p) emit('selected', p)
  } catch (e) {
    message.error(`选择文件失败: ${String(e)}`)
  }
}

function onManualSubmit() {
  if (manualPath.value.trim()) emit('selected', manualPath.value.trim())
}

// Wails 文件拖拽（Wails 启用后会自动接管浏览器拖拽事件）
function onDragOver(e: DragEvent) {
  e.preventDefault()
  dragOver.value = true
}
function onDragLeave() {
  dragOver.value = false
}
function onDrop(e: DragEvent) {
  e.preventDefault()
  dragOver.value = false
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  const first = files[0]
  // @ts-expect-error webkitRelativePath / path 在某些环境可能存在
  const p = (first.path || first.webkitRelativePath || first.name) as string
  if (p) emit('selected', p)
}
</script>

<template>
  <div
    class="drop-zone"
    :class="{over: dragOver}"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <NIcon :size="48" :component="FileTrayFullOutline" class="drop-icon" />
    <NText style="font-size: 16px; margin-bottom: 8px">拖入项目目录或压缩包</NText>
    <NText depth="3" style="margin-bottom: 16px">或者点击下方按钮选择</NText>

    <NSpace>
      <NButton type="primary" @click="onPickDir">
        <template #icon><NIcon :component="FolderOpenOutline" /></template>
        选择目录
      </NButton>
      <NButton @click="onPickZip">
        <template #icon><NIcon :component="CloudUploadOutline" /></template>
        选择压缩包
      </NButton>
    </NSpace>

    <div class="manual-row">
      <NText depth="3" style="font-size: 12px">或粘贴路径：</NText>
      <NUpload :default-upload="false" :show-file-list="false" />
      <div class="manual-input">
        <input
          v-model="manualPath"
          class="path-input"
          placeholder="C:\\path\\to\\project"
          @keydown.enter="onManualSubmit"
        />
        <NButton size="small" @click="onManualSubmit">使用</NButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drop-zone {
  border: 2px dashed var(--n-border-color);
  border-radius: 12px;
  padding: 40px 24px;
  text-align: center;
  transition: all 0.15s;
  background: var(--n-card-color);
}
.drop-zone.over {
  border-color: var(--n-primary-color);
  background: var(--n-primary-color-hover);
}
.drop-icon {
  color: var(--n-primary-color);
  margin-bottom: 12px;
}
.manual-row {
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.manual-input {
  display: flex;
  gap: 8px;
  width: 100%;
  max-width: 480px;
}
.path-input {
  flex: 1;
  padding: 6px 10px;
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  background: var(--n-color);
  color: var(--n-text-color);
  font-family: 'Fira Code', Menlo, monospace;
  font-size: 13px;
  outline: none;
}
.path-input:focus {
  border-color: var(--n-primary-color);
}
</style>