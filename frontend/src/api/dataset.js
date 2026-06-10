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

// HuggingFace
export function searchHuggingFace(keyword, { sort, tag, limit } = {}) {
  return http.get("/datasets/search-huggingface", {
    params: { keyword, sort, tag, limit: limit || 20 },
    silent: true
  });
}

export function pullHuggingFaceDataset(repo, subset, split) {
  return http.post("/datasets/pull-huggingface", { repo, subset, split }, { silent: true });
}

export function getHuggingFaceDetail(repo) {
  return http.get("/datasets/huggingface-detail", { params: { repo }, silent: true });
}

// Custom dataset
export function uploadDatasetFile(file, meta = {}) {
  const formData = new FormData();
  formData.append("file", file);
  if (meta.displayName) formData.append("displayName", meta.displayName);
  if (meta.description) formData.append("description", meta.description);
  if (meta.taskType) formData.append("taskType", meta.taskType);
  return http.post("/datasets/upload", formData, {
    silent: true,
    headers: { "Content-Type": "multipart/form-data" }
  });
}

export function createCustomFromPath(payload) {
  return http.post("/datasets/custom-from-path", payload, { silent: true });
}

// Demo datasets
export function getDemoDatasets() {
  return http.get("/datasets/demo", { silent: true });
}

// Preview
export function previewDataset(id, limit = 10) {
  return http.get(`/datasets/${id}/preview`, { params: { limit }, silent: true });
}

export function previewDatasetByPath(path, limit = 10) {
  return http.get("/datasets/preview-by-path", { params: { path, limit }, silent: true });
}
