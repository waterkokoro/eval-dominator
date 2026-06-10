<template>
  <div class="dataset-list">
    <PageHeader
      :title="$t('dataset.list.title')"
      :description="$t('dataset.list.description')"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          {{ $t("common.actions.refresh") }}
        </el-button>
        <el-button
          icon="el-icon-magic-stick"
          :loading="syncing"
          @click="handleSync"
        >
          {{ $t("dataset.list.syncBuiltin") }}
        </el-button>
        <el-button icon="el-icon-download" @click="hfDialogVisible = true">
          {{ $t("dataset.list.importHuggingFace") }}
        </el-button>
        <el-button type="primary" icon="el-icon-plus" @click="openCreate">
          {{ $t("dataset.list.addCustom") }}
        </el-button>
      </template>
    </PageHeader>

    <el-card shadow="never" class="filter-card">
      <el-form :inline="true" size="small" @submit.native.prevent>
        <el-form-item :label="$t('dataset.list.filters.includeDisabled')">
          <el-switch v-model="includeDisabled" @change="loadList" />
        </el-form-item>
        <el-form-item :label="$t('dataset.list.filters.source')">
          <el-radio-group v-model="sourceFilter" @change="loadList">
            <el-radio-button label="all">{{ $t("dataset.list.filters.all") }}</el-radio-button>
            <el-radio-button label="builtin">{{ $t("dataset.list.filters.builtin") }}</el-radio-button>
            <el-radio-button label="custom">{{ $t("dataset.list.filters.custom") }}</el-radio-button>
            <el-radio-button label="huggingface">{{ $t("dataset.list.filters.huggingface") }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Demo datasets section -->
    <el-card v-if="demos.length" shadow="never" class="demo-card">
      <div slot="header" class="demo-header">
        <span>{{ $t("dataset.demo.title") }}</span>
        <span class="demo-desc">{{ $t("dataset.demo.description") }}</span>
      </div>
      <div class="demo-list">
        <el-tag
          v-for="demo in demos"
          :key="demo.name"
          size="medium"
          effect="plain"
          class="demo-tag"
        >
          <i class="el-icon-document" />
          {{ demo.name }}
          <span class="demo-meta">({{ demo.taskType }}, {{ demo.sampleCount }} samples)</span>
          <el-button
            size="mini"
            type="text"
            class="demo-import-btn"
            @click="handlePreviewDemo(demo)"
          >
            {{ $t("dataset.demo.previewButton") }}
          </el-button>
          <el-button
            size="mini"
            type="text"
            class="demo-import-btn"
            @click="handleImportDemo(demo)"
          >
            {{ $t("dataset.demo.importButton") }}
          </el-button>
        </el-tag>
      </div>
    </el-card>

    <el-card shadow="never" class="table-card">
      <el-table
        v-loading="loading"
        :data="filteredRows"
        :empty-text="emptyText"
        size="small"
        stripe
      >
        <el-table-column :label="$t('dataset.list.columns.dataset')" min-width="240">
          <template #default="{ row }">
            <div class="cell-main">
              <div class="cell-title">
                <span>{{ row.displayName }}</span>
                <el-tag size="mini" :type="sourceTagType(row.source)">
                  {{ $t(`dataset.source.${row.source}`) }}
                </el-tag>
              </div>
              <div class="cell-code">{{ row.code }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.list.columns.description')" min-width="280">
          <template #default="{ row }">
            <span class="description">{{ row.description || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.list.columns.inferenceMode')" width="160" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.inferenceMode"
              size="mini"
              :type="row.inferenceMode === 'gen' ? 'success' : 'warning'"
            >
              {{ row.inferenceMode === "gen" ? $t("dataset.list.inferenceMode.gen") : $t("dataset.list.inferenceMode.ppl") }}
            </el-tag>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.list.columns.sampleCount')" width="90" align="right">
          <template #default="{ row }">
            {{ row.sampleCount || "-" }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.list.columns.enabled')" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              :value="row.enabled"
              :active-color="enabledColor"
              @change="handleToggle(row, $event)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="$t('dataset.list.columns.actions')" width="220" align="right">
          <template #default="{ row }">
            <el-button size="mini" type="text" @click="handlePreview(row)">
              {{ $t("dataset.list.actions.preview") }}
            </el-button>
            <el-button size="mini" type="text" @click="useDataset(row)">
              {{ $t("dataset.list.actions.use") }}
            </el-button>
            <el-button
              v-if="row.source === 'custom'"
              size="mini"
              type="text"
              @click="openEdit(row)"
            >
              {{ $t("dataset.list.actions.edit") }}
            </el-button>
            <el-button
              v-if="row.source !== 'builtin'"
              size="mini"
              type="text"
              class="danger-btn"
              @click="handleDelete(row)"
            >
              {{ $t("dataset.list.actions.delete") }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create/Edit custom dataset dialog -->
    <el-dialog
      :title="dialog.id ? $t('dataset.dialog.editTitle') : $t('dataset.dialog.createTitle')"
      :visible.sync="dialog.visible"
      width="560px"
      append-to-body
      @closed="resetDialog"
    >
      <el-form
        ref="dialogForm"
        :model="dialog.form"
        :rules="dialogRules"
        label-width="100px"
        size="small"
      >
        <template v-if="!dialog.id">
          <el-form-item :label="$t('dataset.dialog.fields.dataSource')">
            <el-radio-group v-model="dialog.form.dataSource" @change="onDataSourceChange">
              <el-radio label="upload">{{ $t("dataset.dialog.fields.fileUpload") }}</el-radio>
              <el-radio label="path">{{ $t("dataset.dialog.fields.localPath") }}</el-radio>
            </el-radio-group>
          </el-form-item>
        </template>

        <el-form-item v-if="dialog.form.dataSource === 'upload' && !dialog.id" :label="$t('dataset.dialog.fields.fileUpload')">
          <el-upload
            ref="fileUpload"
            :auto-upload="false"
            :limit="1"
            :on-change="onFileChange"
            :on-remove="onFileRemove"
            accept=".csv,.jsonl,.json"
            drag
          >
            <i class="el-icon-upload" />
            <div>CSV / JSONL / JSON</div>
          </el-upload>
        </el-form-item>

        <el-form-item v-if="dialog.form.dataSource === 'path' && !dialog.id" :label="$t('dataset.dialog.fields.localPath')">
          <el-input
            v-model="dialog.form.localPath"
            :placeholder="$t('dataset.dialog.fields.localPathPlaceholder')"
          />
        </el-form-item>

        <el-form-item v-if="!dialog.id" :label="$t('dataset.dialog.fields.taskType')">
          <el-radio-group v-model="dialog.form.taskType">
            <el-radio label="choice">{{ $t("dataset.dialog.fields.taskTypeChoice") }}</el-radio>
            <el-radio label="qa">{{ $t("dataset.dialog.fields.taskTypeQA") }}</el-radio>
            <el-radio label="classification">{{ $t("dataset.dialog.fields.taskTypeClassification") }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="dialog.id" :label="$t('dataset.dialog.fields.code')">
          <el-input v-model="dialog.form.code" disabled />
        </el-form-item>
        <el-form-item :label="$t('dataset.dialog.fields.displayName')" prop="displayName">
          <el-input v-model="dialog.form.displayName" :placeholder="$t('dataset.dialog.fields.displayNamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('dataset.dialog.fields.description')">
          <el-input
            v-model="dialog.form.description"
            type="textarea"
            :rows="2"
            :placeholder="$t('dataset.dialog.fields.descriptionPlaceholder')"
          />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialog.visible = false">{{ $t("common.actions.cancel") }}</el-button>
        <el-button type="primary" :loading="dialog.saving" @click="handleSave">
          {{ $t("common.actions.save") }}
        </el-button>
      </span>
    </el-dialog>

    <!-- HuggingFace search dialog -->
    <HuggingFaceSearchDialog
      :visible.sync="hfDialogVisible"
      @pulled="loadList"
    />

    <!-- Delete confirm dialog -->
    <el-dialog
      :title="$t('dataset.messages.deleteConfirmTitle')"
      :visible.sync="deleteConfirm.visible"
      width="420px"
      :close-on-click-modal="false"
      append-to-body
    >
      <p v-if="deleteConfirm.target">{{ $t('dataset.messages.deleteConfirm', { name: deleteConfirm.target.displayName }) }}</p>
      <div slot="footer">
        <el-button @click="deleteConfirm.visible = false">{{ $t('common.actions.cancel') }}</el-button>
        <el-button type="danger" :loading="deleteConfirm.deleting" @click="confirmDelete">{{ $t('dataset.messages.deleteConfirmOk') }}</el-button>
      </div>
    </el-dialog>

    <!-- Preview dialog -->
    <el-dialog
      :title="preview.title"
      :visible.sync="preview.visible"
      width="80%"
      append-to-body
      top="5vh"
    >
      <div v-loading="preview.loading" class="preview-body">
        <div v-if="preview.error" class="preview-error">
          <i class="el-icon-warning" /> {{ preview.error }}
        </div>
        <template v-else-if="preview.data">
          <div class="preview-meta">
            <span>{{ $t("dataset.preview.format") }}: <el-tag size="mini">{{ preview.data.fileFormat }}</el-tag></span>
            <span>{{ $t("dataset.preview.total") }}: <strong>{{ preview.data.total }}</strong></span>
            <span>{{ $t("dataset.preview.showing") }}: <strong>{{ preview.data.previewSize }}</strong></span>
          </div>
          <div v-if="preview.data.totalColumns" class="preview-warning">
            <i class="el-icon-warning" />
            {{ $t("dataset.preview.columnsTruncated", { total: preview.data.totalColumns, shown: preview.data.headers.length }) }}
          </div>
          <el-table
            :data="preview.data.rows"
            size="mini"
            stripe
            border
            max-height="480"
            class="preview-table"
          >
            <el-table-column
              v-for="header in preview.data.headers"
              :key="header"
              :prop="header"
              :label="header"
              min-width="140"
              show-overflow-tooltip
            >
              <template #default="{ row }">
                <span class="preview-cell">{{ formatCellValue(row[header]) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";
import HuggingFaceSearchDialog from "./HuggingFaceSearchDialog.vue";

import {
  listDatasets,
  createDataset,
  updateDataset,
  setDatasetEnabled,
  deleteDataset,
  syncDatasets,
  uploadDatasetFile,
  createCustomFromPath,
  getDemoDatasets,
  previewDataset,
  previewDatasetByPath
} from "@/api/dataset";
import { resolveApiErrorMessage } from "@/api/http";

const buildDialog = () => ({
  visible: false,
  saving: false,
  id: null,
  form: {
    dataSource: "upload",
    taskType: "qa",
    displayName: "",
    description: "",
    localPath: "",
    code: "",
    file: null
  }
});

export default {
  name: "DatasetListView",
  components: { PageHeader, HuggingFaceSearchDialog },
  data() {
    return {
      loading: false,
      syncing: false,
      includeDisabled: true,
      sourceFilter: "all",
      rows: [],
      demos: [],
      dialog: buildDialog(),
      enabledColor: "#67c23a",
      hfDialogVisible: false,
      preview: {
        visible: false,
        loading: false,
        title: "",
        data: null,
        error: null
      },
      deleteConfirm: {
        visible: false,
        deleting: false,
        target: null
      }
    };
  },
  computed: {
    filteredRows() {
      if (this.sourceFilter === "all") return this.rows;
      return this.rows.filter((row) => row.source === this.sourceFilter);
    },
    emptyText() {
      return this.$t("dataset.list.empty");
    },
    dialogRules() {
      const t = (k) => this.$t(`dataset.dialog.rules.${k}`);
      return {
        displayName: [{ required: true, message: t("displayNameRequired"), trigger: "blur" }]
      };
    }
  },
  created() {
    this.loadList();
    this.loadDemos();
  },
  methods: {
    async loadList() {
      this.loading = true;
      try {
        const data = await listDatasets(this.includeDisabled);
        this.rows = Array.isArray(data) ? data : data?.items || [];
      } catch (error) {
        this.rows = [];
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.list.loadFailed"));
      } finally {
        this.loading = false;
      }
    },
    async loadDemos() {
      try {
        const data = await getDemoDatasets();
        this.demos = Array.isArray(data) ? data : data?.items || [];
      } catch {
        this.demos = [];
      }
    },
    async handleSync() {
      this.syncing = true;
      try {
        const result = await syncDatasets();
        this.$message.success(
          this.$t("dataset.messages.syncSuccess", {
            scanned: result.scanned,
            inserted: result.inserted,
            updated: result.updated
          })
        );
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.messages.syncFailed"));
      } finally {
        this.syncing = false;
      }
    },
    async handleToggle(row, enabled) {
      try {
        await setDatasetEnabled(row.id, enabled);
        row.enabled = enabled;
        this.$message.success(enabled ? this.$t("dataset.messages.toggleEnabled") : this.$t("dataset.messages.toggleDisabled"));
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.messages.toggleFailed"));
      }
    },
    openCreate() {
      this.dialog = buildDialog();
      this.dialog.visible = true;
    },
    openEdit(row) {
      this.dialog = buildDialog();
      this.dialog.id = row.id;
      this.dialog.form.code = row.code;
      this.dialog.form.displayName = row.displayName;
      this.dialog.form.description = row.description;
      this.dialog.visible = true;
    },
    resetDialog() {
      this.$refs.dialogForm?.resetFields();
      this.dialog.form.file = null;
    },
    onDataSourceChange() {
      this.dialog.form.file = null;
      this.dialog.form.localPath = "";
    },
    onFileChange(file) {
      this.dialog.form.file = file.raw;
    },
    onFileRemove() {
      this.dialog.form.file = null;
    },
    async handleSave() {
      const valid = await this.$refs.dialogForm.validate().catch(() => false);
      if (!valid) return;
      this.dialog.saving = true;
      try {
        if (this.dialog.id) {
          await updateDataset(this.dialog.id, {
            displayName: this.dialog.form.displayName,
            description: this.dialog.form.description,
            enabled: true
          });
        } else if (this.dialog.form.dataSource === "upload" && this.dialog.form.file) {
          await uploadDatasetFile(this.dialog.form.file, {
            displayName: this.dialog.form.displayName,
            description: this.dialog.form.description,
            taskType: this.dialog.form.taskType
          });
        } else if (this.dialog.form.dataSource === "path" && this.dialog.form.localPath) {
          await createCustomFromPath({
            displayName: this.dialog.form.displayName,
            description: this.dialog.form.description,
            localPath: this.dialog.form.localPath,
            taskType: this.dialog.form.taskType
          });
        } else {
          this.$message.warning("Please select a data source");
          this.dialog.saving = false;
          return;
        }
        this.$message.success(this.$t("dataset.messages.saveSuccess"));
        this.dialog.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.messages.saveFailed"));
      } finally {
        this.dialog.saving = false;
      }
    },
    handleDelete(row) {
      this.deleteConfirm.target = row;
      this.deleteConfirm.visible = true;
    },
    async confirmDelete() {
      const row = this.deleteConfirm.target;
      if (!row) return;
      this.deleteConfirm.deleting = true;
      try {
        await deleteDataset(row.id);
        this.$message.success(this.$t("dataset.messages.deleteSuccess"));
        this.deleteConfirm.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.messages.deleteFailed"));
      } finally {
        this.deleteConfirm.deleting = false;
      }
    },
    async handleImportDemo(demo) {
      try {
        await createCustomFromPath({
          code: `demo_${demo.name}`,
          displayName: `Demo: ${demo.name}`,
          description: demo.description,
          localPath: demo.path,
          taskType: demo.taskType
        });
        this.$message.success(this.$t("dataset.demo.importSuccess"));
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error));
      }
    },
    useDataset(row) {
      this.$router.push({
        name: "eval-submit",
        query: {
          datasetType: row.type,
          datasetName: row.code
        }
      });
    },
    async handlePreview(row) {
      this.preview = {
        visible: true,
        loading: true,
        title: `${this.$t("dataset.preview.title")} - ${row.displayName}`,
        data: null,
        error: null
      };
      try {
        const data = await previewDataset(row.id, 20);
        this.preview.data = data;
      } catch (error) {
        this.preview.error = resolveApiErrorMessage(error) || this.$t("dataset.preview.failed");
      } finally {
        this.preview.loading = false;
      }
    },
    async handlePreviewDemo(demo) {
      this.preview = {
        visible: true,
        loading: true,
        title: `${this.$t("dataset.preview.title")} - ${demo.name}`,
        data: null,
        error: null
      };
      try {
        const data = await previewDatasetByPath(demo.path, 20);
        this.preview.data = data;
      } catch (error) {
        this.preview.error = resolveApiErrorMessage(error) || this.$t("dataset.preview.failed");
      } finally {
        this.preview.loading = false;
      }
    },
    formatCellValue(val) {
      if (val === null || val === undefined) return "";
      if (typeof val === "object") return JSON.stringify(val);
      return String(val);
    },
    sourceTagType(source) {
      if (source === "builtin") return "info";
      if (source === "huggingface") return "success";
      return "warning";
    }
  }
};
</script>

<style scoped>
.dataset-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.filter-card,
.table-card,
.demo-card {
  border-radius: 8px;
}
.demo-header {
  display: flex;
  align-items: center;
  gap: 12px;
  font-weight: 500;
}
.demo-desc {
  font-size: 12px;
  color: #909399;
  font-weight: normal;
}
.demo-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.demo-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.demo-meta {
  font-size: 11px;
  color: #909399;
}
.demo-import-btn {
  margin-left: 4px;
  padding: 0 4px;
}
.cell-main {
  display: flex;
  flex-direction: column;
}
.cell-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #303133;
}
.cell-code {
  font-size: 12px;
  color: #909399;
  font-family: "Menlo", "Monaco", monospace;
  margin-top: 2px;
}
.description {
  font-size: 12px;
  color: #606266;
  line-height: 1.6;
}
.danger-btn {
  color: #f56c6c;
}
.muted {
  color: #c0c4cc;
}
.preview-body {
  min-height: 120px;
}
.preview-error {
  text-align: center;
  padding: 40px 0;
  color: #e6a23c;
  font-size: 14px;
}
.preview-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #606266;
}
.preview-warning {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  padding: 6px 12px;
  font-size: 12px;
  color: #e6a23c;
  background: #fdf6ec;
  border-radius: 4px;
}
.preview-table {
  width: 100%;
}
.preview-cell {
  word-break: break-all;
  font-size: 12px;
  line-height: 1.4;
}
</style>
