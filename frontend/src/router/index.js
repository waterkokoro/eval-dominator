import Vue from "vue";
import VueRouter from "vue-router";

import LoginView from "@/views/LoginView.vue";
import AppLayout from "@/layout/AppLayout.vue";

import { getToken } from "@/utils/token";

Vue.use(VueRouter);

const routes = [
  {
    path: "/login",
    name: "login",
    component: LoginView,
    meta: { public: true, title: "登录" }
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
        meta: { title: "任务列表", group: "评测中心" }
      },
      {
        path: "eval/submit",
        name: "eval-submit",
        component: () => import("@/views/eval/EvalSubmitView.vue"),
        meta: { title: "提交评测", group: "评测中心" }
      },
      {
        path: "eval/tasks/:evalTaskId",
        name: "eval-task-detail",
        component: () => import("@/views/eval/EvalTaskDetailView.vue"),
        meta: {
          title: "任务详情",
          group: "评测中心",
          activeMenu: "/eval/tasks"
        }
      },
      {
        path: "models",
        name: "model-list",
        component: () => import("@/views/model/ModelListView.vue"),
        meta: { title: "模型管理", group: "模型管理" }
      },
      {
        path: "datasets",
        name: "dataset-list",
        component: () => import("@/views/dataset/DatasetListView.vue"),
        meta: { title: "数据集中心", group: "数据集" }
      },
      {
        path: "about",
        name: "about",
        component: () => import("@/views/system/AboutView.vue"),
        meta: { title: "关于", group: "系统" }
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

router.afterEach((to) => {
  const base = "Eval Dominator";
  document.title = to.meta?.title ? `${to.meta.title} · ${base}` : base;
});

export default router;
