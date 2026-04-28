<template>
  <el-progress
    :percentage="percent"
    :stroke-width="strokeWidth"
    :color="color"
    :status="statusType"
    :show-text="showText"
    :text-inside="textInside"
  />
</template>

<script>
import {
  getEvalStatusProgress,
  getEvalProgressColor,
  getEvalProgressStatusType
} from "@/constants/eval-status";

export default {
  name: "EvalProgress",
  props: {
    status: { type: String, default: "" },
    strokeWidth: { type: Number, default: 6 },
    showText: { type: Boolean, default: true },
    textInside: { type: Boolean, default: false }
  },
  computed: {
    percent() {
      return getEvalStatusProgress(this.status);
    },
    color() {
      return getEvalProgressColor(this.status);
    },
    statusType() {
      return getEvalProgressStatusType(this.status);
    }
  }
};
</script>
