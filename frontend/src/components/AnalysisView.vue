<template>
  <div class="analysis-view" v-loading="loading">
    <div v-if="error" class="analysis-error">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        :title="$t('eval.analysis.unavailableTitle')"
        :description="error"
      />
    </div>

    <template v-else-if="data">
      <!-- 顶部分类汇总 -->
      <div class="analysis-summary">
        <div
          v-for="cat in categoryList"
          :key="cat.key"
          class="summary-cell"
          :class="['cell-' + cat.key, { active: activeCategory === cat.key }]"
          @click="toggleCategory(cat.key)"
        >
          <div class="summary-num">{{ summaryCount(cat.key) }}</div>
          <div class="summary-label">
            <span class="dot" :class="'dot-' + cat.key" />
            {{ cat.label }}
          </div>
          <div class="summary-range">{{ cat.range }}</div>
        </div>
        <div class="summary-cell summary-total">
          <div class="summary-num">{{ data.summary.total }}</div>
          <div class="summary-label">{{ $t("eval.analysis.summary.total") }}</div>
        </div>
      </div>

      <!-- 工具栏 -->
      <div class="analysis-toolbar">
        <el-input
          v-model="keyword"
          size="small"
          clearable
          :placeholder="$t('eval.analysis.searchPlaceholder')"
          prefix-icon="el-icon-search"
          class="toolbar-search"
        />
        <el-checkbox v-model="hideHighScore" size="small">
          {{ $t("eval.analysis.hidePass") }}
        </el-checkbox>
        <span class="toolbar-tip">
          {{
            $t("eval.analysis.showingCount", {
              count: visibleCount,
              total: data.items.length
            })
          }}
        </span>
      </div>

      <!-- 分类分组列表 -->
      <div class="analysis-groups">
        <div
          v-for="cat in categoryList"
          :key="cat.key"
          v-show="groupedItems[cat.key].length > 0"
          class="analysis-group"
          :class="'group-' + cat.key"
        >
          <div
            class="group-header"
            :class="{ collapsed: !groupExpanded[cat.key] }"
            @click="toggleGroup(cat.key)"
          >
            <i
              :class="
                groupExpanded[cat.key]
                  ? 'el-icon-arrow-down'
                  : 'el-icon-arrow-right'
              "
            />
            <span class="dot" :class="'dot-' + cat.key" />
            <span class="group-title">{{ cat.label }}</span>
            <span class="group-count">{{ groupedItems[cat.key].length }}</span>
            <span class="group-range">{{ cat.range }}</span>
          </div>
          <div v-show="groupExpanded[cat.key]" class="group-body">
            <div
              v-for="item in groupedItems[cat.key]"
              :key="item.index"
              class="analysis-item"
            >
              <div class="item-head" @click="toggleItem(item.index)">
                <span class="item-index">#{{ item.index }}</span>
                <span class="item-prompt-preview" :title="item.prompt">
                  {{ promptPreview(item.prompt) }}
                </span>
                <span class="item-spacer" />
                <span class="item-score" :class="'score-' + item.category">
                  {{ formatScore(item) }}
                </span>
                <i
                  :class="
                    expandedItems[item.index]
                      ? 'el-icon-arrow-down'
                      : 'el-icon-arrow-right'
                  "
                  class="item-toggle"
                />
              </div>
              <div v-show="expandedItems[item.index]" class="item-body">
                <div class="item-section">
                  <div class="section-label">
                    {{ $t("eval.analysis.fields.prompt") }}
                  </div>
                  <div class="section-content content-prompt">
                    {{ item.prompt || "—" }}
                  </div>
                </div>
                <div class="item-section">
                  <div class="section-label">
                    {{ $t("eval.analysis.fields.prediction") }}
                  </div>
                  <div
                    class="section-content content-prediction"
                    :class="{ 'is-failure': item.category === 'failed' }"
                  >
                    {{ item.prediction || "—" }}
                  </div>
                </div>
                <div class="item-section">
                  <div class="section-label">
                    {{ $t("eval.analysis.fields.reference") }}
                  </div>
                  <div class="section-content">
                    <template
                      v-if="item.referenceTokens && item.referenceTokens.length"
                    >
                      <el-tag
                        v-for="tok in item.referenceTokens"
                        :key="'ref-' + tok"
                        size="mini"
                        :type="
                          (item.hitTokens || []).indexOf(tok) >= 0
                            ? 'success'
                            : 'info'
                        "
                        effect="plain"
                        class="token-tag"
                      >
                        <i
                          :class="
                            (item.hitTokens || []).indexOf(tok) >= 0
                              ? 'el-icon-check'
                              : 'el-icon-close'
                          "
                        />
                        {{ tok }}
                      </el-tag>
                    </template>
                    <span v-else class="content-fallback">
                      {{ item.reference || "—" }}
                    </span>
                  </div>
                </div>
                <div v-if="item.note" class="item-note">
                  <i class="el-icon-warning-outline" />
                  {{ item.note }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <EmptyState
        v-if="visibleCount === 0"
        type="todo"
        :title="$t('eval.analysis.empty.title')"
        :description="$t('eval.analysis.empty.description')"
      />
    </template>

    <EmptyState
      v-else-if="!loading"
      type="todo"
      :title="$t('eval.analysis.empty.title')"
      :description="$t('eval.analysis.empty.waitDescription')"
    />
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";
import { getEvalAnalysis } from "@/api/eval-task";
import { resolveApiErrorMessage } from "@/api/http";

const PROMPT_PREVIEW_LEN = 80;

export default {
  name: "AnalysisView",
  components: { EmptyState },
  props: {
    evalTaskId: { type: String, default: "" },
    taskStatus: { type: String, default: "" }
  },
  data() {
    return {
      loading: false,
      data: null,
      error: "",
      keyword: "",
      hideHighScore: false,
      activeCategory: "",
      groupExpanded: {
        failed: true,
        low: true,
        mid: true,
        pass: false
      },
      expandedItems: {}
    };
  },
  computed: {
    categoryList() {
      return [
        {
          key: "failed",
          label: this.$t("eval.analysis.categories.failed"),
          range: this.$t("eval.analysis.ranges.failed")
        },
        {
          key: "low",
          label: this.$t("eval.analysis.categories.low"),
          range: this.$t("eval.analysis.ranges.low")
        },
        {
          key: "mid",
          label: this.$t("eval.analysis.categories.mid"),
          range: this.$t("eval.analysis.ranges.mid")
        },
        {
          key: "pass",
          label: this.$t("eval.analysis.categories.pass"),
          range: this.$t("eval.analysis.ranges.pass")
        }
      ];
    },
    filteredItems() {
      if (!this.data) return [];
      const kw = (this.keyword || "").trim().toLowerCase();
      return this.data.items.filter((item) => {
        if (this.hideHighScore && item.category === "pass") return false;
        if (
          this.activeCategory &&
          item.category !== this.activeCategory
        ) {
          return false;
        }
        if (!kw) return true;
        return (
          (item.prompt || "").toLowerCase().includes(kw) ||
          (item.prediction || "").toLowerCase().includes(kw) ||
          (item.reference || "").toLowerCase().includes(kw)
        );
      });
    },
    groupedItems() {
      const map = { failed: [], low: [], mid: [], pass: [] };
      this.filteredItems.forEach((it) => {
        const arr = map[it.category];
        if (arr) arr.push(it);
      });
      return map;
    },
    visibleCount() {
      return this.filteredItems.length;
    }
  },
  watch: {
    evalTaskId: {
      immediate: true,
      handler() {
        this.fetchData();
      }
    },
    taskStatus(val, old) {
      // 任务从非终态切到 succeeded 时再拉一次
      if (val === "succeeded" && val !== old) {
        this.fetchData();
      }
    }
  },
  methods: {
    async fetchData() {
      if (!this.evalTaskId) return;
      this.loading = true;
      this.error = "";
      try {
        this.data = await getEvalAnalysis(this.evalTaskId);
      } catch (e) {
        this.data = null;
        this.error =
          resolveApiErrorMessage(e) ||
          this.$t("eval.analysis.unavailableDescription");
      } finally {
        this.loading = false;
      }
    },
    toggleCategory(key) {
      this.activeCategory = this.activeCategory === key ? "" : key;
    },
    toggleGroup(key) {
      this.$set(this.groupExpanded, key, !this.groupExpanded[key]);
    },
    toggleItem(index) {
      this.$set(this.expandedItems, index, !this.expandedItems[index]);
    },
    summaryCount(key) {
      if (!this.data) return 0;
      return this.data.summary[key] || 0;
    },
    promptPreview(text) {
      const t = (text || "").replace(/\s+/g, " ").trim();
      if (t.length <= PROMPT_PREVIEW_LEN) return t;
      return t.slice(0, PROMPT_PREVIEW_LEN) + "…";
    },
    formatScore(item) {
      if (item.category === "failed") {
        return this.$t("eval.analysis.scoreFailed");
      }
      const pct = Math.round((item.score || 0) * 100);
      const hits = (item.hitTokens || []).length;
      const total =
        (item.referenceTokens || []).length || (item.score > 0 ? 1 : 0);
      if (total > 0) {
        return `${pct}% (${hits}/${total})`;
      }
      return `${pct}%`;
    }
  }
};
</script>

<style scoped>
.analysis-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.analysis-error {
  margin-top: 8px;
}

/* 顶部分类卡片 */
.analysis-summary {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
}
.summary-cell {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px 16px;
  background: #fafbfc;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.summary-cell:hover {
  border-color: #c0c4cc;
}
.summary-cell.active {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.12);
}
.summary-total {
  cursor: default;
  background: #f5f7fa;
}
.summary-num {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
}
.summary-label {
  font-size: 13px;
  color: #606266;
  margin-top: 2px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.summary-range {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}
.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.dot-failed {
  background: #f56c6c;
}
.dot-low {
  background: #e6a23c;
}
.dot-mid {
  background: #409eff;
}
.dot-pass {
  background: #67c23a;
}

/* 工具栏 */
.analysis-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 4px 0;
}
.toolbar-search {
  width: 280px;
}
.toolbar-tip {
  margin-left: auto;
  font-size: 12px;
  color: #909399;
}

/* 分组 */
.analysis-groups {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.analysis-group {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}
.group-header {
  padding: 10px 14px;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-weight: 500;
  user-select: none;
}
.group-header:hover {
  background: #ecf5ff;
}
.group-title {
  color: #303133;
  font-size: 14px;
}
.group-count {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 10px;
  padding: 0 8px;
  font-size: 12px;
  color: #606266;
  line-height: 18px;
}
.group-range {
  margin-left: auto;
  color: #909399;
  font-size: 12px;
}
.group-body {
  border-top: 1px solid #ebeef5;
}

/* 单条 */
.analysis-item {
  border-bottom: 1px solid #f0f2f5;
}
.analysis-item:last-child {
  border-bottom: none;
}
.item-head {
  padding: 8px 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  font-size: 13px;
}
.item-head:hover {
  background: #fafbfc;
}
.item-index {
  font-family: "Menlo", "Monaco", monospace;
  color: #909399;
  font-size: 12px;
  min-width: 36px;
}
.item-prompt-preview {
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.item-spacer {
  flex: 0 0 8px;
}
.item-score {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  padding: 1px 8px;
  border-radius: 10px;
  border: 1px solid transparent;
}
.score-failed {
  color: #f56c6c;
  border-color: #fbc4c4;
  background: #fef0f0;
}
.score-low {
  color: #e6a23c;
  border-color: #f5dab1;
  background: #fdf6ec;
}
.score-mid {
  color: #409eff;
  border-color: #b3d8ff;
  background: #ecf5ff;
}
.score-pass {
  color: #67c23a;
  border-color: #c2e7b0;
  background: #f0f9eb;
}
.item-toggle {
  color: #909399;
}

.item-body {
  padding: 8px 14px 14px 50px;
  background: #fafbfc;
}
.item-section {
  margin-top: 8px;
}
.section-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.section-content {
  font-size: 13px;
  color: #303133;
  white-space: pre-wrap;
  word-break: break-word;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 8px 10px;
  line-height: 1.6;
}
.content-prompt {
  background: #fff;
}
.content-prediction {
  background: #fff;
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
}
.content-prediction.is-failure {
  color: #f56c6c;
  background: #fef0f0;
  border-color: #fbc4c4;
}
.content-fallback {
  color: #606266;
}
.token-tag {
  margin-right: 6px;
  margin-bottom: 4px;
}
.token-tag i {
  margin-right: 4px;
}
.item-note {
  margin-top: 6px;
  font-size: 12px;
  color: #e6a23c;
}
.item-note i {
  margin-right: 4px;
}

@media (max-width: 900px) {
  .analysis-summary {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
