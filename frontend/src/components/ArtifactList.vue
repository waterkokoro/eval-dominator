<template>
  <div>
    <el-table v-if="rows.length" :data="rows" border size="small">
      <el-table-column prop="label" :label="$t('eval.artifacts.columns.type')" width="160">
        <template #default="{ row }">
          <el-tag size="small" :type="row.tagType">{{ row.label }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="path" :label="$t('eval.artifacts.columns.path')" min-width="280">
        <template #default="{ row }">
          <span class="artifact-path" :title="row.path">{{ row.path || "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('eval.artifacts.columns.actions')" width="240" align="right">
        <template #default="{ row }">
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handlePreview(row)"
          >
            {{ $t("eval.artifacts.actions.preview") }}
          </el-button>
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handleDownload(row)"
          >
            {{ $t("eval.artifacts.actions.download") }}
          </el-button>
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handleCopy(row)"
          >
            {{ $t("eval.artifacts.actions.copy") }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <EmptyState
      v-else
      type="todo"
      :title="$t('eval.artifacts.empty.title')"
      :description="$t('eval.artifacts.empty.description')"
    />
    <p class="artifact-tip">
      <i18n path="eval.artifacts.tip" tag="span">
        <template #dir>
          <code>runtime/</code>
        </template>
      </i18n>
    </p>

    <el-dialog
      :visible.sync="preview.visible"
      :title="preview.title"
      width="80%"
      top="6vh"
      append-to-body
      :destroy-on-close="true"
    >
      <div v-loading="preview.loading">
        <el-alert
          v-if="preview.truncated"
          type="warning"
          show-icon
          :closable="false"
          :title="$t('eval.artifacts.preview.truncated')"
          style="margin-bottom: 12px"
        />
        <div class="preview-meta">
          <span class="preview-meta-item">
            <strong>{{ $t("eval.artifacts.preview.path") }}</strong>{{ preview.relativePath || preview.path }}
          </span>
          <span class="preview-meta-item">
            <strong>{{ $t("eval.artifacts.preview.size") }}</strong>{{ formatSize(preview.size) }}
          </span>
        </div>
        <pre class="preview-content">{{ preview.content || (preview.error || $t("eval.artifacts.preview.empty")) }}</pre>
      </div>
      <template #footer>
        <el-button v-if="preview.path" @click="handleDownload({ path: preview.path, label: preview.title })">
          {{ $t("eval.artifacts.actions.downloadFull") }}
        </el-button>
        <el-button type="primary" @click="preview.visible = false">
          {{ $t("common.actions.close") }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";
import { previewArtifact, downloadArtifact } from "@/api/artifact";
import { resolveApiErrorMessage } from "@/api/http";

const TYPE_KEYS = {
  report: { tagType: "success" },
  raw_result: { tagType: "warning" },
  log: { tagType: "info" },
  config: { tagType: "" },
  other: { tagType: "info" }
};

export default {
  name: "ArtifactList",
  components: { EmptyState },
  props: {
    evalTaskId: { type: String, default: "" },
    reportPath: { type: String, default: "" },
    rawResultPath: { type: String, default: "" },
    logPath: { type: String, default: "" },
    artifacts: { type: [Array, String, Object], default: () => [] }
  },
  data() {
    return {
      preview: {
        visible: false,
        loading: false,
        title: "",
        path: "",
        relativePath: "",
        size: 0,
        truncated: false,
        content: "",
        error: ""
      }
    };
  },
  computed: {
    rows() {
      const list = [];
      if (this.reportPath) {
        list.push({
          key: "report",
          label: this.typeLabel("report"),
          tagType: TYPE_KEYS.report.tagType,
          path: this.reportPath
        });
      }
      if (this.rawResultPath) {
        list.push({
          key: "raw_result",
          label: this.typeLabel("raw_result"),
          tagType: TYPE_KEYS.raw_result.tagType,
          path: this.rawResultPath
        });
      }
      if (this.logPath) {
        list.push({
          key: "log",
          label: this.typeLabel("log"),
          tagType: TYPE_KEYS.log.tagType,
          path: this.logPath
        });
      }

      const parsed = this.parseArtifacts();
      parsed.forEach((item) => {
        const meta = TYPE_KEYS[item.type] || TYPE_KEYS.other;
        list.push({
          key: `${item.type || "other"}-${item.name || item.path}`,
          label: item.name || this.typeLabel(item.type || "other"),
          tagType: meta.tagType,
          path: item.path
        });
      });

      const seen = new Set();
      return list.filter((row) => {
        const id = row.path;
        if (!id || seen.has(id)) return false;
        seen.add(id);
        return true;
      });
    }
  },
  methods: {
    typeLabel(type) {
      const key = TYPE_KEYS[type] ? type : "other";
      return this.$t(`eval.artifacts.types.${key}`);
    },
    parseArtifacts() {
      const raw = this.artifacts;
      if (!raw) return [];
      let parsed = raw;
      if (typeof raw === "string") {
        try {
          parsed = JSON.parse(raw);
        } catch (e) {
          return [];
        }
      }
      if (Array.isArray(parsed)) return parsed;
      if (parsed && Array.isArray(parsed.artifacts)) return parsed.artifacts;
      return [];
    },
    async handlePreview(row) {
      if (!this.evalTaskId || !row.path) return;
      this.preview = {
        visible: true,
        loading: true,
        title: row.label || this.$t("eval.artifacts.preview.fallbackTitle"),
        path: row.path,
        relativePath: "",
        size: 0,
        truncated: false,
        content: "",
        error: ""
      };
      try {
        const resp = await previewArtifact(this.evalTaskId, row.path);
        this.preview.relativePath = resp?.relativePath || "";
        this.preview.size = resp?.size || 0;
        this.preview.truncated = !!resp?.truncated;
        if (resp?.isText === false) {
          this.preview.content = "";
          this.preview.error = this.$t("eval.artifacts.preview.binary");
        } else {
          this.preview.content = resp?.content ?? "";
        }
      } catch (e) {
        this.preview.error = resolveApiErrorMessage(e) || this.$t("eval.artifacts.preview.failed");
      } finally {
        this.preview.loading = false;
      }
    },
    async handleDownload(row) {
      if (!this.evalTaskId || !row.path) return;
      try {
        const filename = row.path.split(/[\\/]/).pop();
        await downloadArtifact(this.evalTaskId, row.path, filename);
      } catch (e) {
        this.$message.error(resolveApiErrorMessage(e) || this.$t("eval.artifacts.preview.downloadFailed"));
      }
    },
    handleCopy(row) {
      if (!row.path) return;
      const text = row.path;
      const fallback = () => {
        const input = document.createElement("textarea");
        input.value = text;
        document.body.appendChild(input);
        input.select();
        document.execCommand("copy");
        document.body.removeChild(input);
      };
      if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text).catch(fallback);
      } else {
        fallback();
      }
      this.$message.success(this.$t("common.messages.copySuccess"));
    },
    formatSize(bytes) {
      if (!bytes && bytes !== 0) return "—";
      if (bytes < 1024) return `${bytes} B`;
      if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
      return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
    }
  }
};
</script>

<style scoped>
.artifact-path {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #606266;
  word-break: break-all;
}
.artifact-tip {
  margin-top: 12px;
  font-size: 12px;
  color: #909399;
}
.artifact-tip code {
  padding: 1px 6px;
  background: #f4f4f5;
  border-radius: 4px;
  font-size: 12px;
}

.preview-meta {
  margin-bottom: 8px;
  color: #606266;
  font-size: 12px;
}
.preview-meta-item {
  margin-right: 16px;
}

.preview-content {
  background: #1f2329;
  color: #f5f6f7;
  border-radius: 4px;
  padding: 12px;
  max-height: 60vh;
  overflow: auto;
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
