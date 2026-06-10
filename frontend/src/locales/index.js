import Vue from "vue";
import VueI18n from "vue-i18n";
import ElementLocale from "element-ui/lib/locale";
import elementZhCN from "element-ui/lib/locale/lang/zh-CN";
import elementEnUS from "element-ui/lib/locale/lang/en";

import zhCN from "./zh-CN";
import enUS from "./en-US";

Vue.use(VueI18n);

const STORAGE_KEY = "evalDominatorLang";
const DEFAULT_LANG = "zh-CN";

/**
 * 语言注册表。fork 想接新语言时只在这里加一项即可，业务代码无需改动。
 * - code: BCP 47 风格标识，与 vue-i18n locale 一致
 * - label: 当前语言下的"自显示"名称
 * - app:   业务文案字典（来自 ./{lang}/index.js）
 * - el:    ElementUI 内置 locale 包（标题/分页/确定取消等）
 */
export const LANGUAGES = [
  { code: "zh-CN", label: "中文", app: zhCN, el: elementZhCN },
  { code: "en-US", label: "English", app: enUS, el: elementEnUS }
];

/** 深度合并：将 src 中的叶子节点递归合入 target（不修改原始对象） */
function deepMerge(target, src) {
  const out = { ...target };
  for (const key of Object.keys(src)) {
    if (
      src[key] &&
      typeof src[key] === "object" &&
      !Array.isArray(src[key]) &&
      out[key] &&
      typeof out[key] === "object" &&
      !Array.isArray(out[key])
    ) {
      out[key] = deepMerge(out[key], src[key]);
    } else {
      out[key] = src[key];
    }
  }
  return out;
}

const messages = LANGUAGES.reduce((acc, item) => {
  acc[item.code] = deepMerge(item.app, item.el || {});
  return acc;
}, {});

function detectInitialLang() {
  if (typeof window === "undefined") return DEFAULT_LANG;
  const saved = window.localStorage?.getItem(STORAGE_KEY);
  if (saved && LANGUAGES.some((l) => l.code === saved)) return saved;
  // 首次进入：仅当浏览器明确是英文时才用英文，其他全部走中文（中文优先）
  const nav = (navigator?.language || "").toLowerCase();
  if (nav.startsWith("en")) return "en-US";
  return DEFAULT_LANG;
}

const initialLang = detectInitialLang();

const i18n = new VueI18n({
  locale: initialLang,
  fallbackLocale: DEFAULT_LANG,
  silentFallbackWarn: true,
  messages
});

ElementLocale.i18n((key, value) => i18n.t(key, value));
applyElementLocale(initialLang);

function applyElementLocale(code) {
  const entry = LANGUAGES.find((l) => l.code === code);
  if (entry?.el) ElementLocale.use(entry.el);
}

export function setLanguage(code) {
  const entry = LANGUAGES.find((l) => l.code === code);
  if (!entry) return;
  i18n.locale = code;
  applyElementLocale(code);
  if (typeof window !== "undefined") {
    window.localStorage?.setItem(STORAGE_KEY, code);
    document.documentElement.lang = code;
  }
}

export function getCurrentLanguage() {
  return i18n.locale;
}

if (typeof document !== "undefined") {
  document.documentElement.lang = initialLang;
}

export { i18n };
export default i18n;
