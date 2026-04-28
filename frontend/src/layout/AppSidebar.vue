<template>
  <div class="app-sidebar">
    <div class="brand" @click="goHome">
      <span class="brand-mark">ED</span>
      <span v-if="!collapsed" class="brand-name">Eval Dominator</span>
    </div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      :collapse-transition="false"
      :unique-opened="false"
      router
      background-color="#1f2329"
      text-color="#c0c4cc"
      active-text-color="#409EFF"
    >
      <template v-for="group in menuGroups">
        <el-submenu :index="group.key" :key="group.key">
          <template #title>
            <i :class="group.icon" />
            <span>{{ group.title }}</span>
          </template>
          <el-menu-item
            v-for="item in group.items"
            :key="item.path"
            :index="item.path"
          >
            <i :class="item.icon" />
            <span slot="title">{{ item.title }}</span>
          </el-menu-item>
        </el-submenu>
      </template>
    </el-menu>
  </div>
</template>

<script>
import { appStore } from "@/store/app";

export default {
  name: "AppSidebar",
  data() {
    return {
      menuGroups: [
        {
          key: "eval",
          title: "评测中心",
          icon: "el-icon-data-analysis",
          items: [
            {
              path: "/eval/tasks",
              title: "任务列表",
              icon: "el-icon-tickets"
            },
            {
              path: "/eval/submit",
              title: "提交评测",
              icon: "el-icon-edit-outline"
            }
          ]
        },
        {
          key: "model",
          title: "模型管理",
          icon: "el-icon-cpu",
          items: [
            {
              path: "/models",
              title: "模型管理",
              icon: "el-icon-key"
            }
          ]
        },
        {
          key: "dataset",
          title: "数据集",
          icon: "el-icon-collection",
          items: [
            {
              path: "/datasets",
              title: "数据集中心",
              icon: "el-icon-folder-opened"
            }
          ]
        },
        {
          key: "system",
          title: "系统",
          icon: "el-icon-s-tools",
          items: [
            {
              path: "/about",
              title: "关于",
              icon: "el-icon-info"
            }
          ]
        }
      ]
    };
  },
  computed: {
    collapsed() {
      return appStore.collapsed;
    },
    activeMenu() {
      const matched = this.$route.matched || [];
      const routeMeta = this.$route.meta || {};
      if (routeMeta.activeMenu) return routeMeta.activeMenu;
      const last = matched[matched.length - 1];
      return last ? last.path : this.$route.path;
    }
  },
  methods: {
    goHome() {
      if (this.$route.path !== "/eval/tasks") {
        this.$router.push("/eval/tasks");
      }
    }
  }
};
</script>

<style scoped>
.app-sidebar {
  height: 100vh;
  background: #1f2329;
  color: #fff;
  display: flex;
  flex-direction: column;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 16px;
  border-bottom: 1px solid #2c2f36;
  cursor: pointer;
  user-select: none;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: linear-gradient(135deg, #409eff, #336cff);
  color: #fff;
  font-weight: 700;
  font-size: 13px;
  flex-shrink: 0;
}

.brand-name {
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
}

.app-sidebar >>> .el-menu {
  border-right: 0;
  flex: 1;
}

.app-sidebar >>> .el-submenu__title {
  height: 44px;
  line-height: 44px;
}

.app-sidebar >>> .el-menu-item {
  height: 40px;
  line-height: 40px;
}

.app-sidebar >>> .el-menu-item.is-active {
  background-color: #2a2f37 !important;
}

.app-sidebar >>> .el-menu-item:hover,
.app-sidebar >>> .el-submenu__title:hover {
  background-color: #2a2f37 !important;
}
</style>
