import Vue from "vue";
import VueRouter from "vue-router";

import LoginView from "@/views/LoginView.vue";
import AppLayout from "@/layout/AppLayout.vue";

import { getToken } from "@/utils/token";
import i18n from "@/locales";

Vue.use(VueRouter);

const routes = [
  {
    path: "/login",
    name: "login",
    component: LoginView,
    meta: { public: true, titleKey: "auth.login.title" }
  },
  {
    path: "/",
    component: AppLayout,
    redirect: "/eval/tasks",
    children: [
      {
        path: "eval/tasks",
        name: "eval-task-list",
        component: () => import("@/views/eval/EvalTaskListView.vue"),
        meta: { titleKey: "eval.list.title", groupKey: "eval.group" }
      },
      {
        path: "eval/submit",
        name: "eval-submit",
        component: () => import("@/views/eval/EvalSubmitView.vue"),
        meta: { titleKey: "eval.submit.title", groupKey: "eval.group" }
      },
      {
        path: "eval/tasks/:evalTaskId",
        name: "eval-task-detail",
        component: () => import("@/views/eval/EvalTaskDetailView.vue"),
        meta: {
          titleKey: "eval.detail.title",
          groupKey: "eval.group",
          activeMenu: "/eval/tasks"
        }
      },
      {
        path: "models",
        name: "model-list",
        component: () => import("@/views/model/ModelListView.vue"),
        meta: { titleKey: "model.list.title", groupKey: "model.group" }
      },
      {
        path: "datasets",
        name: "dataset-list",
        component: () => import("@/views/dataset/DatasetListView.vue"),
        meta: { titleKey: "dataset.list.title", groupKey: "dataset.group" }
      },
      {
        path: "about",
        name: "about",
        component: () => import("@/views/system/AboutView.vue"),
        meta: { titleKey: "system.about.title", groupKey: "system.group" }
      }
    ]
  },
  {
    path: "*",
    redirect: "/"
  }
];

const router = new VueRouter({ routes });

router.beforeEach((to, from, next) => {
  if (to.meta.public || getToken()) {
    if (to.name === "login" && getToken()) {
      next({ path: to.query?.redirect || "/eval/tasks" });
      return;
    }
    next();
    return;
  }
  next({ name: "login", query: { redirect: to.fullPath } });
});

function applyDocumentTitle(route) {
  const base = "Eval Dominator";
  const key = route?.meta?.titleKey;
  const title = key ? i18n.t(key) : "";
  document.title = title ? `${title} · ${base}` : base;
}

router.afterEach(applyDocumentTitle);

// 切语言后 router.afterEach 不会再触发，需要手动重算一次。
if (typeof window !== "undefined") {
  // vue-i18n 会在 locale 变化时触发 reactive 更新；用 watcher 监听 i18n.locale。
  // 这里通过 Vue.observable 模式访问 i18n 内部 reactive 状态。
  const Watcher = Vue.extend({
    computed: {
      locale() {
        return i18n.locale;
      }
    },
    watch: {
      locale() {
        applyDocumentTitle(router.currentRoute);
      }
    }
  });
  new Watcher();
}

export default router;
