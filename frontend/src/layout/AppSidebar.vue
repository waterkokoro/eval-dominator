<template>
  <div class="app-sidebar">
    <div class="brand" @click="goHome">
      <span class="brand-mark">😈</span>
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
            <span>{{ $t(group.titleKey) }}</span>
          </template>
          <el-menu-item
            v-for="item in group.items"
            :key="item.path"
            :index="item.path"
          >
            <i :class="item.icon" />
            <span slot="title">{{ $t(item.titleKey) }}</span>
          </el-menu-item>
        </el-submenu>
      </template>
    </el-menu>

    <div class="sidebar-footer" :class="{ collapsed: collapsed }">
      <a
        class="footer-gh"
        href="https://github.com/waterkokoro/eval-dominator"
        target="_blank"
        rel="noopener noreferrer"
        :title="collapsed ? 'GitHub' : ''"
      >
        <svg class="gh-icon" viewBox="0 0 16 16" fill="currentColor" width="16" height="16">
          <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
        </svg>
        <template v-if="!collapsed">
          <span v-if="starCount !== null" class="gh-stars">
            <svg class="star-icon" viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
              <path d="M8 .25a.75.75 0 01.673.418l1.882 3.815 4.21.612a.75.75 0 01.416 1.279l-3.046 2.97.719 4.192a.75.75 0 01-1.088.791L8 12.347l-3.766 1.98a.75.75 0 01-1.088-.79l.72-4.194L.818 6.374a.75.75 0 01.416-1.28l4.21-.611L7.327.668A.75.75 0 018 .25z"/>
            </svg>
            {{ starCount }}
          </span>
          <span class="gh-author">{{ $t('common.footer.openSource') }}</span>
        </template>
      </a>
    </div>
  </div>
</template>

<script>
import { appStore } from "@/store/app";

const MENU_GROUPS = [
  {
    key: "eval",
    titleKey: "eval.group",
    icon: "el-icon-data-analysis",
    items: [
      {
        path: "/eval/tasks",
        titleKey: "eval.list.title",
        icon: "el-icon-tickets"
      },
      {
        path: "/eval/submit",
        titleKey: "eval.submit.title",
        icon: "el-icon-edit-outline"
      }
    ]
  },
  {
    key: "model",
    titleKey: "model.group",
    icon: "el-icon-cpu",
    items: [
      {
        path: "/models",
        titleKey: "model.list.title",
        icon: "el-icon-key"
      }
    ]
  },
  {
    key: "dataset",
    titleKey: "dataset.group",
    icon: "el-icon-collection",
    items: [
      {
        path: "/datasets",
        titleKey: "dataset.list.title",
        icon: "el-icon-folder-opened"
      }
    ]
  },
  {
    key: "system",
    titleKey: "system.group",
    icon: "el-icon-s-tools",
    items: [
      {
        path: "/about",
        titleKey: "system.about.title",
        icon: "el-icon-info"
      }
    ]
  }
];

export default {
  name: "AppSidebar",
  data() {
    return {
      menuGroups: MENU_GROUPS,
      starCount: null
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
  mounted() {
    this.fetchStarCount();
  },
  methods: {
    goHome() {
      if (this.$route.path !== "/eval/tasks") {
        this.$router.push("/eval/tasks");
      }
    },
    async fetchStarCount() {
      // 使用 Shields.io API（中国大陆可访问）
      try {
        const res = await fetch(
          "https://img.shields.io/github/stars/waterkokoro/eval-dominator.json",
          { headers: { Accept: "application/json" } }
        );
        if (res.ok) {
          const data = await res.json();
          // message 字段可能是 "128" 或 "1.2k" 等格式
          const count = this.parseStarCount(data.message);
          this.starCount = count;
        } else {
          console.warn("[AppSidebar] Shields API error:", res.status);
          this.starCount = 0;
        }
      } catch (err) {
        console.warn("[AppSidebar] Failed to fetch star count:", err);
        this.starCount = 0;
      }
    },
    parseStarCount(message) {
      if (!message || message === "unknown") return 0;
      // 处理 "1.2k" 格式
      if (message.endsWith("k")) {
        return Math.round(parseFloat(message) * 1000);
      }
      const num = parseInt(message, 10);
      return isNaN(num) ? 0 : num;
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
  font-size: 26px;
  line-height: 1;
  flex-shrink: 0;
}

.brand-name {
  color: #fff;
  font-family: 'Orbitron', sans-serif;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.5px;
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

.sidebar-footer {
  flex-shrink: 0;
  border-top: 1px solid #2c2f36;
  padding: 10px 16px;
}

.sidebar-footer.collapsed {
  padding: 10px 0;
  text-align: center;
}

.footer-gh {
  display: flex;
  align-items: center;
  gap: 7px;
  text-decoration: none;
  color: #6b7280;
  font-size: 12px;
  transition: color 0.2s;
}

.sidebar-footer.collapsed .footer-gh {
  justify-content: center;
}

.footer-gh:hover {
  color: #c9d1d9;
}

.gh-icon {
  flex-shrink: 0;
  opacity: 0.85;
}

.gh-stars {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: #e3b341;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.star-icon {
  flex-shrink: 0;
}

.gh-author {
  color: #6b7280;
  white-space: nowrap;
}

.footer-gh:hover .gh-author {
  color: #9ca3af;
}
</style>
