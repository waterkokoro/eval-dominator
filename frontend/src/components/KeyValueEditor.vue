<template>
  <div class="kv-editor">
    <div v-for="(row, index) in rows" :key="index" class="kv-row">
      <el-input
        v-model="row.key"
        :placeholder="placeholderKey"
        size="small"
        class="kv-input"
        @input="emitChange"
      />
      <el-input
        v-model="row.value"
        :placeholder="placeholderValue"
        size="small"
        class="kv-input"
        @input="emitChange"
      />
      <el-button
        type="text"
        icon="el-icon-delete"
        class="kv-remove"
        @click="removeRow(index)"
      />
    </div>
    <el-button
      icon="el-icon-plus"
      size="mini"
      type="text"
      @click="addRow"
    >
      {{ $t("common.actions.addItem") }}
    </el-button>
  </div>
</template>

<script>
const fromObject = (obj) =>
  Object.entries(obj || {}).map(([key, value]) => ({ key, value }));

export default {
  name: "KeyValueEditor",
  model: { prop: "value", event: "change" },
  props: {
    value: { type: Object, default: () => ({}) },
    placeholderKey: { type: String, default: "key" },
    placeholderValue: { type: String, default: "value" }
  },
  data() {
    return {
      rows: fromObject(this.value)
    };
  },
  watch: {
    value: {
      handler(newVal) {
        const objAsRows = fromObject(newVal);
        const same =
          objAsRows.length === this.rows.length &&
          objAsRows.every(
            (item, i) =>
              item.key === this.rows[i]?.key &&
              item.value === this.rows[i]?.value
          );
        if (!same) this.rows = objAsRows;
      },
      deep: true
    }
  },
  methods: {
    addRow() {
      this.rows.push({ key: "", value: "" });
    },
    removeRow(index) {
      this.rows.splice(index, 1);
      this.emitChange();
    },
    emitChange() {
      const result = {};
      this.rows.forEach((row) => {
        if (row.key) result[row.key] = row.value;
      });
      this.$emit("change", result);
    }
  }
};
</script>

<style scoped>
.kv-editor {
  width: 100%;
}
.kv-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.kv-input {
  flex: 1;
}
.kv-remove {
  color: #f56c6c;
}
</style>
