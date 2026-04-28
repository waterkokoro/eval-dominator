<template>
  <div class="eval-submit">
    <PageHeader
      :title="$t('eval.submit.title')"
      :description="$t('eval.submit.description')"
    >
      <template #actions>
        <el-button @click="$router.push({ name: 'eval-task-list' })">
          {{ $t("eval.submit.back") }}
        </el-button>
      </template>
    </PageHeader>

    <el-form
      ref="form"
      :model="form"
      :rules="rules"
      label-width="120px"
      class="eval-form"
    >
      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-document" />
          <span>{{ $t("eval.submit.section.task") }}</span>
        </div>
        <el-form-item :label="$t('eval.submit.task.name')">
          <el-input
            v-model="form.taskName"
            maxlength="200"
            show-word-limit
            clearable
            :placeholder="$t('eval.submit.task.namePlaceholder')"
          />
        </el-form-item>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-cpu" />
          <span>{{ $t("eval.submit.section.model") }}</span>
        </div>
        <el-form-item :label="$t('eval.submit.model.source')">
          <div class="model-source-row">
            <el-radio-group v-model="form.evalModelKind" @change="onEvalModelKindChange">
              <el-radio label="api">
                {{ $t("eval.submit.model.kindApi") }}
                <span class="model-source-desc">{{ $t("eval.submit.model.kindApiDesc") }}</span>
              </el-radio>
              <el-radio label="local" disabled class="model-source-disabled">
                {{ $t("eval.submit.model.kindLocal") }}
                <span class="model-source-desc">{{ $t("eval.submit.model.kindLocalDesc") }}</span>
                <el-tooltip :content="$t('eval.submit.model.kindLocalTip')" placement="top">
                  <i class="el-icon-question model-source-tip-icon" />
                </el-tooltip>
              </el-radio>
            </el-radio-group>
          </div>
          <p class="form-hint model-source-hint">
            {{ $t("eval.submit.model.sourceHint") }}
          </p>
        </el-form-item>
        <el-form-item :label="$t('eval.submit.model.mode')">
          <el-radio-group v-model="form.modelMode">
            <el-radio-button label="manual">{{ $t("eval.submit.model.modeManual") }}</el-radio-button>
            <el-radio-button label="preset" :disabled="!modelPresets.length">
              {{ $t("eval.submit.model.modePreset") }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.modelMode === 'preset'">
          <el-form-item :label="$t('eval.submit.model.preset')" prop="modelPresetId">
            <el-select
              v-model="form.modelPresetId"
              filterable
              :placeholder="$t('eval.submit.model.presetPlaceholder')"
              style="width: 100%"
              @change="onModelPresetChange"
            >
              <el-option
                v-for="item in modelPresets"
                :key="item.id"
                :value="item.id"
                :label="`${item.displayName || item.modelName} · ${item.modelName}${item.version ? ' v' + item.version : ''} (${item.maskedKey})`"
              >
                <div class="preset-option">
                  <span>{{ item.displayName || item.modelName }}</span>
                  <span class="preset-option-meta">
                    <span class="preset-option-model">{{ item.modelName }}</span>
                    <el-tag v-if="item.version" size="mini" type="warning">v{{ item.version }}</el-tag>
                  </span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="selectedPreset" :label="$t('eval.submit.model.presetDetail')">
            <el-descriptions :column="1" size="small" border>
              <el-descriptions-item :label="$t('eval.submit.model.presetFields.modelName')">
                {{ selectedPreset.modelName }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('eval.submit.model.presetFields.provider')">
                {{ selectedPreset.provider }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('eval.submit.model.presetFields.baseUrl')">
                {{ selectedPreset.baseUrl || "-" }}
              </el-descriptions-item>
              <el-descriptions-item v-if="selectedPreset.version" :label="$t('eval.submit.model.presetFields.version')">
                {{ selectedPreset.version }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('eval.submit.model.presetFields.apiKey')">
                {{ selectedPreset.maskedKey }}
              </el-descriptions-item>
            </el-descriptions>
          </el-form-item>
          <el-alert
            v-if="!modelPresets.length"
            type="warning"
            :closable="false"
            :title="$t('eval.submit.model.presetEmptyTitle')"
            :description="$t('eval.submit.model.presetEmptyDescription')"
          />
        </template>

        <template v-else>
          <el-form-item :label="$t('eval.submit.model.provider')" prop="provider">
            <el-input
              v-model="form.provider"
              :placeholder="$t('eval.submit.model.providerPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.modelName')" prop="modelName">
            <el-input
              v-model="form.modelName"
              :placeholder="$t('eval.submit.model.modelNamePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.displayName')">
            <el-input
              v-model="form.displayName"
              :placeholder="$t('eval.submit.model.displayNamePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.version')">
            <el-input
              v-model="form.version"
              :placeholder="$t('eval.submit.model.versionPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.baseUrl')" prop="baseUrl">
            <el-input
              v-model="form.baseUrl"
              :placeholder="$t('eval.submit.model.baseUrlPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.apiKey')" prop="apiKey">
            <el-input
              v-model="form.apiKey"
              type="password"
              show-password
              :placeholder="$t('eval.submit.model.apiKeyPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.model.saveAsPreset')">
            <el-switch v-model="form.saveModel" />
            <span class="form-hint">
              {{ $t("eval.submit.model.saveAsPresetHint") }}
            </span>
          </el-form-item>
        </template>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-collection" />
          <span>{{ $t("eval.submit.section.dataset") }}</span>
          <el-button
            type="text"
            class="section-toggle"
            icon="el-icon-refresh"
            :loading="datasetsLoading"
            @click="loadDatasets"
          >
            {{ $t("common.actions.refresh") }}
          </el-button>
        </div>
        <el-form-item :label="$t('eval.submit.dataset.select')" prop="datasetId">
          <el-select
            v-model="form.datasetId"
            filterable
            :placeholder="compatibleDatasets.length ? $t('eval.submit.dataset.selectPlaceholder') : $t('eval.submit.dataset.selectPlaceholderEmpty')"
            style="width: 100%"
            :loading="datasetsLoading"
            :no-data-text="compatibleDatasetsEmptyHint"
            @change="onDatasetChange"
          >
            <el-option
              v-for="item in compatibleDatasets"
              :key="item.id"
              :value="item.id"
              :label="datasetOptionLabel(item)"
            >
              <div class="dataset-option">
                <span>{{ item.displayName }}</span>
                <span class="dataset-option-meta">
                  <el-tag size="mini" :type="inferenceModeTagType(item.inferenceMode)">
                    {{ inferenceModeLabel(item.inferenceMode) }}
                  </el-tag>
                  <el-tag size="mini" :type="item.source === 'builtin' ? 'info' : 'warning'">
                    {{ $t(`dataset.source.${item.source}`) }}
                  </el-tag>
                  <span v-if="item.sampleCount" class="dataset-option-count">
                    {{ $t("eval.submit.dataset.samples", { count: item.sampleCount }) }}
                  </span>
                </span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-if="selectedDataset" :label="$t('eval.submit.dataset.description')">
          <el-alert
            :title="selectedDataset.displayName"
            :description="`${selectedDataset.description || $t('eval.submit.dataset.noDescription')}\nCode: ${selectedDataset.code}`"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form-item>
        <el-form-item v-if="datasetIncompatible" label="">
          <el-alert
            type="error"
            :closable="false"
            :title="$t('eval.submit.dataset.incompatibleTitle')"
            :description="$t('eval.submit.dataset.incompatibleDescription')"
            show-icon
          />
        </el-form-item>
        <el-form-item :label="$t('eval.submit.dataset.params')">
          <KeyValueEditor v-model="form.params" :placeholder-key="$t('common.placeholders.key')" :placeholder-value="$t('common.placeholders.value')" />
        </el-form-item>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-s-tools" />
          <span>{{ $t("eval.submit.section.runtime") }}</span>
          <el-button
            type="text"
            class="section-toggle"
            @click="runtimeOpen = !runtimeOpen"
          >
            {{ runtimeOpen ? $t("common.actions.collapse") : $t("common.actions.expand") }}
            <i :class="runtimeOpen ? 'el-icon-arrow-up' : 'el-icon-arrow-down'" />
          </el-button>
        </div>
        <div v-show="runtimeOpen">
          <el-form-item :label="$t('eval.submit.runtime.timeout')">
            <el-input-number
              v-model="form.runtime.timeoutSeconds"
              :min="0"
              :step="60"
              controls-position="right"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.runtime.maxWorkers')">
            <el-input-number
              v-model="form.runtime.maxWorkers"
              :min="1"
              :max="32"
              controls-position="right"
            />
          </el-form-item>
          <el-form-item :label="$t('eval.submit.runtime.keepRawOutputs')">
            <el-switch v-model="form.runtime.keepRawOutputs" />
          </el-form-item>
        </div>
      </el-card>

      <div class="form-actions">
        <el-button @click="resetForm">{{ $t("eval.submit.footer.reset") }}</el-button>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="datasetIncompatible"
          icon="el-icon-s-promotion"
          @click="handleSubmit"
        >
          {{ $t("eval.submit.footer.create") }}
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import KeyValueEditor from "@/components/KeyValueEditor.vue";

import { createEvalTask } from "@/api/eval-task";
import { listModels } from "@/api/model";
import { listDatasets } from "@/api/dataset";

const buildInitialForm = () => ({
  taskName: "",
  evalModelKind: "api",
  modelMode: "manual",
  modelPresetId: null,
  provider: "openai-compatible",
  modelName: "",
  displayName: "",
  version: "",
  baseUrl: "",
  apiKey: "",
  saveModel: false,
  datasetId: null,
  datasetType: "opencompass_demo",
  datasetName: "",
  params: {},
  runtime: {
    timeoutSeconds: 1800,
    maxWorkers: 4,
    keepRawOutputs: true
  }
});

export default {
  name: "EvalSubmitView",
  components: { PageHeader, KeyValueEditor },
  data() {
    return {
      loading: false,
      runtimeOpen: false,
      modelPresets: [],
      datasets: [],
      datasetsLoading: false,
      form: buildInitialForm()
    };
  },
  computed: {
    rules() {
      const t = (k) => this.$t(`eval.submit.model.rules.${k}`);
      return {
        provider: [{ required: true, message: t("providerRequired"), trigger: "blur" }],
        modelName: [{ required: true, message: t("modelNameRequired"), trigger: "blur" }],
        baseUrl: [{ required: true, message: t("baseUrlRequired"), trigger: "blur" }],
        apiKey: [{ required: true, message: t("apiKeyRequired"), trigger: "blur" }],
        modelPresetId: [{ required: true, message: t("presetRequired"), trigger: "change" }],
        datasetId: [{ required: true, message: this.$t("eval.submit.dataset.datasetRequired"), trigger: "change" }]
      };
    },
    selectedPreset() {
      return this.modelPresets.find((p) => p.id === this.form.modelPresetId) || null;
    },
    selectedDataset() {
      return this.datasets.find((d) => d.id === this.form.datasetId) || null;
    },
    /** 与当前「模型来源」匹配的数据集（API：排除 PPL；本地：全部，待功能开放） */
    compatibleDatasets() {
      return this.datasets.filter((d) => this.datasetMatchesEvalModelKind(d));
    },
    compatibleDatasetsEmptyHint() {
      if (this.datasetsLoading) return this.$t("eval.submit.dataset.loading");
      if (!this.datasets.length) return this.$t("eval.submit.dataset.emptyAll");
      if (this.form.evalModelKind === "api") {
        return this.$t("eval.submit.dataset.emptyApi");
      }
      return this.$t("eval.submit.dataset.noData");
    },
    datasetIncompatible() {
      const ds = this.selectedDataset;
      if (!ds) return false;
      if (this.form.evalModelKind === "api") {
        return ds.inferenceMode === "ppl";
      }
      return false;
    }
  },
  watch: {
    compatibleDatasets() {
      this.$nextTick(() => this.pruneInvalidDatasetSelection());
    }
  },
  created() {
    this.loadDatasets();
    this.loadModelPresets();
    this.$nextTick(() => this.applyPrefill());
  },
  methods: {
    datasetMatchesEvalModelKind(d) {
      if (!d) return false;
      if (this.form.evalModelKind === "local") {
        return true;
      }
      // API 模型：与后端 domain.Dataset.IsRemoteCompatible 一致
      return d.inferenceMode !== "ppl";
    },
    inferenceModeLabel(mode) {
      if (mode === "ppl") return "PPL";
      if (mode === "gen") return "GEN";
      return "GEN";
    },
    inferenceModeTagType(mode) {
      if (mode === "ppl") return "warning";
      return "success";
    },
    datasetOptionLabel(item) {
      const tag = this.inferenceModeLabel(item.inferenceMode);
      return `${item.displayName} (${item.code}) · ${tag}`;
    },
    onEvalModelKindChange() {
      this.pruneInvalidDatasetSelection();
    },
    pruneInvalidDatasetSelection() {
      if (!this.form.datasetId) return;
      const ok = this.compatibleDatasets.some((d) => d.id === this.form.datasetId);
      if (ok) return;
      this.form.datasetId = null;
      this.form.datasetType = "opencompass_demo";
      this.form.datasetName = "";
      this.$refs.form?.clearValidate(["datasetId"]);
    },
    applyPrefill() {
      const prefill = this.$route.query;
      if (!prefill) return;
      if (prefill.provider) this.form.provider = prefill.provider;
      if (prefill.modelName) this.form.modelName = prefill.modelName;
      if (prefill.baseUrl) this.form.baseUrl = prefill.baseUrl;
      if (prefill.datasetName) {
        const target = this.datasets.find((d) => d.code === prefill.datasetName);
        if (target && this.datasetMatchesEvalModelKind(target)) {
          this.form.datasetId = target.id;
          this.onDatasetChange(target.id);
        } else if (target && !this.datasetMatchesEvalModelKind(target)) {
          this.$message.warning(this.$t("eval.submit.dataset.incompatibleSkip"));
        }
      }
    },
    async loadModelPresets() {
      try {
        const list = await listModels();
        this.modelPresets = Array.isArray(list) ? list : list?.items || [];
      } catch (e) {
        this.modelPresets = [];
      }
    },
    async loadDatasets() {
      this.datasetsLoading = true;
      try {
        const data = await listDatasets(false);
        const items = Array.isArray(data) ? data : data?.items || [];
        this.datasets = items;
        if (this.form.datasetId) {
          const stillExists = items.some((d) => d.id === this.form.datasetId);
          if (!stillExists) {
            this.form.datasetId = null;
            this.form.datasetType = "opencompass_demo";
            this.form.datasetName = "";
          }
        }
        if (!this.form.datasetId && this.$route.query?.datasetName) {
          this.applyPrefill();
        }
        this.pruneInvalidDatasetSelection();
      } catch (e) {
        this.datasets = [];
      } finally {
        this.datasetsLoading = false;
      }
    },
    onDatasetChange(datasetId) {
      const target = this.datasets.find((d) => d.id === datasetId);
      if (!target) return;
      this.form.datasetType = target.type || "opencompass_demo";
      this.form.datasetName = target.code;
    },
    onModelPresetChange() {
      this.$refs.form?.clearValidate(["modelPresetId"]);
    },
    resetForm() {
      this.form = buildInitialForm();
      this.$refs.form?.clearValidate();
    },
    buildPayload() {
      const dataset = this.selectedDataset;
      const name = (this.form.taskName && String(this.form.taskName).trim()) || "";
      const payload = {
        taskName: name,
        datasetId: this.form.datasetId,
        datasetType: dataset?.type || this.form.datasetType,
        datasetName: dataset?.code || this.form.datasetName,
        params: this.form.params || {},
        runtime: { ...this.form.runtime }
      };
      if (this.form.modelMode === "preset" && this.form.modelPresetId) {
        payload.modelPresetId = this.form.modelPresetId;
        payload.saveModel = false;
      } else {
        payload.provider = this.form.provider;
        payload.modelName = this.form.modelName;
        payload.displayName = this.form.displayName;
        payload.version = this.form.version;
        payload.baseUrl = this.form.baseUrl;
        payload.apiKey = this.form.apiKey;
        payload.saveModel = this.form.saveModel;
      }
      return payload;
    },
    async handleSubmit() {
      const valid = await this.$refs.form.validate().catch(() => false);
      if (!valid) return;
      this.loading = true;
      try {
        const response = await createEvalTask(this.buildPayload());
        this.$message.success(this.$t("eval.submit.footer.createSuccess"));
        this.$router.push({
          name: "eval-task-detail",
          params: { evalTaskId: response.evalTaskId }
        });
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style scoped>
.eval-submit {
  max-width: 960px;
}

.eval-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-card >>> .el-card__header {
  padding: 12px 18px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.section-toggle {
  margin-left: auto;
}

.form-hint {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.dataset-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.dataset-option-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.dataset-option-count {
  font-size: 12px;
  color: #909399;
}

.preset-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.preset-option-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.preset-option-model {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #909399;
}

.model-source-row {
  line-height: 1.6;
}
.model-source-row >>> .el-radio {
  display: block;
  margin-bottom: 8px;
  margin-right: 0;
  white-space: normal;
  line-height: 1.5;
}
.model-source-row >>> .el-radio:last-child {
  margin-bottom: 0;
}
.model-source-desc {
  font-size: 12px;
  color: #909399;
  font-weight: normal;
}
.model-source-disabled {
  color: #c0c4cc;
}
.model-source-disabled .model-source-desc {
  color: #c0c4cc;
}
.model-source-tip-icon {
  margin-left: 4px;
  color: #c0c4cc;
  cursor: help;
  vertical-align: middle;
}
.model-source-hint {
  margin: 0;
  line-height: 1.5;
  max-width: 720px;
}
</style>
