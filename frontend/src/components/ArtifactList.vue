<template>
  <div>
    <el-table v-if="rows.length" :data="rows" border size="small">
      <el-table-column prop="label" label="类型" width="160">
        <template #default="{ row }">
          <el-tag size="small" :type="row.tagType">{{ row.label }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="path" label="路径" min-width="280">
        <template #default="{ row }">
          <span class="artifact-path" :title="row.path">{{ row.path || "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" align="right">
        <template #default="{ row }">
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handlePreview(row)"
          >
            预览
          </el-button>
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handleDownload(row)"
          >
            下载
          </el-button>
          <el-button
            size="mini"
            type="text"
            :disabled="!row.path"
            @click="handleCopy(row)"
          >
            复制路径
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <EmptyState
      v-else
      type="todo"
      title="暂无产物"
      description="任务完成后将在此展示报告、原始结果与日志路径"
    />
    <p class="artifact-tip">
      产物文件均位于服务端 <code>runtime/</code> 输出目录下，「预览」最多展示 512KB 文本，超出请用「下载」。
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
          title="内容超过 512KB，已截断显示。完整内容请下载。"
          style="margin-bottom: 12px"
        />
        <div class="preview-meta">
          <span class="preview-meta-item"><strong>路径：</strong>{{ preview.relativePath || preview.path }}</span>
          <span class="preview-meta-item"><strong>大小：</strong>{{ formatSize(preview.size) }}</span>
        </div>
        <pre class="preview-content">{{ preview.content || (preview.error || "(空)") }}</pre>
      </div>
      <template #footer>
        <el-button v-if="preview.path" @click="handleDownload({ path: preview.path, label: preview.title })">
          下载完整文件
        </el-button>
        <el-button type="primary" @click="preview.visible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";
import { previewArtifact, downloadArtifact } from "@/api/artifact";

const TYPE_LABELS = {
  report: { label: "评测报告", tagType: "success" },
  raw_result: { label: "原始结果", tagType: "warning" },
  log: { label: "执行日志", tagType: "info" },
  config: { label: "评测配置", tagType: "" },
  other: { label: "其他", tagType: "info" }
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
        list.push({ key: "report", ...TYPE_LABELS.report, path: this.reportPath });
      }
      if (this.rawResultPath) {
        list.push({
          key: "raw_result",
          ...TYPE_LABELS.raw_result,
          path: this.rawResultPath
        });
      }
      if (this.logPath) {
        list.push({ key: "log", ...TYPE_LABELS.log, path: this.logPath });
      }

      const parsed = this.parseArtifacts();
      parsed.forEach((item) => {
        const meta = TYPE_LABELS[item.type] || TYPE_LABELS.other;
        list.push({
          key: `${item.type || "other"}-${item.name || item.path}`,
          label: item.name || meta.label,
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
        title: row.label || "产物预览",
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
          this.preview.error = "二进制文件不支持预览，请下载查看。";
        } else {
          this.preview.content = resp?.content ?? "";
        }
      } catch (e) {
        this.preview.error = e?.response?.data?.message || e?.message || "预览失败";
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
        this.$message.error(e?.response?.data?.message || e?.message || "下载失败");
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
      this.$message.success("路径已复制");
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
