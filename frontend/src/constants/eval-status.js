import i18n from "@/locales";

/**
 * 任务状态枚举与本地化无关的属性（颜色、进度、tag 类型）。
 * 文案统一通过 i18n key `eval.status.<status>` 取，避免硬编码任何具体语言。
 */

export const EVAL_STATUS_KEYS = [
  "pending",
  "validating",
  "building",
  "running",
  "parsing",
  "succeeded",
  "failed",
  "timeout",
  "cancelled"
];

export const evalStatusType = {
  pending: "info",
  validating: "warning",
  building: "warning",
  running: "warning",
  parsing: "warning",
  succeeded: "success",
  failed: "danger",
  timeout: "danger",
  cancelled: "info"
};

export const evalStatusProgress = {
  pending: 5,
  validating: 15,
  building: 25,
  running: 60,
  parsing: 85,
  succeeded: 100,
  failed: 100,
  timeout: 100,
  cancelled: 100
};

/**
 * 状态筛选下拉的选项。返回的是函数而非常量，因为 label 依赖当前语言；
 * 直接在模板的 computed 里调用即可让选项跟随切语言更新。
 */
export function getEvalStatusOptions() {
  return EVAL_STATUS_KEYS.map((value) => ({
    value,
    label: i18n.t(`eval.status.${value}`)
  }));
}

export function getEvalStatusText(status) {
  if (!status) return i18n.t("eval.status.unknown");
  if (EVAL_STATUS_KEYS.includes(status)) {
    return i18n.t(`eval.status.${status}`);
  }
  return status;
}

export function getEvalStatusType(status) {
  return evalStatusType[status] || "info";
}

export function isEvalStatusFinal(status) {
  return (
    status === "succeeded" ||
    status === "failed" ||
    status === "timeout" ||
    status === "cancelled"
  );
}

export function canCancelEvalStatus(status) {
  return status && !isEvalStatusFinal(status);
}

export function getEvalStatusProgress(status) {
  if (status in evalStatusProgress) return evalStatusProgress[status];
  return 0;
}

export function getEvalProgressColor(status) {
  if (status === "succeeded") return "#67c23a";
  if (status === "failed" || status === "timeout") return "#f56c6c";
  if (status === "cancelled") return "#909399";
  return "#409eff";
}

export function getEvalProgressStatusType(status) {
  if (status === "succeeded") return "success";
  if (status === "failed" || status === "timeout") return "exception";
  return undefined;
}
