import http from "./http";

export function getSystemHealth() {
  return http.get("/system/health", { silent: true });
}
