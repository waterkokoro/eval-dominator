import Vue from "vue";

export const appStore = Vue.observable({
  collapsed: false,
  health: {
    backend: { ok: null, message: "" },
    core: { ok: null, message: "" },
    checkedAt: null
  }
});

export function toggleSidebar() {
  appStore.collapsed = !appStore.collapsed;
}

export function setSidebarCollapsed(collapsed) {
  appStore.collapsed = collapsed;
}

export function setHealth(partial) {
  appStore.health = {
    ...appStore.health,
    ...partial,
    checkedAt: new Date().toISOString()
  };
}
