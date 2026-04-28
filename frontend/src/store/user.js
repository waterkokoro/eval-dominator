import Vue from "vue";

import { getToken, setToken, clearToken } from "@/utils/token";

export const userStore = Vue.observable({
  token: getToken() || "",
  user: null
});

export function loginSuccess(token) {
  setToken(token);
  userStore.token = token;
}

export function setUser(user) {
  userStore.user = user;
}

export function logoutLocal() {
  clearToken();
  userStore.token = "";
  userStore.user = null;
}
