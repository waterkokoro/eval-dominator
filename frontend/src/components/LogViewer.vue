<template>
  <div class="log-viewer">
    <div class="log-toolbar">
      <span class="log-title">
        {{ $t("eval.log.tail", { count: tail }) }}
        <el-tag v-if="autoRefresh" size="mini" effect="dark" type="warning">
          {{ $t("eval.log.autoRefresh") }}
        </el-tag>
      </span>
      <div class="log-actions">
        <el-checkbox
          v-model="autoRefresh"
          :disabled="taskFinished"
          class="auto-refresh-toggle"
          size="small"
        >
          {{ $t("eval.log.autoRefresh") }}
        </el-checkbox>
        <el-button size="mini" :loading="loading" @click="handleReload">
          {{ $t("eval.log.refresh") }}
        </el-button>
      </div>
    </div>
    <div class="log-body">
      <pre v-if="content" ref="logContent" class="log-content">{{ content }}</pre>
      <EmptyState
        v-else-if="errorText"
        type="error"
        :title="$t('eval.log.unavailable')"
        :description="errorText"
      />
      <EmptyState
        v-else
        type="todo"
        :title="$t('eval.log.empty')"
        :description="$t('eval.log.emptyDescription')"
      />
    </div>
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";
import { getEvalTaskLog } from "@/api/eval-task";
import { isEvalStatusFinal } from "@/constants/eval-status";
import { resolveApiErrorMessage } from "@/api/http";

export default {
  name: "LogViewer",
  components: { EmptyState },
  props: {
    evalTaskId: { type: String, required: true },
    tail: { type: Number, default: 200 },
    taskStatus: { type: String, default: "" }
  },
  data() {
    return {
      loading: false,
      content: "",
      errorText: "",
      autoRefresh: true,
      timer: null
    };
  },
  computed: {
    taskFinished() {
      return isEvalStatusFinal(this.taskStatus);
    }
  },
  watch: {
    evalTaskId: {
      immediate: true,
      handler() {
        this.handleReload();
      }
    },
    taskStatus: {
      immediate: true,
      handler(newVal) {
        if (isEvalStatusFinal(newVal)) {
          this.stopTimer();
          // 终态时再拉一次拿到最终内容
          if (this.evalTaskId) this.handleReload();
        } else if (this.autoRefresh) {
          this.startTimer();
        }
      }
    },
    autoRefresh(newVal) {
      if (newVal && !this.taskFinished) {
        this.startTimer();
      } else {
        this.stopTimer();
      }
    }
  },
  beforeDestroy() {
    this.stopTimer();
  },
  methods: {
    async handleReload() {
      this.loading = true;
      this.errorText = "";
      try {
        const data = await getEvalTaskLog(this.evalTaskId, this.tail);
        const content = data?.content || data?.log || "";
        this.content = content;
        if (!content) {
          this.errorText = this.$t("eval.log.emptyContent");
        }
        this.$nextTick(() => {
          if (this.$refs.logContent) {
            this.$refs.logContent.parentElement.scrollTop = this.$refs.logContent.parentElement.scrollHeight;
          }
        });
      } catch (error) {
        this.content = "";
        this.errorText = resolveApiErrorMessage(error) || this.$t("eval.log.loadFailed");
      } finally {
        this.loading = false;
      }
    },
    startTimer() {
      this.stopTimer();
      this.timer = setInterval(() => this.handleReload(), 5000);
    },
    stopTimer() {
      if (this.timer) {
        clearInterval(this.timer);
        this.timer = null;
      }
    }
  }
};
</script>

<style scoped>
.log-viewer {
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}

.log-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #2c2c2c;
  color: #ddd;
}

.log-title {
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.log-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.log-actions >>> .auto-refresh-toggle .el-checkbox__label {
  color: #ddd;
  font-size: 12px;
}

.log-body {
  min-height: 240px;
  max-height: 480px;
  overflow: auto;
  padding: 12px;
  background: #1e1e1e;
}

.log-content {
  margin: 0;
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #f5f5f5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
