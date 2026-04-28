<template>
  <el-tag :type="tagType" size="small" effect="light">
    {{ tagText }}
  </el-tag>
</template>

<script>
import {
  getEvalStatusText,
  getEvalStatusType
} from "@/constants/eval-status";

export default {
  name: "StatusTag",
  props: {
    status: {
      type: String,
      default: ""
    }
  },
  computed: {
    tagType() {
      return getEvalStatusType(this.status);
    },
    tagText() {
      return getEvalStatusText(this.status);
    }
  }
};
</script>
