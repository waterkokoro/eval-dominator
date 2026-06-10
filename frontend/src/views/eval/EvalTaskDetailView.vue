<template>
  <div class="task-detail">
    <PageHeader
      :title="$t('eval.detail.title')"
      :description="pageDescription"
    >
      <template #actions>
        <el-button @click="$router.push({ name: 'eval-task-list' })">
          {{ $t("eval.detail.back") }}
        </el-button>
        <el-button
          icon="el-icon-refresh"
          :loading="loading"
          @click="loadAll"
        >
          {{ $t("common.actions.refresh") }}
        </el-button>
        <el-button
          v-if="task && canCancel"
          type="danger"
          plain
          icon="el-icon-circle-close"
          :loading="cancelling"
          @click="handleCancel"
        >
          {{ $t("eval.detail.cancel") }}
        </el-button>
        <el-button
          v-if="task && canRerunEval"
          type="warning"
          plain
          icon="el-icon-refresh-right"
          :loading="rerunningEval"
          @click="handleRerunEval"
        >
          {{ $t("eval.detail.rerunEval") }}
        </el-button>
        <el-button
          type="primary"
          icon="el-icon-magic-stick"
          @click="rerun"
          :disabled="!task"
        >
          {{ $t("eval.detail.rerun") }}
        </el-button>
      </template>
    </PageHeader>

    <el-card v-if="task" shadow="never" class="summary-card">
        <div class="summary-head">
        <div class="summary-status">
          <StatusTag :status="task.status" />
          <span v-if="task.taskName" class="task-title">{{ task.taskName }}</span>
          <span class="task-id">{{ task.evalTaskId }}</span>
        </div>
        <div class="summary-meta">
          <div v-if="task.modelName">
            <i class="el-icon-cpu" /> {{ task.modelName }}
          </div>
          <div v-if="task.datasetName">
            <i class="el-icon-collection" /> {{ task.datasetName }}
          </div>
        </div>
      </div>

      <div class="summary-progress">
        <EvalSteps
          :status="task.status"
          :progress="task.progress"
          :progress-text="task.progressText"
          :running-phase="task.runningPhase"
        />
      </div>

      <el-alert
        v-if="task.errorMessage"
        type="error"
        :closable="false"
        :title="task.errorCode || $t('eval.detail.errorTitle')"
        :description="task.errorMessage"
        class="error-alert"
        show-icon
      />
    </el-card>

    <el-card shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane :label="$t('eval.detail.tabs.overview')" name="overview">
          <div v-if="task" class="overview-container">
            <!-- 任务信息 -->
            <div class="overview-section overview-section--task">
              <div class="overview-section-title">
                <i class="el-icon-document" />
                {{ $t('eval.detail.overview.sectionTask') }}
              </div>
              <div class="overview-items">
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.taskName') }}</span>
                  <span class="overview-value">{{ task.taskName || "—" }}</span>
                </div>
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.taskId') }}</span>
                  <span class="overview-value overview-value--mono">{{ task.evalTaskId }}</span>
                </div>
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.status') }}</span>
                  <span class="overview-value"><StatusTag :status="task.status" /></span>
                </div>
              </div>
            </div>

            <!-- 模型配置 -->
            <div class="overview-section overview-section--model">
              <div class="overview-section-title">
                <i class="el-icon-cpu" />
                {{ $t('eval.detail.overview.sectionModel') }}
              </div>
              <div class="overview-items">
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.modelProvider') }}</span>
                  <span class="overview-value">{{ task.modelProvider || "—" }}</span>
                </div>
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.modelName') }}</span>
                  <span class="overview-value">{{ task.modelName || "—" }}</span>
                </div>
                <div class="overview-item overview-item--full">
                  <span class="overview-label">{{ $t('eval.detail.overview.baseUrl') }}</span>
                  <span class="overview-value overview-value--mono">{{ task.modelBaseUrl || "—" }}</span>
                </div>
              </div>
            </div>

            <!-- 数据集 -->
            <div class="overview-section overview-section--dataset">
              <div class="overview-section-title">
                <i class="el-icon-collection" />
                {{ $t('eval.detail.overview.sectionDataset') }}
              </div>
              <div class="overview-items">
                <div class="overview-item overview-item--full">
                  <span class="overview-label">{{ $t('eval.detail.overview.dataset') }}</span>
                  <span class="overview-value">{{ task.datasetName || "—" }} <span v-if="datasetText" class="overview-sub">({{ datasetText }})</span></span>
                </div>
              </div>
            </div>

            <!-- 时间线 -->
            <div class="overview-section overview-section--timeline">
              <div class="overview-section-title">
                <i class="el-icon-time" />
                {{ $t('eval.detail.overview.sectionTimeline') }}
              </div>
              <div class="overview-items">
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.createdAt') }}</span>
                  <span class="overview-value">{{ formatTime(task.createdAt) }}</span>
                </div>
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.startedAt') }}</span>
                  <span class="overview-value">{{ formatTime(task.startedAt) }}</span>
                </div>
                <div class="overview-item">
                  <span class="overview-label">{{ $t('eval.detail.overview.finishedAt') }}</span>
                  <span class="overview-value">{{ formatTime(task.finishedAt) }}</span>
                </div>
              </div>
            </div>

            <!-- 错误信息 -->
            <div v-if="task.errorMessage" class="overview-section overview-section--error">
              <div class="overview-section-title">
                <i class="el-icon-warning" />
                {{ $t('eval.detail.overview.sectionError') }}
              </div>
              <div class="overview-items">
                <div class="overview-item overview-item--full">
                  <span class="overview-label">{{ task.errorCode || $t('eval.detail.errorTitle') }}</span>
                  <span class="overview-value overview-value--error">{{ task.errorMessage }}</span>
                </div>
              </div>
            </div>
          </div>
          <EmptyState v-else type="loading" :title="$t('eval.detail.loading')" />
        </el-tab-pane>

        <el-tab-pane :label="$t('eval.detail.tabs.metrics')" name="metrics">
          <MetricsTable :metrics="metrics" />
        </el-tab-pane>

        <el-tab-pane :label="$t('eval.detail.tabs.analysis')" name="analysis" lazy>
          <AnalysisView
            :eval-task-id="evalTaskId"
            :task-status="task && task.status || ''"
          />
        </el-tab-pane>

        <el-tab-pane :label="$t('eval.detail.tabs.artifacts')" name="artifacts">
          <ArtifactList
            :eval-task-id="evalTaskId"
            :report-path="result?.reportPath || ''"
            :raw-result-path="result?.rawResultPath || ''"
            :log-path="result?.logPath || ''"
            :artifacts="result?.artifactsJson || []"
          />
        </el-tab-pane>

        <el-tab-pane :label="$t('eval.detail.tabs.log')" name="log">
          <LogViewer :eval-task-id="evalTaskId" :task-status="task && task.status || ''" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Cancel confirm dialog -->
    <el-dialog
      :title="$t('eval.detail.cancelConfirmTitle')"
      :visible.sync="cancelConfirmVisible"
      width="460px"
      :close-on-click-modal="false"
      append-to-body
    >
      <p>{{ $t('eval.detail.cancelConfirmContent') }}</p>
      <div slot="footer">
        <el-button @click="cancelConfirmVisible = false">{{ $t('eval.detail.cancelConfirmCancel') }}</el-button>
        <el-button type="danger" :loading="cancelling" @click="confirmCancel">{{ $t('eval.detail.cancelConfirmOk') }}</el-button>
      </div>
    </el-dialog>

    <!-- Rerun-eval confirm dialog -->
    <el-dialog
      :title="$t('eval.detail.rerunEvalConfirmTitle')"
      :visible.sync="rerunEvalConfirmVisible"
      width="500px"
      :close-on-click-modal="false"
      append-to-body
    >
      <p class="rerun-eval-content">{{ $t('eval.detail.rerunEvalConfirmContent') }}</p>
      <p class="rerun-eval-hint">{{ $t('eval.detail.rerunEvalConfirmHint') }}</p>
      <div slot="footer">
        <el-button @click="rerunEvalConfirmVisible = false">{{ $t('eval.detail.cancelConfirmCancel') }}</el-button>
        <el-button type="warning" :loading="rerunningEval" @click="confirmRerunEval">{{ $t('eval.detail.rerunEvalConfirmOk') }}</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import StatusTag from "@/components/StatusTag.vue";
import MetricsTable from "@/components/MetricsTable.vue";
import ArtifactList from "@/components/ArtifactList.vue";
import LogViewer from "@/components/LogViewer.vue";
import AnalysisView from "@/components/AnalysisView.vue";
import EmptyState from "@/components/EmptyState.vue";
import EvalSteps from "@/components/EvalSteps.vue";

import { getEvalTask, getEvalResult, cancelEvalTask, rerunEvalNode } from "@/api/eval-task";
import { getDatasetTypeText } from "@/constants/dataset";
import { formatDateTime } from "@/utils/time";
import { canCancelEvalStatus } from "@/constants/eval-status";
import {
  isEvalStatusFinal,
  getEvalStatusText
} from "@/constants/eval-status";

export default {
  name: "EvalTaskDetailView",
  components: {
    PageHeader,
    StatusTag,
    MetricsTable,
    ArtifactList,
    LogViewer,
    AnalysisView,
    EmptyState,
    EvalSteps
  },
  data() {
    return {
      activeTab: "overview",
      loading: false,
      cancelling: false,
      cancelConfirmVisible: false,
      rerunningEval: false,
      rerunEvalConfirmVisible: false,
      task: null,
      result: null,
      pollTimer: null
    };
  },
  computed: {
    canCancel() {
      return this.task && canCancelEvalStatus(this.task.status);
    },
    // 仅在终态、且任务有 outputDir 时允许仅重跑评测节点；
    // 后端会做更严格的产物存在性校验。
    canRerunEval() {
      return (
        this.task &&
        isEvalStatusFinal(this.task.status) &&
        !!this.task.outputDir
      );
    },
    evalTaskId() {
      return this.$route.params.evalTaskId;
    },
    datasetText() {
      return getDatasetTypeText(this.task?.datasetType);
    },
    metrics() {
      return this.result?.metricsJson || this.result?.metrics || [];
    },
    progressLabel() {
      // kept for backward compat – no longer shown in header
      return "";
    },
    pageDescription() {
      if (!this.task) return "";
      if (this.task.taskName) {
        return `${this.task.taskName} · ${this.task.evalTaskId}`;
      }
      return `${this.$t("eval.detail.taskIdLabel")}: ${this.task.evalTaskId}`;
    }
  },
  watch: {
    evalTaskId: {
      immediate: true,
      handler() {
        this.loadAll();
      }
    }
  },
  beforeDestroy() {
    this.stopPolling();
  },
  methods: {
    async loadAll() {
      if (!this.evalTaskId) return;
      this.loading = true;
      try {
        const taskData = await getEvalTask(this.evalTaskId);
        this.task = { evalTaskId: this.evalTaskId, ...taskData };
        if (this.task.status === "succeeded") {
          await this.loadResult();
        }
        this.refreshPolling();
      } finally {
        this.loading = false;
      }
    },
    async loadResult() {
      try {
        this.result = await getEvalResult(this.evalTaskId);
      } catch (e) {
        this.result = null;
      }
    },
    refreshPolling() {
      this.stopPolling();
      if (this.task && !isEvalStatusFinal(this.task.status)) {
        this.pollTimer = setInterval(this.loadAll, 5000);
      }
    },
    stopPolling() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer);
        this.pollTimer = null;
      }
    },
    rerun() {
      if (!this.task) return;
      this.$router.push({
        name: "eval-submit",
        query: {
          provider: this.task.modelProvider,
          modelName: this.task.modelName,
          baseUrl: this.task.modelBaseUrl,
          datasetType: this.task.datasetType,
          datasetName: this.task.datasetName
        }
      });
    },
    formatTime(s) {
      return formatDateTime(s, "—");
    },
    handleCancel() {
      if (!this.task) return;
      this.cancelConfirmVisible = true;
    },
    async confirmCancel() {
      this.cancelling = true;
      try {
        await cancelEvalTask(this.evalTaskId);
        this.$message.success(this.$t("eval.detail.cancelRequested"));
        this.cancelConfirmVisible = false;
        await this.loadAll();
      } finally {
        this.cancelling = false;
      }
    },
    handleRerunEval() {
      if (!this.task) return;
      this.rerunEvalConfirmVisible = true;
    },
    async confirmRerunEval() {
      this.rerunningEval = true;
      try {
        await rerunEvalNode(this.evalTaskId);
        this.$message.success(this.$t("eval.detail.rerunEvalRequested"));
        this.rerunEvalConfirmVisible = false;
        await this.loadAll();
      } finally {
        this.rerunningEval = false;
      }
    }
  }
};
</script>

<style scoped>
.task-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary-card {
  border-radius: 8px;
}

.summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
}

.summary-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.task-title {
  font-weight: 600;
  color: #303133;
  font-size: 15px;
  margin-right: 8px;
}

.task-id {
  font-family: "Menlo", "Monaco", monospace;
  color: #606266;
  font-size: 13px;
}

.summary-meta {
  display: flex;
  gap: 20px;
  color: #606266;
  font-size: 13px;
}

.summary-meta i {
  margin-right: 4px;
  color: #909399;
}

.error-alert {
  margin-top: 16px;
}

.summary-progress {
  margin-top: 18px;
  padding: 16px 20px 8px;
  background: #fafbfc;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.overview-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.overview-section {
  padding: 16px 20px;
  border-radius: 8px;
  background: #fafbfc;
  border-left: 3px solid #c0c4cc;
}

.overview-section--task   { border-left-color: #409eff; }
.overview-section--model  { border-left-color: #9b59b6; }
.overview-section--dataset { border-left-color: #67c23a; }
.overview-section--timeline { border-left-color: #e6a23c; }
.overview-section--error  { border-left-color: #f56c6c; background: #fef0f0; }

.overview-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.overview-section-title i {
  font-size: 15px;
  color: #606266;
}

.overview-section--task   .overview-section-title i { color: #409eff; }
.overview-section--model  .overview-section-title i { color: #9b59b6; }
.overview-section--dataset .overview-section-title i { color: #67c23a; }
.overview-section--timeline .overview-section-title i { color: #e6a23c; }
.overview-section--error  .overview-section-title i { color: #f56c6c; }

.overview-items {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
}

.overview-item {
  flex: 1 1 calc(50% - 16px);
  min-width: 180px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.overview-item--full {
  flex: 1 1 100%;
}

.overview-label {
  font-size: 12px;
  color: #909399;
  line-height: 1;
}

.overview-value {
  font-size: 14px;
  color: #303133;
  line-height: 1.5;
  word-break: break-all;
}

.overview-value--mono {
  font-family: "Menlo", "Monaco", "Consolas", monospace;
  font-size: 13px;
}

.overview-value--error {
  color: #f56c6c;
  white-space: pre-wrap;
}

.overview-sub {
  color: #909399;
  font-size: 13px;
}

.rerun-eval-content {
  white-space: pre-line;
  line-height: 1.6;
}

.rerun-eval-hint {
  margin-top: 8px;
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}
</style>
