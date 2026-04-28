import http from "./http";

export function previewArtifact(evalTaskId, path) {
  return http.get(`/eval/tasks/${evalTaskId}/artifacts/preview`, {
    params: { path },
    silent: true
  });
}

/**
 * 强制按浏览器附件下载（带后端的 Content-Disposition）。
 * 这里 axios 拿 blob，再触发本地保存。
 */
export async function downloadArtifact(evalTaskId, path, filename) {
  const blob = await http.get(`/eval/tasks/${evalTaskId}/artifacts/download`, {
    params: { path },
    responseType: "blob",
    silent: true
  });
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename || (path.split(/[\\/]/).pop() || "artifact");
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(url);
}
