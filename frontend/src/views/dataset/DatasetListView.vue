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
          </el-radio-group>
        </el-form-item>
      </el-form>
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
                <el-tag size="mini" :type="row.source === 'builtin' ? 'info' : 'warning'">
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
        <el-table-column :label="$t('dataset.list.columns.actions')" width="180" align="right">
          <template #default="{ row }">
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
              v-if="row.source === 'custom'"
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

    <el-dialog
      :title="dialog.id ? $t('dataset.dialog.editTitle') : $t('dataset.dialog.createTitle')"
      :visible.sync="dialog.visible"
      width="540px"
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
        <el-form-item :label="$t('dataset.dialog.fields.code')" prop="code">
          <el-input
            v-model="dialog.form.code"
            :disabled="!!dialog.id"
            :placeholder="$t('dataset.dialog.fields.codePlaceholder')"
          />
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
        <el-form-item :label="$t('dataset.dialog.fields.sampleCount')">
          <el-input-number
            v-model="dialog.form.sampleCount"
            :min="0"
            controls-position="right"
          />
        </el-form-item>
        <el-form-item :label="$t('dataset.dialog.fields.type')">
          <el-input v-model="dialog.form.type" :placeholder="$t('dataset.dialog.fields.typePlaceholder')" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialog.visible = false">{{ $t("common.actions.cancel") }}</el-button>
        <el-button type="primary" :loading="dialog.saving" @click="handleSave">
          {{ $t("common.actions.save") }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";

import {
  listDatasets,
  createDataset,
  updateDataset,
  setDatasetEnabled,
  deleteDataset,
  syncDatasets
} from "@/api/dataset";
import { resolveApiErrorMessage } from "@/api/http";

const buildDialog = () => ({
  visible: false,
  saving: false,
  id: null,
  form: {
    code: "",
    displayName: "",
    description: "",
    sampleCount: 0,
    type: "custom"
  }
});

export default {
  name: "DatasetListView",
  components: { PageHeader },
  data() {
    return {
      loading: false,
      syncing: false,
      includeDisabled: true,
      sourceFilter: "all",
      rows: [],
      dialog: buildDialog(),
      enabledColor: "#67c23a"
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
        code: [{ required: true, message: t("codeRequired"), trigger: "blur" }],
        displayName: [{ required: true, message: t("displayNameRequired"), trigger: "blur" }]
      };
    }
  },
  created() {
    this.loadList();
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
      this.dialog.form.sampleCount = row.sampleCount;
      this.dialog.form.type = row.type || "custom";
      this.dialog.visible = true;
    },
    resetDialog() {
      this.$refs.dialogForm?.resetFields();
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
            type: this.dialog.form.type,
            sampleCount: this.dialog.form.sampleCount,
            enabled: true
          });
        } else {
          await createDataset({
            code: this.dialog.form.code,
            displayName: this.dialog.form.displayName,
            description: this.dialog.form.description,
            type: this.dialog.form.type,
            sampleCount: this.dialog.form.sampleCount
          });
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
    async handleDelete(row) {
      try {
        await this.$confirm(
          this.$t("dataset.messages.deleteConfirm", { name: row.displayName }),
          this.$t("common.messages.tip"),
          { type: "warning" }
        );
      } catch (e) {
        return;
      }
      try {
        await deleteDataset(row.id);
        this.$message.success(this.$t("dataset.messages.deleteSuccess"));
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("dataset.messages.deleteFailed"));
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
.table-card {
  border-radius: 8px;
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
</style>
