/**
 * 后端目前下发的时间格式为 'YYYY-MM-DD HH:MM:SS'（本地时区），
 * 同时兼容历史数据/缓存里出现的 ISO 8601 格式。
 */

const ISO_RE = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2}):(\d{2})/;

/** 解析后端时间字符串到本地 Date（解析失败返回 null）。 */
export function parseBackendDate(input) {
  if (!input) return null;
  if (input instanceof Date) return Number.isNaN(input.getTime()) ? null : input;
  const s = String(input).trim();
  if (!s) return null;

  // 'YYYY-MM-DD HH:MM:SS' 在 Safari 下 new Date 不一定可解析，统一手动解析为本地时间。
  const m = ISO_RE.exec(s);
  if (m) {
    const [, y, mo, d, h, mi, se] = m;
    const dt = new Date(Number(y), Number(mo) - 1, Number(d), Number(h), Number(mi), Number(se));
    return Number.isNaN(dt.getTime()) ? null : dt;
  }
  const t = Date.parse(s);
  return Number.isNaN(t) ? null : new Date(t);
}

/** 'YYYY-MM-DD HH:MM:SS' 的展示形式；空值返回 fallback。 */
export function formatDateTime(input, fallback = "—") {
  const d = parseBackendDate(input);
  if (!d) return fallback;
  const y = d.getFullYear();
  const mo = String(d.getMonth() + 1).padStart(2, "0");
  const da = String(d.getDate()).padStart(2, "0");
  const h = String(d.getHours()).padStart(2, "0");
  const mi = String(d.getMinutes()).padStart(2, "0");
  const se = String(d.getSeconds()).padStart(2, "0");
  return `${y}-${mo}-${da} ${h}:${mi}:${se}`;
}

/** 计算两点时间的耗时字符串（s / m s / h m）。 */
export function durationText(startInput, endInput) {
  const start = parseBackendDate(startInput);
  const end = parseBackendDate(endInput);
  if (!start || !end) return "-";
  const sec = Math.max(0, Math.round((end.getTime() - start.getTime()) / 1000));
  if (sec < 60) return `${sec}s`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ${sec % 60}s`;
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return `${h}h ${m}m`;
}
