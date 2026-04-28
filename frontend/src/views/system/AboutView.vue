<template>
  <div class="about-view">
    <PageHeader
      title="关于"
      description="项目信息、服务健康状态与使用提示"
    >
      <template #actions>
        <el-button icon="el-icon-refresh" :loading="loading" @click="loadHealth">
          刷新状态
        </el-button>
      </template>
    </PageHeader>

    <div class="health-grid">
      <el-card shadow="never">
        <div class="health-card">
          <div class="health-icon" :class="`is-${level(backend.ok)}`">
            <i class="el-icon-monitor" />
          </div>
          <div class="health-main">
            <div class="health-title">Go Web 后端</div>
            <div class="health-meta">{{ backend.message || "等待检查" }}</div>
          </div>
          <el-tag :type="tagType(backend.ok)" size="small">
            {{ statusText(backend.ok) }}
          </el-tag>
        </div>
      </el-card>

      <el-card shadow="never">
        <div class="health-card">
          <div class="health-icon" :class="`is-${level(core.ok)}`">
            <i class="el-icon-cpu" />
          </div>
          <div class="health-main">
            <div class="health-title">Python Core</div>
            <div class="health-meta">{{ core.message || "等待检查" }}</div>
          </div>
          <el-tag :type="tagType(core.ok)" size="small">
            {{ statusText(core.ok) }}
          </el-tag>
        </div>
      </el-card>
    </div>

    <el-card shadow="never">
      <div class="info-title">项目信息</div>
      <el-descriptions :column="1" size="small" border>
        <el-descriptions-item label="项目名称">Eval Dominator</el-descriptions-item>
        <el-descriptions-item label="定位">轻量级模型评测中心，封装 OpenCompass 原子能力</el-descriptions-item>
        <el-descriptions-item label="技术栈">
          Vue2 + ElementUI · Go + Gin · Python Core (gRPC) · SQLite
        </el-descriptions-item>
        <el-descriptions-item label="健康检查时间">{{ checkedAt }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="todo-card">
      <div class="info-title">后端待补接口</div>
      <ul class="todo-list">
        <li v-for="api in pendingApis" :key="api"><code>{{ api }}</code></li>
      </ul>
    </el-card>
  </div>
</template>

<script>
import PageHeader from "@/components/PageHeader.vue";

import { getSystemHealth } from "@/api/system";
import { appStore, setHealth } from "@/store/app";

const PENDING_APIS = [
  "POST /auth/logout",
  "GET /auth/me",
  "GET /eval/tasks",
  "GET /eval/tasks/:id/log",
  "GET /eval/tasks/:id/artifacts/:type",
  "GET / POST / PUT / DELETE /models",
  "GET /datasets",
  "GET /system/health"
];

export default {
  name: "AboutView",
  components: { PageHeader },
  data() {
    return {
      loading: false,
      pendingApis: PENDING_APIS
    };
  },
  computed: {
    backend() {
      return appStore.health.backend || { ok: null, message: "" };
    },
    core() {
      return appStore.health.core || { ok: null, message: "" };
    },
    checkedAt() {
      return appStore.health.checkedAt || "未检查";
    }
  },
  created() {
    this.loadHealth();
  },
  methods: {
    async loadHealth() {
      this.loading = true;
      try {
        const data = await getSystemHealth();
        setHealth({
          backend: data?.backend || { ok: true, message: "ok" },
          core: data?.core || { ok: true, message: "ok" }
        });
      } catch (error) {
        const status = error?.response?.status;
        if (!status || status === 404) {
          setHealth({
            backend: { ok: null, message: "/system/health 接口待后端补齐" },
            core: { ok: null, message: "/system/health 接口待后端补齐" }
          });
        } else {
          setHealth({
            backend: { ok: false, message: "unreachable" },
            core: { ok: false, message: "unreachable" }
          });
        }
      } finally {
        this.loading = false;
      }
    },
    level(ok) {
      if (ok === null || ok === undefined) return "unknown";
      return ok ? "ok" : "down";
    },
    tagType(ok) {
      if (ok === null || ok === undefined) return "info";
      return ok ? "success" : "danger";
    },
    statusText(ok) {
      if (ok === null || ok === undefined) return "未知";
      return ok ? "正常" : "异常";
    }
  }
};
</script>

<style scoped>
.about-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.health-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.health-card {
  display: flex;
  align-items: center;
  gap: 12px;
}

.health-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  background: #f4f4f5;
  color: #909399;
}

.health-icon.is-ok {
  background: #f0f9eb;
  color: #67c23a;
}
.health-icon.is-down {
  background: #fef0f0;
  color: #f56c6c;
}
.health-icon.is-unknown {
  background: #f4f4f5;
  color: #909399;
}

.health-main {
  flex: 1;
}

.health-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.health-meta {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.info-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
}

.todo-list {
  margin: 0;
  padding-left: 20px;
  color: #606266;
  font-size: 13px;
  line-height: 1.8;
}

.todo-list code {
  background: #f4f4f5;
  border-radius: 4px;
  padding: 1px 6px;
  font-family: "Menlo", "Monaco", monospace;
}
</style>
