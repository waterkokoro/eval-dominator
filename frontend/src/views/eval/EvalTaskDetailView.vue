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
        <div class="summary-progress-meta">
          <span class="summary-progress-label">{{ progressLabel }}</span>
        </div>
        <EvalProgress :status="task.status" :stroke-width="10" />
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
          <el-descriptions
            v-if="task"
            :column="2"
            border
            size="small"
            label-class-name="desc-label"
          >
            <el-descriptions-item :label="$t('eval.detail.overview.taskName')">
              {{ task.taskName || "—" }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.taskId')">
              {{ task.evalTaskId }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.status')">
              <StatusTag :status="task.status" />
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.modelProvider')">
              {{ task.modelProvider || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.modelName')">
              {{ task.modelName || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.baseUrl')">
              {{ task.modelBaseUrl || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.dataset')">
              {{ task.datasetName || "-" }} ({{ datasetText }})
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.createdAt')">
              {{ formatTime(task.createdAt) }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.startedAt')">
              {{ formatTime(task.startedAt) }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.finishedAt')">
              {{ formatTime(task.finishedAt) }}
            </el-descriptions-item>
            <el-descriptions-item :label="$t('eval.detail.overview.errorMessage')">
              {{ task.errorMessage || "-" }}
            </el-descriptions-item>
          </el-descriptions>
          <EmptyState v-else type="loading" :title="$t('eval.detail.loading')" />
        </el-tab-pane>

        <el-tab-pane :label="$t('eval.detail.tabs.metrics')" name="metrics">
          <MetricsTable :metrics="metrics" />
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
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import StatusTag from "@/components/StatusTag.vue";
import MetricsTable from "@/components/MetricsTable.vue";
import ArtifactList from "@/components/ArtifactList.vue";
import LogViewer from "@/components/LogViewer.vue";
import EmptyState from "@/components/EmptyState.vue";
import EvalProgress from "@/components/EvalProgress.vue";

import { getEvalTask, getEvalResult, cancelEvalTask } from "@/api/eval-task";
import { getDatasetTypeText } from "@/constants/dataset";
import { formatDateTime } from "@/utils/time";
import { canCancelEvalStatus } from "@/constants/eval-status";
import {
  isEvalStatusFinal,
  getEvalStatusText,
  getEvalStatusProgress
} from "@/constants/eval-status";

export default {
  name: "EvalTaskDetailView",
  components: {
    PageHeader,
    StatusTag,
    MetricsTable,
    ArtifactList,
    LogViewer,
    EmptyState,
    EvalProgress
  },
  data() {
    return {
      activeTab: "overview",
      loading: false,
      cancelling: false,
      task: null,
      result: null,
      pollTimer: null
    };
  },
  computed: {
    canCancel() {
      return this.task && canCancelEvalStatus(this.task.status);
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
      if (!this.task) return "";
      this.$i18n.locale; // eslint-disable-line no-unused-expressions
      const text = getEvalStatusText(this.task.status);
      const pct = getEvalStatusProgress(this.task.status);
      return `${text} · ${pct}%`;
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
    async handleCancel() {
      if (!this.task) return;
      try {
        await this.$confirm(
          this.$t("eval.detail.cancelConfirmContent"),
          this.$t("eval.detail.cancelConfirmTitle"),
          {
            type: "warning",
            confirmButtonText: this.$t("eval.detail.cancelConfirmOk"),
            cancelButtonText: this.$t("eval.detail.cancelConfirmCancel")
          }
        );
      } catch (e) {
        return;
      }
      this.cancelling = true;
      try {
        await cancelEvalTask(this.evalTaskId);
        this.$message.success(this.$t("eval.detail.cancelRequested"));
        await this.loadAll();
      } finally {
        this.cancelling = false;
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
  margin-top: 16px;
}

.summary-progress-meta {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}

.summary-progress-label {
  font-size: 13px;
  color: #606266;
}

.task-detail >>> .desc-label {
  width: 140px;
  background: #f5f7fa;
}
</style>
