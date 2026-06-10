import http from "./http";

export function createEvalTask(payload) {
  return http.post("/eval/tasks", payload);
}

export function getEvalTask(evalTaskId) {
  return http.get(`/eval/tasks/${evalTaskId}`);
}

export function getEvalResult(evalTaskId) {
  return http.get(`/eval/tasks/${evalTaskId}/result`);
}

export function listEvalTasks(query = {}) {
  return http.get("/eval/tasks", { params: query, silent: true });
}

export function getEvalTaskLog(evalTaskId, tail = 200, logId = "") {
  const params = { tail };
  if (logId) params.logId = logId;
  return http.get(`/eval/tasks/${evalTaskId}/log`, {
    params,
    silent: true
  });
}

export function listEvalTaskLogs(evalTaskId) {
  return http.get(`/eval/tasks/${evalTaskId}/logs`, { silent: true });
}

export function cancelEvalTask(evalTaskId) {
  return http.post(`/eval/tasks/${evalTaskId}/cancel`);
}

// 仅重跑 evaluate 节点：复用旧的 predictions，不重新调用 LLM
export function rerunEvalNode(evalTaskId) {
  return http.post(`/eval/tasks/${evalTaskId}/rerun-eval`);
}

// 获取逐题分析数据：prompt / prediction / 关键词命中 / 得分 / 分类
export function getEvalAnalysis(evalTaskId) {
  return http.get(`/eval/tasks/${evalTaskId}/analysis`, { silent: true });
}
