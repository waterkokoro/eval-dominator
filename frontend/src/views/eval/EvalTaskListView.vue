<template>
  <div class="task-list">
    <PageHeader
      title="任务列表"
      description="查看历史评测任务的状态、模型与数据集"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          刷新
        </el-button>
        <el-button
          type="primary"
          icon="el-icon-plus"
          @click="$router.push({ name: 'eval-submit' })"
        >
          新建评测
        </el-button>
      </template>
    </PageHeader>

    <el-alert
      v-if="apiNotReady"
      type="info"
      :closable="false"
      title="任务列表接口待后端补齐"
      description="后端实现 GET /eval/tasks 后此处将自动渲染历史任务。当前可通过新建评测后跳转到任务详情手动追踪。"
      class="page-alert"
      show-icon
    />

    <el-card shadow="never" class="filter-card">
      <el-form :inline="true" :model="filter" size="small" @submit.native.prevent>
        <el-form-item label="任务">
          <el-input
            v-model="filter.search"
            placeholder="任务名称或任务 ID（模糊）"
            clearable
            style="width: 240px"
          />
        </el-form-item>
        <el-form-item label="创建时间">
          <el-date-picker
            v-model="filter.dateRange"
            type="daterange"
            align="right"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="yyyy-MM-dd"
            unlink-panels
            clearable
            style="width: 280px"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="filter.status"
            placeholder="全部"
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
        <el-form-item label="模型">
          <el-input
            v-model="filter.keyword"
            placeholder="模型名 / Provider"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item label="数据集">
          <el-select
            v-model="filter.datasetType"
            placeholder="全部"
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
            查询
          </el-button>
          <el-button @click="resetFilter">重置</el-button>
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
        <el-table-column label="任务名称" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ displayTaskName(row) }}
          </template>
        </el-table-column>
        <el-table-column label="任务 ID" min-width="140">
          <template #default="{ row }">
            <span class="task-id">{{ shortId(row.evalTaskId) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="模型" min-width="200">
          <template #default="{ row }">
            <div class="model-cell">
              <span>{{ row.modelName || "-" }}</span>
              <span class="model-meta">{{ row.modelProvider }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="数据集" min-width="180">
          <template #default="{ row }">
            <div class="dataset-cell">
              <span>{{ row.datasetName || "-" }}</span>
              <span class="model-meta">{{ datasetText(row.datasetType) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <StatusTag :status="row.status" />
          </template>
        </el-table-column>
        <el-table-column label="进度" width="160">
          <template #default="{ row }">
            <EvalProgress :status="row.status" :stroke-width="6" />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">
            {{ duration(row) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="right">
          <template #default="{ row }">
            <el-button
              type="text"
              size="mini"
              @click.stop="goDetail(row.evalTaskId)"
            >
              查看
            </el-button>
            <el-button
              v-if="canCancelEvalStatus(row.status)"
              type="text"
              size="mini"
              class="row-cancel"
              :loading="cancellingId === row.evalTaskId"
              @click.stop="handleCancel(row)"
            >
              终止
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
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import StatusTag from "@/components/StatusTag.vue";
import EvalProgress from "@/components/EvalProgress.vue";

import { listEvalTasks, cancelEvalTask } from "@/api/eval-task";
import {
  evalStatusOptions,
  isEvalStatusFinal,
  canCancelEvalStatus
} from "@/constants/eval-status";
import { datasetTypeOptions, getDatasetTypeText } from "@/constants/dataset";
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
      pagination: {
        page: 1,
        pageSize: 10
      },
      filter: buildFilter(),
      statusOptions: evalStatusOptions,
      datasetTypeOptions
    };
  },
  computed: {
    emptyText() {
      if (this.apiNotReady) return "接口待后端补齐";
      return "暂无评测任务，点击右上角「新建评测」开始";
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
      try {
        await this.$confirm(
          `确定要终止任务「${row.taskName || row.evalTaskId}」吗？`,
          "终止评测",
          { type: "warning", confirmButtonText: "确定终止", cancelButtonText: "取消" }
        );
      } catch (e) {
        return;
      }
      this.cancellingId = row.evalTaskId;
      try {
        await cancelEvalTask(row.evalTaskId);
        this.$message.success("已请求终止");
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
