<template>
  <div class="log-viewer">
    <div class="log-toolbar">
      <span class="log-title">
        <i class="el-icon-document toolbar-icon" />
        <span class="current-name">{{ currentDisplayName || $t('eval.log.empty') }}</span>
        <el-tag v-if="autoRefresh && !taskFinished" size="mini" effect="dark" type="warning">
          {{ $t("eval.log.autoRefresh") }}
        </el-tag>
        <span v-if="content" class="tail-hint">· {{ $t("eval.log.tail", { count: tail }) }}</span>
      </span>
      <div class="log-actions">
        <el-checkbox
          v-model="autoSelectLatest"
          class="toolbar-checkbox"
          size="small"
        >
          {{ $t("eval.log.autoSelectLatest") }}
        </el-checkbox>
        <el-checkbox
          v-model="autoRefresh"
          :disabled="taskFinished"
          class="toolbar-checkbox"
          size="small"
        >
          {{ $t("eval.log.autoRefresh") }}
        </el-checkbox>
        <el-button size="mini" :loading="loading" @click="reloadAll">
          {{ $t("eval.log.refresh") }}
        </el-button>
      </div>
    </div>
    <div class="log-main">
      <!-- 左侧菜单 -->
      <div class="log-sidebar">
        <div v-if="!logList.length" class="sidebar-empty">
          {{ $t("eval.log.noLogFiles") }}
        </div>
        <div v-else class="sidebar-groups">
          <div
            v-for="group in groupedLogs"
            :key="group.type"
            class="log-group"
          >
            <div class="group-header" @click="toggleGroup(group.type)">
              <i
                class="el-icon-arrow-right group-arrow"
                :class="{ open: !isCollapsed(group.type) }"
              />
              <i :class="group.icon" class="group-type-icon" />
              <span class="group-label">{{ group.label }}</span>
              <span class="group-count">{{ group.items.length }}</span>
            </div>
            <transition name="group-fade">
              <ul v-show="!isCollapsed(group.type)" class="group-list">
                <li
                  v-for="item in group.items"
                  :key="item.id"
                  class="log-item"
                  :class="{ active: item.id === selectedLogId }"
                  :title="item.displayName"
                  @click="handleSelect(item.id)"
                >
                  <span class="item-name">{{ item.displayName }}</span>
                  <span v-if="item.size != null" class="item-size">{{ formatSize(item.size) }}</span>
                </li>
              </ul>
            </transition>
          </div>
        </div>
      </div>

      <!-- 右侧内容 -->
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
  </div>
</template>

<script>
import EmptyState from "@/components/EmptyState.vue";
import { getEvalTaskLog, listEvalTaskLogs } from "@/api/eval-task";
import { isEvalStatusFinal } from "@/constants/eval-status";
import { resolveApiErrorMessage } from "@/api/http";

const GROUP_ORDER = ["main", "infer", "eval", "system"];
const GROUP_ICONS = {
  main: "el-icon-files",
  infer: "el-icon-cpu",
  eval: "el-icon-data-analysis",
  system: "el-icon-monitor",
};

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
      autoSelectLatest: true,
      timer: null,
      logList: [],
      selectedLogId: "",
      collapsedGroups: {}, // type -> bool
      userPickedAt: 0, // 用户最近一次点击时间，用于在 autoSelectLatest 时避免覆盖
    };
  },
  computed: {
    taskFinished() {
      return isEvalStatusFinal(this.taskStatus);
    },
    groupedLogs() {
      const groups = {};
      for (const item of this.logList) {
        const t = item.type || "system";
        if (!groups[t]) groups[t] = [];
        groups[t].push(item);
      }
      // 每组按 mtime 倒序
      Object.values(groups).forEach((arr) =>
        arr.sort((a, b) => (b.mtime || 0) - (a.mtime || 0))
      );
      const result = [];
      for (const t of GROUP_ORDER) {
        if (groups[t] && groups[t].length) {
          result.push({
            type: t,
            label: this.$t(`eval.log.groups.${t}`),
            icon: GROUP_ICONS[t] || "el-icon-document",
            items: groups[t],
          });
        }
      }
      // 兜底：未在 GROUP_ORDER 的类型
      Object.keys(groups).forEach((t) => {
        if (!GROUP_ORDER.includes(t)) {
          result.push({
            type: t,
            label: t,
            icon: "el-icon-document",
            items: groups[t],
          });
        }
      });
      return result;
    },
    currentDisplayName() {
      const item = this.logList.find((x) => x.id === this.selectedLogId);
      return item ? item.displayName : "";
    },
    latestLogId() {
      let best = null;
      for (const item of this.logList) {
        if (!best || (item.mtime || 0) > (best.mtime || 0)) best = item;
      }
      return best ? best.id : "";
    }
  },
  watch: {
    evalTaskId: {
      immediate: true,
      handler() {
        this.selectedLogId = "";
        this.content = "";
        this.reloadAll();
      }
    },
    taskStatus: {
      immediate: true,
      handler(newVal) {
        if (isEvalStatusFinal(newVal)) {
          this.stopTimer();
          if (this.evalTaskId) this.reloadAll();
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
    isCollapsed(type) {
      return !!this.collapsedGroups[type];
    },
    toggleGroup(type) {
      this.$set(this.collapsedGroups, type, !this.collapsedGroups[type]);
    },
    handleSelect(id) {
      if (id === this.selectedLogId) return;
      this.selectedLogId = id;
      this.userPickedAt = Date.now();
      this.fetchContent();
    },
    async reloadAll() {
      // 同时刷新日志列表和内容
      await this.fetchLogList();
      this.maybeAutoPick();
      await this.fetchContent();
    },
    async fetchLogList() {
      try {
        const data = await listEvalTaskLogs(this.evalTaskId);
        this.logList = (data && data.items) || [];
      } catch (error) {
        // 列表失败不阻塞内容
        this.logList = [];
      }
    },
    maybeAutoPick() {
      if (!this.logList.length) return;
      // 用户在最近 30 秒内手动选过则不强制覆盖
      const recentlyPicked = Date.now() - this.userPickedAt < 30 * 1000;
      const stillExists = this.logList.some((x) => x.id === this.selectedLogId);

      if (!stillExists) {
        // 当前选中的没了，必须重选
        this.selectedLogId = this.autoSelectLatest ? this.latestLogId : this.logList[0].id;
        return;
      }
      if (this.autoSelectLatest && !recentlyPicked) {
        this.selectedLogId = this.latestLogId;
      }
    },
    async fetchContent() {
      this.loading = true;
      this.errorText = "";
      try {
        const data = await getEvalTaskLog(this.evalTaskId, this.tail, this.selectedLogId);
        const content = (data && (data.content || data.log)) || "";
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
      this.timer = setInterval(() => this.reloadAll(), 5000);
    },
    stopTimer() {
      if (this.timer) {
        clearInterval(this.timer);
        this.timer = null;
      }
    },
    formatSize(bytes) {
      if (bytes == null) return "";
      if (bytes < 1024) return bytes + " B";
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
      return (bytes / 1024 / 1024).toFixed(1) + " MB";
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
  gap: 12px;
}

.log-title {
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.toolbar-icon {
  color: #67e8f9;
}

.current-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 320px;
  font-weight: 600;
  color: #f5f5f5;
}

.tail-hint {
  color: #909399;
  font-size: 12px;
}

.log-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.log-actions >>> .toolbar-checkbox .el-checkbox__label {
  color: #ddd;
  font-size: 12px;
}

/* ---- Main split ---- */
.log-main {
  display: flex;
  min-height: 320px;
  max-height: 520px;
}

/* ---- Sidebar ---- */
.log-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: #252526;
  color: #ccc;
  overflow-y: auto;
  border-right: 1px solid #333;
}
.sidebar-empty {
  padding: 12px;
  font-size: 12px;
  color: #888;
}
.sidebar-groups {
  padding: 4px 0;
}

.log-group + .log-group {
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  font-size: 12px;
  font-weight: 600;
  color: #d0d0d0;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}
.group-header:hover {
  background: rgba(255, 255, 255, 0.04);
}
.group-arrow {
  font-size: 12px;
  color: #888;
  transition: transform 0.2s;
}
.group-arrow.open {
  transform: rotate(90deg);
  color: #67e8f9;
}
.group-type-icon {
  font-size: 14px;
  color: #67e8f9;
}
.group-label {
  flex: 1;
}
.group-count {
  font-size: 11px;
  color: #888;
  background: rgba(255, 255, 255, 0.06);
  padding: 1px 6px;
  border-radius: 8px;
  font-variant-numeric: tabular-nums;
}

.group-list {
  list-style: none;
  margin: 0;
  padding: 2px 0 6px;
}
.log-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px 5px 28px;
  font-size: 12px;
  color: #bdbdbd;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  position: relative;
}
.log-item:hover {
  background: rgba(64, 158, 255, 0.08);
  color: #fff;
}
.log-item.active {
  background: linear-gradient(90deg, rgba(64, 158, 255, 0.22), rgba(167, 139, 250, 0.12));
  color: #fff;
}
.log-item.active::before {
  content: "";
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(180deg, #67e8f9, #409eff, #a78bfa);
}
.item-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.item-size {
  font-size: 11px;
  color: #888;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

/* ---- Body ---- */
.log-body {
  flex: 1;
  overflow: auto;
  padding: 12px;
  background: #1e1e1e;
  min-width: 0;
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

.group-fade-enter-active,
.group-fade-leave-active {
  transition: max-height 0.2s ease, opacity 0.2s ease;
  overflow: hidden;
}
.group-fade-enter,
.group-fade-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
