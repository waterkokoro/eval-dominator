<template>
  <div>
    <div v-if="rows.length" class="metrics-summary">
      <div class="summary-cell">
        <span class="summary-label">指标条数</span>
        <span class="summary-value">{{ rows.length }}</span>
      </div>
      <div class="summary-cell">
        <span class="summary-label">有效条数</span>
        <span class="summary-value">{{ summary.validCount }}</span>
      </div>
      <div class="summary-cell">
        <span class="summary-label">平均得分</span>
        <span class="summary-value">{{ formatValue(summary.average) }}</span>
      </div>
      <div class="summary-cell">
        <span class="summary-label">最高/最低</span>
        <span class="summary-value">
          {{ formatValue(summary.max) }} / {{ formatValue(summary.min) }}
        </span>
      </div>
      <div class="summary-cell summary-actions">
        <el-input
          v-model="keyword"
          placeholder="按数据集/指标名搜索"
          size="small"
          clearable
          style="width: 200px"
        />
        <el-select
          v-model="modeFilter"
          size="small"
          placeholder="推理方式"
          clearable
          style="width: 120px; margin-left: 8px"
        >
          <el-option v-for="m in modeOptions" :key="m" :value="m" :label="m" />
        </el-select>
      </div>
    </div>

    <el-table
      v-if="filteredRows.length"
      :data="filteredRows"
      border
      stripe
      size="small"
      :default-sort="{ prop: 'numericValue', order: 'descending' }"
    >
      <el-table-column label="指标" min-width="180">
        <template #default="{ row }">
          <span class="metric-name">{{ row.displayName || row.name }}</span>
          <div
            v-if="row.name && row.displayName && row.name !== row.displayName"
            class="metric-key"
          >
            {{ row.name }}
          </div>
        </template>
      </el-table-column>
      <el-table-column label="数据集" width="220">
        <template #default="{ row }">
          <span class="cell-mono">{{ row.dataset || "—" }}</span>
        </template>
      </el-table-column>
      <el-table-column label="推理方式" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.mode" size="mini" :type="modeTagType(row.mode)">
            {{ row.mode.toUpperCase() }}
          </el-tag>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column label="模型" min-width="160">
        <template #default="{ row }">
          <span class="cell-mono">{{ row.model || "—" }}</span>
        </template>
      </el-table-column>
      <el-table-column label="得分" width="220" prop="numericValue" sortable>
        <template #default="{ row }">
          <div v-if="row.isPercent" class="score-bar">
            <el-progress
              :percentage="Math.max(0, Math.min(100, row.percent))"
              :color="progressColor(row.percent)"
              :stroke-width="10"
              :show-text="false"
            />
            <span class="score-text">{{ formatPercent(row.percent) }}</span>
          </div>
          <span v-else class="metric-value">{{ formatValue(row.value) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="说明" min-width="200">
        <template #default="{ row }">
          <span>{{ row.description || "—" }}</span>
        </template>
      </el-table-column>
    </el-table>
    <EmptyState
      v-else
      title="暂无指标数据"
      :description="rows.length ? '没有符合条件的指标，请调整筛选' : '任务执行成功后会展示评测指标'"
    />
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";

const PERCENT_HINTS = [
  /(^|[._-])accuracy$/i,
  /(^|[._-])acc$/i,
  /(^|[._-])score$/i,
  /(^|[._-])ratio$/i,
  /(^|[._-])precision$/i,
  /(^|[._-])recall$/i,
  /(^|[._-])f1$/i
];

function looksLikePercent(name, value) {
  if (typeof value !== "number") return false;
  if (value > 1.5 && value <= 100) return true;
  if (value >= 0 && value <= 1) {
    return PERCENT_HINTS.some((re) => re.test(name || ""));
  }
  return false;
}

export default {
  name: "MetricsTable",
  components: { EmptyState },
  props: {
    metrics: {
      type: [Array, String, Object],
      default: () => []
    }
  },
  data() {
    return {
      keyword: "",
      modeFilter: ""
    };
  },
  computed: {
    rows() {
      const raw = this.metrics;
      if (!raw) return [];
      let parsed = raw;
      if (typeof raw === "string") {
        try {
          parsed = JSON.parse(raw);
        } catch (e) {
          return [];
        }
      }
      let list = [];
      if (Array.isArray(parsed)) list = parsed;
      else if (parsed && Array.isArray(parsed.metrics)) list = parsed.metrics;
      return list.map((m) => this.decorate(m));
    },
    modeOptions() {
      const set = new Set();
      this.rows.forEach((r) => r.mode && set.add(r.mode));
      return Array.from(set);
    },
    filteredRows() {
      const kw = this.keyword.trim().toLowerCase();
      return this.rows.filter((r) => {
        if (this.modeFilter && r.mode !== this.modeFilter) return false;
        if (!kw) return true;
        return [r.displayName, r.name, r.dataset, r.model]
          .filter(Boolean)
          .some((v) => String(v).toLowerCase().includes(kw));
      });
    },
    summary() {
      const valid = this.rows.filter((r) => r.numericValue != null && !Number.isNaN(r.numericValue));
      if (!valid.length) {
        return { validCount: 0, average: null, max: null, min: null };
      }
      const values = valid.map((r) => r.numericValue);
      const sum = values.reduce((a, b) => a + b, 0);
      return {
        validCount: valid.length,
        average: sum / valid.length,
        max: Math.max(...values),
        min: Math.min(...values)
      };
    }
  },
  methods: {
    decorate(m) {
      const extra = m.extra || {};
      const value = m.value;
      const isPercent = looksLikePercent(m.name, value);
      const numericValue = typeof value === "number" ? value : Number(value);
      const percent = isPercent
        ? typeof value === "number" && value <= 1
          ? value * 100
          : value
        : null;
      return {
        ...m,
        dataset: extra.dataset || extra.subset || "",
        mode: extra.mode && extra.mode !== "-" ? extra.mode : "",
        model: extra.model || "",
        isPercent,
        percent,
        numericValue: Number.isNaN(numericValue) ? null : numericValue
      };
    },
    formatValue(value) {
      if (value === null || value === undefined || value === "") return "—";
      if (typeof value === "number") {
        if (Number.isInteger(value)) return String(value);
        return value.toFixed(4);
      }
      return String(value);
    },
    formatPercent(p) {
      if (p === null || p === undefined || Number.isNaN(p)) return "—";
      return `${p.toFixed(2)}%`;
    },
    modeTagType(mode) {
      if (mode === "ppl") return "warning";
      if (mode === "gen") return "success";
      return "info";
    },
    progressColor(p) {
      if (p == null || Number.isNaN(p)) return "#909399";
      if (p >= 80) return "#67c23a";
      if (p >= 60) return "#e6a23c";
      return "#f56c6c";
    }
  }
};
</script>

<style scoped>
.metrics-summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  padding: 12px 14px;
  margin-bottom: 12px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background: #fafbfc;
}
.summary-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.summary-label {
  font-size: 12px;
  color: #909399;
}
.summary-value {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  font-family: "Menlo", "Monaco", monospace;
}
.summary-actions {
  margin-left: auto;
  flex-direction: row;
  align-items: center;
  gap: 0;
}

.metric-name {
  font-weight: 500;
  color: #303133;
}
.metric-key {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.metric-value {
  font-family: "Menlo", "Monaco", monospace;
  color: #409eff;
}
.cell-mono {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #606266;
}

.score-bar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.score-bar >>> .el-progress {
  flex: 1;
}
.score-text {
  width: 80px;
  text-align: right;
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #303133;
}
</style>
