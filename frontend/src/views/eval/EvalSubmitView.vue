<template>
  <div class="eval-submit">
    <PageHeader
      title="提交评测"
      description="配置模型、数据集与运行参数后创建评测任务"
    >
      <template #actions>
        <el-button @click="$router.push({ name: 'eval-task-list' })">
          返回任务列表
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
          <span>任务信息</span>
        </div>
        <el-form-item label="任务名称">
          <el-input
            v-model="form.taskName"
            maxlength="200"
            show-word-limit
            clearable
            placeholder="可选，用于列表搜索与识别；不填则仅显示短任务 ID"
          />
        </el-form-item>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-cpu" />
          <span>模型配置</span>
        </div>
        <el-form-item label="模型来源">
          <div class="model-source-row">
            <el-radio-group v-model="form.evalModelKind" @change="onEvalModelKindChange">
              <el-radio label="api">
                API 模型
                <span class="model-source-desc">（OpenAI 兼容远程接口）</span>
              </el-radio>
              <el-radio label="local" disabled class="model-source-disabled">
                本地模型
                <span class="model-source-desc">（本机 HuggingFace 权重）</span>
                <el-tooltip content="需直接加载本地权重与 logits，正在开发中" placement="top">
                  <i class="el-icon-question model-source-tip-icon" />
                </el-tooltip>
              </el-radio>
            </el-radio-group>
          </div>
          <p class="form-hint model-source-hint">
            API 模型仅展示与其兼容的评测数据集（一般为 GEN 生成式）；本地模型上线后将支持 PPL 等需 logits 的数据集。
          </p>
        </el-form-item>
        <el-form-item label="选择方式">
          <el-radio-group v-model="form.modelMode">
            <el-radio-button label="manual">手动填写</el-radio-button>
            <el-radio-button label="preset" :disabled="!modelPresets.length">
              使用预设模型
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.modelMode === 'preset'">
          <el-form-item label="预设模型" prop="modelPresetId">
            <el-select
              v-model="form.modelPresetId"
              filterable
              placeholder="选择已保存的模型预设"
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
          <el-form-item v-if="selectedPreset" label="预设详情">
            <el-descriptions :column="1" size="small" border>
              <el-descriptions-item label="模型名称">{{ selectedPreset.modelName }}</el-descriptions-item>
              <el-descriptions-item label="服务商">{{ selectedPreset.provider }}</el-descriptions-item>
              <el-descriptions-item label="Base URL">{{ selectedPreset.baseUrl || "-" }}</el-descriptions-item>
              <el-descriptions-item v-if="selectedPreset.version" label="版本">{{ selectedPreset.version }}</el-descriptions-item>
              <el-descriptions-item label="API Key">{{ selectedPreset.maskedKey }}</el-descriptions-item>
            </el-descriptions>
          </el-form-item>
          <el-alert
            v-if="!modelPresets.length"
            type="warning"
            :closable="false"
            title="尚未保存任何预设模型"
            description="请先到「模型管理」新增预设模型，或切换为手动填写。"
          />
        </template>

        <template v-else>
          <el-form-item label="模型服务商" prop="provider">
            <el-input
              v-model="form.provider"
              placeholder="例如 openai-compatible"
            />
          </el-form-item>
          <el-form-item label="模型名称" prop="modelName">
            <el-input
              v-model="form.modelName"
              placeholder="API 调用的 model 字段，例如 qwen-plus / gpt-4o-mini"
            />
          </el-form-item>
          <el-form-item label="备注名称">
            <el-input
              v-model="form.displayName"
              placeholder="可选。便于识别的别名"
            />
          </el-form-item>
          <el-form-item label="自定义版本">
            <el-input
              v-model="form.version"
              placeholder="可选。给同一模型测不同版本时区分"
            />
          </el-form-item>
          <el-form-item label="Base URL" prop="baseUrl">
            <el-input
              v-model="form.baseUrl"
              placeholder="https://api.example.com/v1"
            />
          </el-form-item>
          <el-form-item label="API Key" prop="apiKey">
            <el-input
              v-model="form.apiKey"
              type="password"
              show-password
              placeholder="仅本次评测使用，可选择保存"
            />
          </el-form-item>
          <el-form-item label="保存为预设">
            <el-switch v-model="form.saveModel" />
            <span class="form-hint">
              开启后将存入「模型管理」，下次可直接选用
            </span>
          </el-form-item>
        </template>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-collection" />
          <span>数据集</span>
          <el-button
            type="text"
            class="section-toggle"
            icon="el-icon-refresh"
            :loading="datasetsLoading"
            @click="loadDatasets"
          >
            刷新
          </el-button>
        </div>
        <el-form-item label="选择数据集" prop="datasetId">
          <el-select
            v-model="form.datasetId"
            filterable
            :placeholder="compatibleDatasets.length ? '从与当前模型来源匹配的数据集中选择' : '暂无匹配的数据集'"
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
                    {{ item.source === "builtin" ? "内置" : "自定义" }}
                  </el-tag>
                  <span v-if="item.sampleCount" class="dataset-option-count">{{ item.sampleCount }} 样本</span>
                </span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-if="selectedDataset" label="说明">
          <el-alert
            :title="selectedDataset.displayName"
            :description="`${selectedDataset.description || '无描述'}\nCode: ${selectedDataset.code}`"
            type="info"
            :closable="false"
            show-icon
          />
        </el-form-item>
        <el-form-item v-if="datasetIncompatible" label="">
          <el-alert
            type="error"
            :closable="false"
            title="该数据集与当前模型来源不兼容"
            description="所选数据集采用 PPL（困惑度）推理方式，需要直接读取模型 logits，仅适用于本地模型。请改选 GEN 类数据集，或待「本地模型」上线后再试。"
            show-icon
          />
        </el-form-item>
        <el-form-item label="附加参数">
          <KeyValueEditor v-model="form.params" placeholder-key="key" placeholder-value="value" />
        </el-form-item>
      </el-card>

      <el-card shadow="never" class="section-card">
        <div slot="header" class="section-title">
          <i class="el-icon-s-tools" />
          <span>运行参数</span>
          <el-button
            type="text"
            class="section-toggle"
            @click="runtimeOpen = !runtimeOpen"
          >
            {{ runtimeOpen ? "收起" : "展开" }}
            <i :class="runtimeOpen ? 'el-icon-arrow-up' : 'el-icon-arrow-down'" />
          </el-button>
        </div>
        <div v-show="runtimeOpen">
          <el-form-item label="超时（秒）">
            <el-input-number
              v-model="form.runtime.timeoutSeconds"
              :min="0"
              :step="60"
              controls-position="right"
            />
          </el-form-item>
          <el-form-item label="最大并发">
            <el-input-number
              v-model="form.runtime.maxWorkers"
              :min="1"
              :max="32"
              controls-position="right"
            />
          </el-form-item>
          <el-form-item label="保留原始输出">
            <el-switch v-model="form.runtime.keepRawOutputs" />
          </el-form-item>
        </div>
      </el-card>

      <div class="form-actions">
        <el-button @click="resetForm">重置</el-button>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="datasetIncompatible"
          icon="el-icon-s-promotion"
          @click="handleSubmit"
        >
          创建评测任务
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
      form: buildInitialForm(),
      rules: {
        provider: [{ required: true, message: "请填写服务商", trigger: "blur" }],
        modelName: [{ required: true, message: "请填写模型名称", trigger: "blur" }],
        baseUrl: [{ required: true, message: "请填写 Base URL", trigger: "blur" }],
        apiKey: [{ required: true, message: "请填写 API Key", trigger: "blur" }],
        modelPresetId: [{ required: true, message: "请选择预设模型", trigger: "change" }],
        datasetId: [{ required: true, message: "请选择数据集", trigger: "change" }]
      }
    };
  },
  computed: {
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
      if (this.datasetsLoading) return "加载中…";
      if (!this.datasets.length) return "暂无已启用数据集，请先在「数据集」中同步或启用";
      if (this.form.evalModelKind === "api") {
        return "当前无与 API 模型兼容的数据集（需 GEN 类）。可到数据集页确认推理方式或同步内置集";
      }
      return "暂无数据";
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
          this.$message.warning("该数据集与当前 API 模型来源不兼容，已跳过预填");
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
        this.$message.success("任务已创建");
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
