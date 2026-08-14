<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem, NTag, NText, NSpace, NDivider, NAnchor, NAnchorLink, NEmpty} from 'naive-ui'
import SeverityTag from './SeverityTag.vue'
import type {Vuln, Severity} from '@/api/types'

const props = defineProps<{show: boolean; vuln: Vuln | null}>()
const emit = defineEmits<{(e: 'update:show', v: boolean): void}>()

const {t} = useI18n()

const visible = computed({
  get: () => props.show,
  set: (v) => emit('update:show', v)
})
</script>

<template>
  <NDrawer v-model:show="visible" :width="720" placement="right">
    <NDrawerContent v-if="vuln" :title="vuln.title || vuln.id" closable>
      <template #header>
        <NSpace align="center">
          <SeverityTag :severity="(vuln.severity as Severity)" size="medium" />
          <NText strong style="font-size: 16px">{{ vuln.title || vuln.id }}</NText>
        </NSpace>
      </template>

      <NDescriptions bordered :column="2" label-placement="top" size="small">
        <NDescriptionsItem :label="t('vulnDetail.vulnId')">
          <NText code>{{ vuln.id || '-' }}</NText>
        </NDescriptionsItem>
        <NDescriptionsItem :label="t('vulnDetail.severity')">
          <SeverityTag :severity="(vuln.severity as Severity)" />
        </NDescriptionsItem>
        <NDescriptionsItem :label="t('vulnDetail.cve')">
          <NSpace>
            <NTag v-for="c in vuln.cve" :key="c" size="small" type="info">{{ c }}</NTag>
            <NText v-if="!vuln.cve?.length" depth="3">-</NText>
          </NSpace>
        </NDescriptionsItem>
        <NDescriptionsItem :label="t('vulnDetail.cwe')">
          <NSpace>
            <NTag v-for="c in vuln.cwe" :key="c" size="small">{{ c }}</NTag>
            <NText v-if="!vuln.cwe?.length" depth="3">-</NText>
          </NSpace>
        </NDescriptionsItem>
        <NDescriptionsItem :label="t('vulnDetail.source')">
          <NText depth="3">{{ vuln.source || '-' }}</NText>
        </NDescriptionsItem>
        <NDescriptionsItem :label="t('vulnDetail.purl')">
          <NText code style="word-break: break-all">{{ vuln.purl || '-' }}</NText>
        </NDescriptionsItem>
      </NDescriptions>

      <NDivider />

      <NAnchor>
        <NAnchorLink :title="t('vulnDetail.desc')" href="#desc" />
        <NAnchorLink :title="t('vulnDetail.suggestion')" href="#fix" />
        <NAnchorLink :title="t('vulnDetail.refs')" href="#refs" />
      </NAnchor>

      <div id="desc" style="margin-top: 16px">
        <NText strong style="font-size: 14px">{{ t('vulnDetail.desc') }}</NText>
        <div style="margin-top: 8px; white-space: pre-wrap; line-height: 1.7">
          {{ vuln.description || t('vulnDetail.noDesc') }}
        </div>
      </div>

      <div id="fix" style="margin-top: 24px">
        <NText strong style="font-size: 14px">{{ t('vulnDetail.suggestion') }}</NText>
        <div style="margin-top: 8px; white-space: pre-wrap; line-height: 1.7">
          {{ vuln.suggestion || t('vulnDetail.noSuggestion') }}
        </div>
      </div>

      <div id="refs" style="margin-top: 24px">
        <NText strong style="font-size: 14px">{{ t('vulnDetail.refs') }}</NText>
        <NEmpty v-if="!vuln.references?.length" size="small" :description="t('vulnDetail.noRefs')" style="margin-top: 12px" />
        <ul v-else style="margin: 8px 0 0; padding-left: 20px; line-height: 1.8">
          <li v-for="r in vuln.references" :key="r">
            <a :href="r" target="_blank" rel="noopener" style="color: var(--n-primary-color)">{{ r }}</a>
          </li>
        </ul>
      </div>
    </NDrawerContent>
  </NDrawer>
</template>
