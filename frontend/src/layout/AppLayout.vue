<template>
  <el-container class="app-layout">
    <el-aside :width="asideWidth" class="app-aside">
      <AppSidebar />
    </el-aside>
    <el-container class="app-body">
      <el-header height="56px" class="app-body-header">
        <AppHeader />
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script>
import AppSidebar from "@/layout/AppSidebar.vue";
import AppHeader from "@/layout/AppHeader.vue";

import { appStore } from "@/store/app";
import { userStore, setUser } from "@/store/user";
import { fetchCurrentUser } from "@/api/auth";

export default {
  name: "AppLayout",
  components: { AppSidebar, AppHeader },
  computed: {
    asideWidth() {
      return appStore.collapsed ? "64px" : "208px";
    }
  },
  mounted() {
    if (userStore.token && !userStore.user) {
      this.loadCurrentUser();
    }
  },
  methods: {
    async loadCurrentUser() {
      try {
        const data = await fetchCurrentUser();
        if (data) setUser(data);
      } catch (e) {
        // 后端未就绪时不阻塞页面
      }
    }
  }
};
</script>

<style scoped>
.app-layout {
  height: 100vh;
}

.app-aside {
  height: 100vh;
  transition: width 0.2s;
  overflow: hidden;
}

.app-body {
  height: 100vh;
}

.app-body-header {
  padding: 0;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
}

.app-main {
  padding: 24px;
  background: #f5f7fa;
  overflow: auto;
}
</style>
