<template>
  <div class="model-list">
    <PageHeader
      :title="$t('model.list.title')"
      :description="$t('model.list.description')"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          {{ $t("common.actions.refresh") }}
        </el-button>
        <el-button type="primary" icon="el-icon-plus" @click="openCreate">
          {{ $t("model.list.create") }}
        </el-button>
      </template>
    </PageHeader>

    <el-card shadow="never">
      <el-table
        v-loading="loading"
        :data="rows"
        :empty-text="emptyText"
        size="small"
        stripe
      >
        <el-table-column :label="$t('model.list.columns.model')" min-width="220">
          <template #default="{ row }">
            <div class="cell-main">
              <div class="cell-title">
                <span>{{ row.displayName || row.modelName }}</span>
                <el-tag v-if="row.version" size="mini" type="warning">v{{ row.version }}</el-tag>
              </div>
              <div class="cell-sub">{{ row.modelName }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('model.list.columns.provider')" prop="provider" width="160" />
        <el-table-column :label="$t('model.list.columns.baseUrl')" prop="baseUrl" min-width="240">
          <template #default="{ row }">
            <span class="mono">{{ row.baseUrl || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('model.list.columns.maskedKey')" min-width="160">
          <template #default="{ row }">
            <span class="mono">{{ row.maskedKey || "******" }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('model.list.columns.createdAt')" width="170">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column :label="$t('model.list.columns.actions')" width="160" align="right">
          <template #default="{ row }">
            <el-button type="text" size="mini" @click="openEdit(row)">
              {{ $t("common.actions.edit") }}
            </el-button>
            <el-button
              type="text"
              size="mini"
              class="danger-btn"
              @click="handleDelete(row)"
            >
              {{ $t("common.actions.delete") }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      :title="dialog.id ? $t('model.dialog.editTitle') : $t('model.dialog.createTitle')"
      :visible.sync="dialog.visible"
      width="540px"
      append-to-body
      @closed="resetDialog"
    >
      <el-form
        ref="dialogForm"
        :model="dialog.form"
        :rules="dialogRules"
        label-width="120px"
        size="small"
      >
        <el-form-item :label="$t('model.dialog.fields.provider')" prop="provider">
          <el-input v-model="dialog.form.provider" :placeholder="$t('model.dialog.fields.providerPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('model.dialog.fields.modelName')" prop="modelName">
          <el-input
            v-model="dialog.form.modelName"
            :placeholder="$t('model.dialog.fields.modelNamePlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="$t('model.dialog.fields.displayName')">
          <el-input
            v-model="dialog.form.displayName"
            :placeholder="$t('model.dialog.fields.displayNamePlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="$t('model.dialog.fields.version')">
          <el-input
            v-model="dialog.form.version"
            :placeholder="$t('model.dialog.fields.versionPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="$t('model.dialog.fields.baseUrl')" prop="baseUrl">
          <el-input v-model="dialog.form.baseUrl" :placeholder="$t('model.dialog.fields.baseUrlPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('model.dialog.fields.apiKey')" :prop="dialog.id ? '' : 'apiKey'">
          <el-input
            v-model="dialog.form.apiKey"
            type="password"
            show-password
            :placeholder="dialog.id ? $t('model.dialog.fields.apiKeyEditPlaceholder') : $t('model.dialog.fields.apiKeyCreatePlaceholder')"
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

    <!-- Delete confirm dialog -->
    <el-dialog
      :title="$t('model.messages.deleteConfirmTitle')"
      :visible.sync="deleteConfirm.visible"
      width="420px"
      :close-on-click-modal="false"
      append-to-body
    >
      <p v-if="deleteConfirm.target">{{ $t('model.messages.deleteConfirm', { name: deleteConfirm.target.displayName || deleteConfirm.target.modelName }) }}</p>
      <div slot="footer">
        <el-button @click="deleteConfirm.visible = false">{{ $t('common.actions.cancel') }}</el-button>
        <el-button type="danger" :loading="deleteConfirm.deleting" @click="confirmDelete">{{ $t('model.messages.deleteConfirmOk') }}</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";

import {
  listModels,
  createModel,
  updateModel,
  deleteModel
} from "@/api/model";
import { formatDateTime } from "@/utils/time";
import { resolveApiErrorMessage } from "@/api/http";

const buildDialog = () => ({
  visible: false,
  saving: false,
  id: null,
  form: {
    provider: "openai-compatible",
    modelName: "",
    displayName: "",
    version: "",
    baseUrl: "",
    apiKey: ""
  }
});

export default {
  name: "ModelListView",
  components: { PageHeader },
  data() {
    return {
      loading: false,
      rows: [],
      dialog: buildDialog(),
      deleteConfirm: {
        visible: false,
        deleting: false,
        target: null
      }
    };
  },
  computed: {
    emptyText() {
      return this.$t("model.list.empty");
    },
    dialogRules() {
      const t = (k) => this.$t(`model.dialog.rules.${k}`);
      return {
        provider: [{ required: true, message: t("providerRequired"), trigger: "blur" }],
        modelName: [{ required: true, message: t("modelNameRequired"), trigger: "blur" }],
        baseUrl: [{ required: true, message: t("baseUrlRequired"), trigger: "blur" }],
        apiKey: [
          { required: true, message: t("apiKeyRequired"), trigger: "blur" },
          { min: 16, message: t("apiKeyMinLength"), trigger: "blur" }
        ]
      };
    }
  },
  created() {
    this.loadList();
  },
  methods: {
    formatTime(s) {
      return formatDateTime(s, "—");
    },
    async loadList() {
      this.loading = true;
      try {
        const data = await listModels();
        const items = Array.isArray(data) ? data : data?.items || [];
        this.rows = items;
      } catch (error) {
        this.rows = [];
        this.$message.error(resolveApiErrorMessage(error) || this.$t("model.list.loadFailed"));
      } finally {
        this.loading = false;
      }
    },
    openCreate() {
      this.dialog = buildDialog();
      this.dialog.visible = true;
    },
    openEdit(row) {
      this.dialog = buildDialog();
      this.dialog.id = row.id;
      this.dialog.form.provider = row.provider;
      this.dialog.form.modelName = row.modelName;
      this.dialog.form.displayName = row.displayName;
      this.dialog.form.version = row.version || "";
      this.dialog.form.baseUrl = row.baseUrl;
      this.dialog.form.apiKey = "";
      this.dialog.visible = true;
    },
    resetDialog() {
      this.$refs.dialogForm?.resetFields();
    },
    async handleSave() {
      const valid = await this.$refs.dialogForm.validate().catch(() => false);
      if (!valid) return;
      const payload = { ...this.dialog.form };
      if (this.dialog.id && !payload.apiKey) delete payload.apiKey;
      this.dialog.saving = true;
      try {
        if (this.dialog.id) {
          await updateModel(this.dialog.id, payload);
        } else {
          await createModel(payload);
        }
        this.$message.success(this.$t("model.messages.saveSuccess"));
        this.dialog.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t("model.messages.saveFailed"));
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
        await deleteModel(row.id);
        this.$message.success(this.$t('model.messages.deleteSuccess'));
        this.deleteConfirm.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(resolveApiErrorMessage(error) || this.$t('model.messages.deleteFailed'));
      } finally {
        this.deleteConfirm.deleting = false;
      }
    }
  }
};
</script>

<style scoped>
.model-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.cell-main {
  display: flex;
  flex-direction: column;
}
.cell-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #303133;
  font-weight: 500;
}
.cell-sub {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
  font-family: "Menlo", "Monaco", monospace;
}
.mono {
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  color: #606266;
}
.danger-btn {
  color: #f56c6c;
}
</style>
