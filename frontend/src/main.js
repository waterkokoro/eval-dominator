import Vue from "vue";
import ElementUI from "element-ui";
import "element-ui/lib/theme-chalk/index.css";

import App from "./App.vue";
import router from "./router";
import i18n from "./locales";
import { logoutLocal } from "@/store/user";

Vue.use(ElementUI);
Vue.config.productionTip = false;

window.addEventListener("eval-dominator:unauthorized", () => {
  logoutLocal();
  if (router.currentRoute?.name !== "login") {
    router.replace({
      name: "login",
      query: { redirect: router.currentRoute?.fullPath }
    });
  }
});

new Vue({
  router,
  i18n,
  render: (h) => h(App)
}).$mount("#app");
