<template>
  <div class="model-list">
    <PageHeader
      title="模型管理"
      description="管理可用于评测的模型预设：服务商 + 模型名 + Base URL + API Key"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadList">
          刷新
        </el-button>
        <el-button type="primary" icon="el-icon-plus" @click="openCreate">
          新增模型
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
        <el-table-column label="模型" min-width="220">
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
        <el-table-column label="服务商" prop="provider" width="160" />
        <el-table-column label="Base URL" prop="baseUrl" min-width="240">
          <template #default="{ row }">
            <span class="mono">{{ row.baseUrl || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column label="脱敏 Key" min-width="160">
          <template #default="{ row }">
            <span class="mono">{{ row.maskedKey || "******" }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="right">
          <template #default="{ row }">
            <el-button type="text" size="mini" @click="openEdit(row)">
              编辑
            </el-button>
            <el-button
              type="text"
              size="mini"
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
      :title="dialog.id ? '编辑模型' : '新增模型'"
      :visible.sync="dialog.visible"
      width="540px"
      append-to-body
      @closed="resetDialog"
    >
      <el-form
        ref="dialogForm"
        :model="dialog.form"
        :rules="dialog.rules"
        label-width="120px"
        size="small"
      >
        <el-form-item label="服务商" prop="provider">
          <el-input v-model="dialog.form.provider" placeholder="例如 openai-compatible / dashscope / openai" />
        </el-form-item>
        <el-form-item label="模型名称" prop="modelName">
          <el-input
            v-model="dialog.form.modelName"
            placeholder="API 调用的 model 字段，例如 qwen-plus / gpt-4o-mini"
          />
        </el-form-item>
        <el-form-item label="备注名称">
          <el-input
            v-model="dialog.form.displayName"
            placeholder="界面展示用的别名（留空时与模型名一致）"
          />
        </el-form-item>
        <el-form-item label="自定义版本">
          <el-input
            v-model="dialog.form.version"
            placeholder="可空。给同一模型测不同版本时区分，例如 2025-04-pre"
          />
        </el-form-item>
        <el-form-item label="模型接口" prop="baseUrl">
          <el-input v-model="dialog.form.baseUrl" placeholder="OpenAI 兼容 baseUrl，例如 https://api.example.com/v1" />
        </el-form-item>
        <el-form-item label="API Key" :prop="dialog.id ? '' : 'apiKey'">
          <el-input
            v-model="dialog.form.apiKey"
            type="password"
            show-password
            :placeholder="dialog.id ? '留空表示不修改；填写则替换旧值' : '请输入完整 API Key（如 sk-xxx...，至少 16 位）'"
          />
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
  listModels,
  createModel,
  updateModel,
  deleteModel
} from "@/api/model";
import { formatDateTime } from "@/utils/time";

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
  },
  rules: {
    provider: [{ required: true, message: "请输入服务商", trigger: "blur" }],
    modelName: [{ required: true, message: "请输入模型名称", trigger: "blur" }],
    baseUrl: [{ required: true, message: "请输入 Base URL", trigger: "blur" }],
    apiKey: [
      { required: true, message: "请输入 API Key", trigger: "blur" },
      { min: 16, message: "API Key 长度过短（< 16），请确认是否填错", trigger: "blur" }
    ]
  }
});

export default {
  name: "ModelListView",
  components: { PageHeader },
  data() {
    return {
      loading: false,
      rows: [],
      dialog: buildDialog()
    };
  },
  computed: {
    emptyText() {
      return "暂无模型预设，点击右上角「新增模型」开始";
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
        this.$message.error(error?.response?.data?.message || "加载模型预设失败");
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
        this.$message.success("保存成功");
        this.dialog.visible = false;
        this.loadList();
      } catch (error) {
        this.$message.error(
          error?.response?.data?.message || "保存失败"
        );
      } finally {
        this.dialog.saving = false;
      }
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确认删除模型「${row.displayName || row.modelName}」？`, "提示", {
          type: "warning"
        });
      } catch (e) {
        return;
      }
      try {
        await deleteModel(row.id);
        this.$message.success("已删除");
        this.loadList();
      } catch (error) {
        this.$message.error(
          error?.response?.data?.message || "删除失败"
        );
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
