import http from "./http";

export function listDatasets(includeDisabled = false) {
  return http.get("/datasets", {
    params: includeDisabled ? { includeDisabled: 1 } : {},
    silent: true
  });
}

export function createDataset(payload) {
  return http.post("/datasets", payload, { silent: true });
}

export function updateDataset(id, payload) {
  return http.put(`/datasets/${id}`, payload, { silent: true });
}

export function setDatasetEnabled(id, enabled) {
  return http.patch(`/datasets/${id}/enabled`, { enabled }, { silent: true });
}

export function deleteDataset(id) {
  return http.delete(`/datasets/${id}`, { silent: true });
}

export function syncDatasets() {
  return http.post("/datasets/sync", {}, { silent: true });
}
