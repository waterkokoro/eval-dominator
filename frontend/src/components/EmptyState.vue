<template>
  <div class="empty-state" :class="`empty-state--${type}`">
    <i class="empty-state-icon" :class="iconClass" />
    <div class="empty-state-title">{{ resolvedTitle }}</div>
    <div v-if="description" class="empty-state-desc">{{ description }}</div>
    <div v-if="$slots.default" class="empty-state-actions">
      <slot />
    </div>
  </div>
</template>

<script>
const iconMap = {
  empty: "el-icon-folder-opened",
  loading: "el-icon-loading",
  error: "el-icon-warning-outline",
  todo: "el-icon-time"
};

export default {
  name: "EmptyState",
  props: {
    type: {
      type: String,
      default: "empty",
      validator: (value) => ["empty", "loading", "error", "todo"].includes(value)
    },
    title: {
      type: String,
      default: ""
    },
    description: {
      type: String,
      default: ""
    }
  },
  computed: {
    iconClass() {
      return iconMap[this.type] || iconMap.empty;
    },
    resolvedTitle() {
      return this.title || this.$t("common.messages.empty");
    }
  }
};
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  color: #909399;
  text-align: center;
}

.empty-state-icon {
  font-size: 48px;
  margin-bottom: 12px;
  color: #c0c4cc;
}

.empty-state--error .empty-state-icon {
  color: #f56c6c;
}

.empty-state--todo .empty-state-icon {
  color: #e6a23c;
}

.empty-state-title {
  font-size: 14px;
  color: #606266;
  margin-bottom: 4px;
}

.empty-state-desc {
  font-size: 13px;
  color: #909399;
  max-width: 360px;
  line-height: 1.6;
}

.empty-state-actions {
  margin-top: 16px;
}
</style>
