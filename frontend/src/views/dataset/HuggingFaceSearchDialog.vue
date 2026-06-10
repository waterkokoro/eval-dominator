<template>
  <el-dialog
    :title="$t('dataset.huggingface.dialogTitle')"
    :visible.sync="dialogVisible"
    width="860px"
    append-to-body
    top="4vh"
    @close="handleClose"
  >
    <div class="hf-search">
      <!-- Search bar -->
      <div class="hf-search-bar">
        <el-input
          v-model="keyword"
          :placeholder="$t('dataset.huggingface.searchPlaceholder')"
          clearable
          size="small"
          @keyup.enter.native="handleSearch"
          @clear="handleClear"
        >
          <el-button
            slot="append"
            icon="el-icon-search"
            :loading="searching"
            @click="handleSearch"
          >
            {{ $t("dataset.huggingface.searchButton") }}
          </el-button>
        </el-input>
      </div>

      <!-- Category chips -->
      <div class="hf-categories">
        <span class="hf-category-label">{{ $t("dataset.huggingface.categories.label") }}</span>
        <el-tag
          v-for="cat in categories"
          :key="cat.tag"
          size="small"
          :type="activeTag === cat.tag ? '' : 'info'"
          :effect="activeTag === cat.tag ? 'dark' : 'plain'"
          class="hf-category-tag"
          @click="toggleCategory(cat.tag)"
        >
          {{ cat.label }}
        </el-tag>
      </div>

      <!-- Sort bar -->
      <div class="hf-sort-bar">
        <el-radio-group v-model="sortBy" size="mini" @change="handleSearch">
          <el-radio-button label="trending">{{ $t("dataset.huggingface.sort.trending") }}</el-radio-button>
          <el-radio-button label="downloads">{{ $t("dataset.huggingface.sort.downloads") }}</el-radio-button>
          <el-radio-button label="likes">{{ $t("dataset.huggingface.sort.likes") }}</el-radio-button>
          <el-radio-button label="lastModified">{{ $t("dataset.huggingface.sort.recent") }}</el-radio-button>
        </el-radio-group>
        <span v-if="results.length" class="hf-result-count">
          {{ results.length }} {{ $t("dataset.huggingface.resultCount") }}
        </span>
      </div>

      <!-- Results table -->
      <el-table
        v-loading="searching"
        :data="results"
        :empty-text="emptyText"
        size="small"
        max-height="420"
        stripe
        highlight-current-row
        @row-click="openPullDialog"
        class="hf-table"
      >
        <el-table-column :label="$t('dataset.huggingface.columns.name')" min-width="220">
          <template #default="{ row }">
            <div class="hf-name">{{ row.id }}</div>
            <div v-if="row.author" class="hf-author">by {{ row.author }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.huggingface.columns.description')" min-width="260">
          <template #default="{ row }">
            <div class="hf-desc">{{ row.description || "-" }}</div>
            <div v-if="getMainTags(row.tags).length" class="hf-tags">
              <el-tag
                v-for="tag in getMainTags(row.tags)"
                :key="tag"
                size="mini"
                type="info"
                effect="plain"
                class="hf-tag"
              >
                {{ formatTag(tag) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.huggingface.columns.downloads')" width="90" align="right">
          <template #default="{ row }">
            <span class="hf-stat">{{ formatNumber(row.downloads) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.huggingface.columns.likes')" width="70" align="right">
          <template #default="{ row }">
            <span class="hf-stat">{{ formatNumber(row.likes) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.huggingface.columns.actions')" width="90" align="center">
          <template #default="{ row }">
            <el-button
              size="mini"
              :type="row.pulled ? 'success' : 'primary'"
              :loading="pullingRepo === row.id"
              @click.stop="openPullDialog(row)"
            >
              {{ row.pulled ? $t("dataset.huggingface.repullButton") : $t("dataset.huggingface.pullButton") }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Pull configuration dialog with detail info -->
    <el-dialog
      :title="pullDetail ? pullDetail.id : $t('dataset.huggingface.pullDialog.title')"
      :visible.sync="pullDialogVisible"
      width="560px"
      append-to-body
    >
      <div v-loading="detailLoading" class="pull-detail-body">
        <!-- Detail info section -->
        <div v-if="pullDetail" class="detail-section">
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.formats") }}</span>
              <el-tag
                v-for="fmt in pullDetail.fileFormats"
                :key="fmt"
                size="mini"
                effect="plain"
                class="detail-tag"
              >
                {{ fmt }}
              </el-tag>
              <span v-if="!pullDetail.fileFormats || !pullDetail.fileFormats.length" class="detail-empty">-</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.taskTypes") }}</span>
              <el-tag
                v-for="tt in pullDetail.taskTypes"
                :key="tt"
                size="mini"
                type="success"
                effect="plain"
                class="detail-tag"
              >
                {{ tt.replace(/-/g, ' ') }}
              </el-tag>
              <span v-if="!pullDetail.taskTypes || !pullDetail.taskTypes.length" class="detail-empty">-</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.size") }}</span>
              <span class="detail-value">{{ formatSize(pullDetail.totalSize) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.files") }}</span>
              <span class="detail-value">{{ pullDetail.fileCount || '-' }}</span>
            </div>
            <div v-if="pullDetail.languages && pullDetail.languages.length" class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.languages") }}</span>
              <el-tag
                v-for="lang in pullDetail.languages.slice(0, 5)"
                :key="lang"
                size="mini"
                type="warning"
                effect="plain"
                class="detail-tag"
              >
                {{ lang }}
              </el-tag>
            </div>
            <div v-if="pullDetail.license" class="detail-item">
              <span class="detail-label">{{ $t("dataset.huggingface.detail.license") }}</span>
              <span class="detail-value">{{ pullDetail.license }}</span>
            </div>
          </div>
        </div>
        <div v-else-if="!detailLoading" class="detail-error">
          {{ $t("dataset.huggingface.detail.loadFailed") }}
        </div>

        <el-divider />

        <!-- Pull form -->
        <el-form label-width="80px" size="small">
          <el-form-item label="Repo">
            <el-input :value="pullForm.repo" disabled />
          </el-form-item>
          <el-form-item :label="$t('dataset.huggingface.pullDialog.subset')">
            <el-input
              v-model="pullForm.subset"
              :placeholder="$t('dataset.huggingface.pullDialog.subsetPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="$t('dataset.huggingface.pullDialog.split')">
            <el-input
              v-model="pullForm.split"
              :placeholder="$t('dataset.huggingface.pullDialog.splitPlaceholder')"
            />
          </el-form-item>
        </el-form>
      </div>
      <span slot="footer">
        <el-button @click="pullDialogVisible = false">{{ $t("common.actions.cancel") }}</el-button>
        <el-button type="primary" :loading="pullingRepo === pullForm.repo" @click="handlePull">
          {{ pullForm.pulled ? $t("dataset.huggingface.repullButton") : $t("dataset.huggingface.pullButton") }}
        </el-button>
      </span>
    </el-dialog>
  </el-dialog>
</template>

<script>
import { searchHuggingFace, pullHuggingFaceDataset, getHuggingFaceDetail } from "@/api/dataset";
import { resolveApiErrorMessage } from "@/api/http";

const CATEGORIES = [
  { tag: "", labelKey: "all" },
  { tag: "task_categories:text-classification", labelKey: "textClassification" },
  { tag: "task_categories:question-answering", labelKey: "questionAnswering" },
  { tag: "task_categories:text-generation", labelKey: "textGeneration" },
  { tag: "task_categories:summarization", labelKey: "summarization" },
  { tag: "task_categories:translation", labelKey: "translation" },
  { tag: "task_categories:token-classification", labelKey: "tokenClassification" },
  { tag: "task_categories:other", labelKey: "other" },
];

export default {
  name: "HuggingFaceSearchDialog",
  props: {
    visible: { type: Boolean, default: false }
  },
  data() {
    return {
      keyword: "",
      searching: false,
      results: [],
      sortBy: "trending",
      activeTag: "",
      pullingRepo: null,
      pullDialogVisible: false,
      pullForm: { repo: "", subset: "", split: "", pulled: false },
      pullDetail: null,
      detailLoading: false,
      initialLoaded: false
    };
  },
  computed: {
    dialogVisible: {
      get() { return this.visible; },
      set(val) { this.$emit("update:visible", val); }
    },
    categories() {
      return CATEGORIES.map(c => ({
        tag: c.tag,
        label: this.$t(`dataset.huggingface.categories.${c.labelKey}`)
      }));
    },
    emptyText() {
      if (this.searching) return "";
      if (!this.initialLoaded) return this.$t("dataset.huggingface.loadingHint");
      return this.$t("dataset.huggingface.noResults");
    }
  },
  watch: {
    visible(val) {
      if (val && !this.initialLoaded) {
        this.loadTrending();
      }
    }
  },
  methods: {
    async loadTrending() {
      this.searching = true;
      try {
        const data = await searchHuggingFace("", { sort: "trending", limit: 20 });
        this.results = Array.isArray(data) ? data : data?.items || [];
        this.initialLoaded = true;
      } catch (error) {
        this.results = [];
        this.$message.error(resolveApiErrorMessage(error) || "Failed to load");
      } finally {
        this.searching = false;
      }
    },
    async handleSearch() {
      this.searching = true;
      try {
        const data = await searchHuggingFace(this.keyword.trim(), {
          sort: this.sortBy,
          tag: this.activeTag || undefined,
          limit: 25
        });
        this.results = Array.isArray(data) ? data : data?.items || [];
        this.initialLoaded = true;
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || "Search failed");
        this.results = [];
      } finally {
        this.searching = false;
      }
    },
    handleClear() {
      this.keyword = "";
      this.activeTag = "";
      this.loadTrending();
    },
    toggleCategory(tag) {
      this.activeTag = this.activeTag === tag ? "" : tag;
      this.handleSearch();
    },
    async openPullDialog(row) {
      this.pullForm = { repo: row.id, subset: "", split: "", pulled: !!row.pulled };
      this.pullDetail = null;
      this.pullDialogVisible = true;
      this.detailLoading = true;
      try {
        this.pullDetail = await getHuggingFaceDetail(row.id);
      } catch (error) {
        // Detail load failed, but dialog is still usable
        this.pullDetail = null;
      } finally {
        this.detailLoading = false;
      }
    },
    async handlePull() {
      this.pullingRepo = this.pullForm.repo;
      try {
        await pullHuggingFaceDataset(this.pullForm.repo, this.pullForm.subset, this.pullForm.split);
        this.$message.success(this.$t("dataset.huggingface.pullSuccess"));
        this.pullDialogVisible = false;
        this.$emit("pulled");
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.huggingface.pullFailed"));
      } finally {
        this.pullingRepo = null;
      }
    },
    getMainTags(tags) {
      if (!tags) return [];
      return tags
        .filter(t => t.startsWith("task_categories:") || t.startsWith("language:"))
        .slice(0, 3);
    },
    formatTag(tag) {
      return tag.replace(/^task_categories:/, "").replace(/^language:/, "🌐 ").replace(/-/g, " ");
    },
    formatNumber(n) {
      if (!n) return "-";
      if (n >= 1000000) return (n / 1000000).toFixed(1) + "M";
      if (n >= 1000) return (n / 1000).toFixed(1) + "K";
      return String(n);
    },
    formatSize(bytes) {
      if (!bytes) return "-";
      if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + " GB";
      if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + " MB";
      if (bytes >= 1024) return (bytes / 1024).toFixed(1) + " KB";
      return bytes + " B";
    },
    handleClose() {
      this.keyword = "";
    }
  }
};
</script>

<style scoped>
.hf-search-bar {
  margin-bottom: 12px;
}
.hf-categories {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}
.hf-category-label {
  font-size: 12px;
  color: #909399;
  margin-right: 4px;
  white-space: nowrap;
}
.hf-category-tag {
  cursor: pointer;
  transition: all 0.2s;
}
.hf-category-tag:hover {
  opacity: 0.8;
}
.hf-sort-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.hf-result-count {
  font-size: 12px;
  color: #909399;
}
.hf-table {
  cursor: pointer;
}
.hf-name {
  font-weight: 500;
  color: #303133;
  word-break: break-all;
  font-size: 13px;
}
.hf-author {
  font-size: 11px;
  color: #909399;
  margin-top: 1px;
}
.hf-desc {
  font-size: 12px;
  color: #606266;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.5;
}
.hf-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}
.hf-tag {
  font-size: 10px;
}
.hf-stat {
  font-size: 12px;
  color: #606266;
}
.pull-detail-body {
  min-height: 80px;
}
.detail-section {
  margin-bottom: 8px;
}
.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.detail-label {
  color: #909399;
  min-width: 70px;
  white-space: nowrap;
}
.detail-value {
  color: #303133;
  font-weight: 500;
}
.detail-tag {
  font-size: 11px;
}
.detail-empty {
  color: #c0c4cc;
}
.detail-error {
  text-align: center;
  padding: 20px;
  color: #e6a23c;
  font-size: 13px;
}
</style>
