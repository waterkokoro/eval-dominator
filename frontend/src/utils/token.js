const tokenKey = process.env.VUE_APP_TOKEN_STORAGE_KEY || "eval_dominator_token";

export function getToken() {
  return window.localStorage.getItem(tokenKey);
}

export function setToken(token) {
  window.localStorage.setItem(tokenKey, token);
}

export function clearToken() {
  window.localStorage.removeItem(tokenKey);
}
