export const evalStatusText = {
  pending: "等待执行",
  validating: "校验配置",
  building: "构建配置",
  running: "执行评测",
  parsing: "解析结果",
  succeeded: "执行成功",
  failed: "执行失败",
  timeout: "执行超时",
  cancelled: "已终止"
};

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

export const evalStatusOptions = Object.keys(evalStatusText).map((value) => ({
  value,
  label: evalStatusText[value]
}));

export function getEvalStatusText(status) {
  return evalStatusText[status] || status || "未知状态";
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
