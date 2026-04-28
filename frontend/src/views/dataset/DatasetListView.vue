<template>
  <div class="dataset-list">
    <PageHeader
      title="数据集中心"
      description="集中管理可用于评测的数据集，提交评测时直接选用"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          刷新
        </el-button>
        <el-button
          icon="el-icon-magic-stick"
          :loading="syncing"
          @click="handleSync"
        >
          同步内置数据集
        </el-button>
        <el-button type="primary" icon="el-icon-plus" @click="openCreate">
          添加自定义
        </el-button>
      </template>
    </PageHeader>

    <el-card shadow="never" class="filter-card">
      <el-form :inline="true" size="small" @submit.native.prevent>
        <el-form-item label="包含禁用">
          <el-switch v-model="includeDisabled" @change="loadList" />
        </el-form-item>
        <el-form-item label="来源">
          <el-radio-group v-model="sourceFilter" @change="loadList">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="builtin">内置</el-radio-button>
            <el-radio-button label="custom">自定义</el-radio-button>
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
        <el-table-column label="数据集" min-width="240">
          <template #default="{ row }">
            <div class="cell-main">
              <div class="cell-title">
                <span>{{ row.displayName }}</span>
                <el-tag size="mini" :type="row.source === 'builtin' ? 'info' : 'warning'">
                  {{ row.source === "builtin" ? "内置" : "自定义" }}
                </el-tag>
              </div>
              <div class="cell-code">{{ row.code }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="说明" min-width="280">
          <template #default="{ row }">
            <span class="description">{{ row.description || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column label="推理方式" width="140" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.inferenceMode"
              size="mini"
              :type="row.inferenceMode === 'gen' ? 'success' : 'warning'"
            >
              {{ row.inferenceMode === "gen" ? "GEN（生成式）" : "PPL（困惑度）" }}
            </el-tag>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="样本数" width="90" align="right">
          <template #default="{ row }">
            {{ row.sampleCount || "-" }}
          </template>
        </el-table-column>
        <el-table-column label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              :value="row.enabled"
              :active-color="enabledColor"
              @change="handleToggle(row, $event)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="right">
          <template #default="{ row }">
            <el-button size="mini" type="text" @click="useDataset(row)">
              用于评测
            </el-button>
            <el-button
              v-if="row.source === 'custom'"
              size="mini"
              type="text"
              @click="openEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              v-if="row.source === 'custom'"
              size="mini"
              type="text"
              class="danger-btn"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      :title="dialog.id ? '编辑自定义数据集' : '添加自定义数据集'"
      :visible.sync="dialog.visible"
      width="540px"
      append-to-body
      @closed="resetDialog"
    >
      <el-form
        ref="dialogForm"
        :model="dialog.form"
        :rules="dialog.rules"
        label-width="100px"
        size="small"
      >
        <el-form-item label="数据集 Code" prop="code">
          <el-input
            v-model="dialog.form.code"
            :disabled="!!dialog.id"
            placeholder="OpenCompass dataset config 名，例如 demo_gsm8k_chat_gen"
          />
        </el-form-item>
        <el-form-item label="显示名称" prop="displayName">
          <el-input v-model="dialog.form.displayName" placeholder="界面展示名" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input
            v-model="dialog.form.description"
            type="textarea"
            :rows="2"
            placeholder="数据集说明（可选）"
          />
        </el-form-item>
        <el-form-item label="样本数">
          <el-input-number
            v-model="dialog.form.sampleCount"
            :min="0"
            controls-position="right"
          />
        </el-form-item>
        <el-form-item label="类型">
          <el-input v-model="dialog.form.type" placeholder="custom / opencompass_standard ..." />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="dialog.saving" @click="handleSave">
          保存
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
  },
  rules: {
    code: [{ required: true, message: "请填写 Code", trigger: "blur" }],
    displayName: [{ required: true, message: "请填写显示名称", trigger: "blur" }]
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
      return "暂无数据集，可点「同步内置数据集」或「添加自定义」";
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
        this.$message.error(error?.response?.data?.message || "加载数据集失败");
      } finally {
        this.loading = false;
      }
    },
    async handleSync() {
      this.syncing = true;
      try {
        const result = await syncDatasets();
        this.$message.success(
          `同步完成：扫描 ${result.scanned}，新增 ${result.inserted}，更新 ${result.updated}`
        );
        this.loadList();
      } catch (error) {
        this.$message.error(error?.response?.data?.message || "同步失败");
      } finally {
        this.syncing = false;
      }
    },
    async handleToggle(row, enabled) {
      try {
        await setDatasetEnabled(row.id, enabled);
        row.enabled = enabled;
        this.$message.success(enabled ? "已启用" : "已禁用");
      } catch (error) {
        this.$message.error(error?.response?.data?.message || "切换失败");
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
        this.$message.success("保存成功");
        this.dialog.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(error?.response?.data?.message || "保存失败");
      } finally {
        this.dialog.saving = false;
      }
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确认删除「${row.displayName}」？`, "提示", {
          type: "warning"
        });
      } catch (e) {
        return;
      }
      try {
        await deleteDataset(row.id);
        this.$message.success("已删除");
        this.loadList();
      } catch (error) {
        this.$message.error(error?.response?.data?.message || "删除失败");
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
