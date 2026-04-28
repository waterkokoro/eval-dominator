import http from "./http";

export function listModels() {
  return http.get("/models", { silent: true });
}

export function createModel(payload) {
  return http.post("/models", payload, { silent: true });
}

export function updateModel(id, payload) {
  return http.put(`/models/${id}`, payload, { silent: true });
}

export function deleteModel(id) {
  return http.delete(`/models/${id}`, { silent: true });
}
