<template>
  <div class="task-list">
    <PageHeader
      :title="$t('eval.list.title')"
      :description="$t('eval.list.description')"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          {{ $t("common.actions.refresh") }}
        </el-button>
        <el-button
          type="primary"
          icon="el-icon-plus"
          @click="$router.push({ name: 'eval-submit' })"
        >
          {{ $t("eval.list.createTask") }}
        </el-button>
      </template>
    </PageHeader>

    <el-alert
      v-if="apiNotReady"
      type="info"
      :closable="false"
      :title="$t('eval.list.apiNotReadyTitle')"
      :description="$t('eval.list.apiNotReadyDescription')"
      class="page-alert"
      show-icon
    />

    <el-card shadow="never" class="filter-card">
      <el-form :inline="true" :model="filter" size="small" @submit.native.prevent>
        <el-form-item :label="$t('eval.list.filter.task')">
          <el-input
            v-model="filter.search"
            :placeholder="$t('eval.list.filter.taskPlaceholder')"
            clearable
            style="width: 240px"
          />
        </el-form-item>
        <el-form-item :label="$t('eval.list.filter.createdAt')">
          <el-date-picker
            v-model="filter.dateRange"
            type="daterange"
            align="right"
            :range-separator="$t('eval.list.filter.rangeSeparator')"
            :start-placeholder="$t('eval.list.filter.startDate')"
            :end-placeholder="$t('eval.list.filter.endDate')"
            value-format="yyyy-MM-dd"
            unlink-panels
            clearable
            style="width: 280px"
          />
        </el-form-item>
        <el-form-item :label="$t('eval.list.filter.status')">
          <el-select
            v-model="filter.status"
            :placeholder="$t('eval.list.filter.statusPlaceholder')"
            multiple
            collapse-tags
            clearable
            style="width: 220px"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :value="item.value"
              :label="item.label"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('eval.list.filter.model')">
          <el-input
            v-model="filter.keyword"
            :placeholder="$t('eval.list.filter.modelPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item :label="$t('eval.list.filter.dataset')">
          <el-select
            v-model="filter.datasetType"
            :placeholder="$t('eval.list.filter.datasetPlaceholder')"
            clearable
            style="width: 200px"
          >
            <el-option
              v-for="item in datasetTypeOptions"
              :key="item.value"
              :value="item.value"
              :label="item.label"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="loadList">
            {{ $t("common.actions.search") }}
          </el-button>
          <el-button @click="resetFilter">{{ $t("common.actions.reset") }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="table-card">
      <el-table
        v-loading="loading"
        :data="rows"
        :empty-text="emptyText"
        stripe
        size="small"
        @row-click="handleRowClick"
      >
        <el-table-column :label="$t('eval.list.columns.taskName')" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ displayTaskName(row) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.taskId')" min-width="140">
          <template #default="{ row }">
            <span class="task-id">{{ shortId(row.evalTaskId) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.model')" min-width="200">
          <template #default="{ row }">
            <div class="model-cell">
              <span>{{ row.modelName || "-" }}</span>
              <span class="model-meta">{{ row.modelProvider }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.dataset')" min-width="180">
          <template #default="{ row }">
            <div class="dataset-cell">
              <span>{{ row.datasetName || "-" }}</span>
              <span class="model-meta">{{ datasetText(row.datasetType) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.status')" width="110">
          <template #default="{ row }">
            <StatusTag :status="row.status" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.progress')" width="160">
          <template #default="{ row }">
            <EvalProgress :status="row.status" :stroke-width="6" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.createdAt')" width="170">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('eval.list.columns.duration')" width="90">
          <template #default="{ row }">
            {{ duration(row) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.fields.actions')" width="160" align="right">
          <template #default="{ row }">
            <el-button
              type="text"
              size="mini"
              @click.stop="goDetail(row.evalTaskId)"
            >
              {{ $t("eval.list.actions.view") }}
            </el-button>
            <el-button
              v-if="canCancelEvalStatus(row.status)"
              type="text"
              size="mini"
              class="row-cancel"
              :loading="cancellingId === row.evalTaskId"
              @click.stop="handleCancel(row)"
            >
              {{ $t("eval.list.actions.cancel") }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="total > 0" class="pagination">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next"
          :total="total"
          :page-sizes="[10, 20, 50]"
          :page-size.sync="pagination.pageSize"
          :current-page.sync="pagination.page"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- Cancel confirm dialog -->
    <el-dialog
      :title="$t('eval.list.cancelConfirmTitle')"
      :visible.sync="cancelConfirm.visible"
      width="460px"
      :close-on-click-modal="false"
      append-to-body
    >
      <p v-if="cancelConfirm.target">{{ $t('eval.list.cancelConfirmContent', { name: cancelConfirm.target.taskName || cancelConfirm.target.evalTaskId }) }}</p>
      <div slot="footer">
        <el-button @click="cancelConfirm.visible = false">{{ $t('eval.list.cancelConfirmCancel') }}</el-button>
        <el-button type="danger" :loading="cancellingId !== ''" @click="confirmCancel">{{ $t('eval.list.cancelConfirmOk') }}</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import StatusTag from "@/components/StatusTag.vue";
import EvalProgress from "@/components/EvalProgress.vue";

import { listEvalTasks, cancelEvalTask } from "@/api/eval-task";
import {
  getEvalStatusOptions,
  isEvalStatusFinal,
  canCancelEvalStatus
} from "@/constants/eval-status";
import { getDatasetTypeOptions, getDatasetTypeText } from "@/constants/dataset";
import { formatDateTime, durationText } from "@/utils/time";

const buildFilter = () => ({
  search: "",
  dateRange: null,
  status: [],
  keyword: "",
  datasetType: ""
});

export default {
  name: "EvalTaskListView",
  components: { PageHeader, StatusTag, EvalProgress },
  data() {
    return {
      loading: false,
      apiNotReady: false,
      rows: [],
      total: 0,
      pollTimer: null,
      cancellingId: "",
      cancelConfirm: {
        visible: false,
        target: null
      },
      pagination: {
        page: 1,
        pageSize: 10
      },
      filter: buildFilter()
    };
  },
  computed: {
    statusOptions() {
      // 依赖 $i18n.locale 让选项随语言切换刷新
      this.$i18n.locale; // eslint-disable-line no-unused-expressions
      return getEvalStatusOptions();
    },
    datasetTypeOptions() {
      this.$i18n.locale; // eslint-disable-line no-unused-expressions
      return getDatasetTypeOptions();
    },
    emptyText() {
      if (this.apiNotReady) return this.$t("eval.list.emptyApiNotReady");
      return this.$t("eval.list.empty");
    }
  },
  created() {
    this.loadList();
  },
  beforeDestroy() {
    if (this.pollTimer) clearInterval(this.pollTimer);
  },
  methods: {
    resetFilter() {
      this.filter = buildFilter();
      this.pagination.page = 1;
      this.loadList();
    },
    async loadList(silent = false) {
      if (!silent) this.loading = true;
      try {
        const params = {
          page: this.pagination.page,
          pageSize: this.pagination.pageSize,
          search: this.filter.search || undefined,
          status: this.filter.status?.length ? this.filter.status.join(",") : undefined,
          keyword: this.filter.keyword || undefined,
          datasetType: this.filter.datasetType || undefined
        };
        const dr = this.filter.dateRange;
        if (dr && dr.length === 2) {
          params.createdFrom = dr[0];
          params.createdTo = dr[1];
        }
        const data = await listEvalTasks(params);
        const items = Array.isArray(data) ? data : data?.items || [];
        this.rows = items;
        this.total = data?.total ?? items.length;
        this.apiNotReady = false;
        this.refreshPolling();
      } catch (error) {
        const status = error?.response?.status;
        this.rows = [];
        this.total = 0;
        this.apiNotReady = !status || status === 404;
      } finally {
        if (!silent) this.loading = false;
      }
    },
    refreshPolling() {
      const hasRunning = this.rows.some((row) => !isEvalStatusFinal(row.status));
      if (hasRunning && !this.pollTimer) {
        this.pollTimer = setInterval(() => this.loadList(true), 5000);
      } else if (!hasRunning && this.pollTimer) {
        clearInterval(this.pollTimer);
        this.pollTimer = null;
      }
    },
    shortId(id) {
      if (!id) return "-";
      if (id.length <= 22) return id;
      return `${id.slice(0, 12)}…${id.slice(-6)}`;
    },
    datasetText(type) {
      return getDatasetTypeText(type);
    },
    duration(row) {
      return durationText(row.startedAt, row.finishedAt);
    },
    formatTime(s) {
      return formatDateTime(s, "—");
    },
    handleRowClick(row) {
      this.goDetail(row.evalTaskId);
    },
    goDetail(evalTaskId) {
      if (!evalTaskId) return;
      this.$router.push({
        name: "eval-task-detail",
        params: { evalTaskId }
      });
    },
    canCancelEvalStatus,
    async handleCancel(row) {
      if (!row?.evalTaskId) return;
      this.cancelConfirm.target = row;
      this.cancelConfirm.visible = true;
    },
    async confirmCancel() {
      const row = this.cancelConfirm.target;
      if (!row?.evalTaskId) return;
      this.cancellingId = row.evalTaskId;
      try {
        await cancelEvalTask(row.evalTaskId);
        this.$message.success(this.$t("eval.list.cancelRequested"));
        this.cancelConfirm.visible = false;
        await this.loadList(true);
      } finally {
        this.cancellingId = "";
      }
    },
    /** 后端 camelCase；若经代理/旧缓存出现 snake_case 也兼容 */
    displayTaskName(row) {
      const n = row && (row.taskName != null && row.taskName !== ""
        ? row.taskName
        : row.task_name);
      return (n && String(n).trim()) || "—";
    }
  }
};
</script>

<style scoped>
.task-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-alert {
  margin-bottom: 0;
}

.filter-card,
.table-card {
  border-radius: 8px;
}

.task-id {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #303133;
}

.model-cell,
.dataset-cell {
  display: flex;
  flex-direction: column;
}

.model-meta {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
