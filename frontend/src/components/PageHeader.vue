<template>
  <div class="page-header">
    <div class="page-header-main">
      <h2 class="page-header-title">{{ title }}</h2>
      <p v-if="description" class="page-header-desc">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script>
export default {
  name: "PageHeader",
  props: {
    title: { type: String, required: true },
    description: { type: String, default: "" }
  }
};
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
  gap: 16px;
}

.page-header-main {
  flex: 1;
}

.page-header-title {
  margin: 0;
  font-size: 18px;
  color: #303133;
  font-weight: 600;
}

.page-header-desc {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}

.page-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
