import axios from "axios";
import { Message } from "element-ui";

import { getToken, clearToken } from "@/utils/token";
import i18n from "@/locales";

const http = axios.create({
  baseURL: process.env.VUE_APP_API_BASE_URL || "http://127.0.0.1:8080/api",
  timeout: Number(process.env.VUE_APP_REQUEST_TIMEOUT || 30000)
});

http.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

/**
 * 把 axios error 解析为本地化提示。
 * 优先级：
 *   1. 后端 code 命中 errors.<CODE> -> 使用本地化文案
 *   2. 后端 message（开发期间一般是中文 fallback）
 *   3. 浏览器/网络层的 error.message
 *   4. common.messages.requestFailed
 */
export function resolveApiErrorMessage(error) {
  const code = error?.response?.data?.code;
  if (code) {
    const key = `errors.${code}`;
    if (i18n.te(key)) return i18n.t(key);
  }
  const fallback = error?.response?.data?.message || error?.message;
  if (fallback) return fallback;
  return i18n.t("common.messages.requestFailed");
}

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const config = error.config || {};
    const status = error.response?.status;
    const message = resolveApiErrorMessage(error);

    if (status === 401) {
      clearToken();
      window.dispatchEvent(new CustomEvent("eval-dominator:unauthorized"));
    }

    if (!config.silent) {
      Message.error(message);
    }
    return Promise.reject(error);
  }
);

export default http;
