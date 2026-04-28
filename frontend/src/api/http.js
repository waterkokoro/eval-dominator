import axios from "axios";
import { Message } from "element-ui";

import { getToken, clearToken } from "@/utils/token";

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

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const config = error.config || {};
    const status = error.response?.status;
    const message =
      error.response?.data?.message || error.message || "请求失败";

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
