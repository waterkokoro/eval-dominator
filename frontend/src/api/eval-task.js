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

export function getEvalTaskLog(evalTaskId, tail = 200) {
  return http.get(`/eval/tasks/${evalTaskId}/log`, {
    params: { tail },
    silent: true
  });
}

export function cancelEvalTask(evalTaskId) {
  return http.post(`/eval/tasks/${evalTaskId}/cancel`);
}
