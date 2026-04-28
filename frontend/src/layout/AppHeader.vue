<template>
  <div class="app-header">
    <div class="left">
      <el-button
        size="mini"
        circle
        :icon="collapsed ? 'el-icon-s-unfold' : 'el-icon-s-fold'"
        @click="onToggle"
      />
      <el-breadcrumb separator="/" class="breadcrumb">
        <el-breadcrumb-item v-if="groupTitle">{{ groupTitle }}</el-breadcrumb-item>
        <el-breadcrumb-item>{{ pageTitle }}</el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="right">
      <el-tooltip :content="healthText" placement="bottom">
        <span class="health-badge" :class="`health-${healthLevel}`">
          <i class="el-icon-cpu" />
          <span>{{ healthLabel }}</span>
        </span>
      </el-tooltip>
      <el-dropdown
        trigger="click"
        @command="handleLangCommand"
      >
        <span class="user-trigger lang-trigger" :title="$t('common.language.switch')">
          <i class="el-icon-s-platform" />
          <span>{{ currentLangLabel }}</span>
          <i class="el-icon-arrow-down" />
        </span>
        <el-dropdown-menu slot="dropdown">
          <el-dropdown-item
            v-for="lang in languages"
            :key="lang.code"
            :command="lang.code"
            :disabled="lang.code === currentLang"
          >
            {{ lang.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </el-dropdown>
      <el-dropdown trigger="click" @command="handleCommand">
        <span class="user-trigger">
          <i class="el-icon-user-solid user-avatar" />
          <span class="user-name">{{ displayName }}</span>
          <i class="el-icon-arrow-down" />
        </span>
        <el-dropdown-menu slot="dropdown">
          <el-dropdown-item command="about">{{ $t("auth.header.about") }}</el-dropdown-item>
          <el-dropdown-item divided command="logout">{{ $t("auth.header.logout") }}</el-dropdown-item>
        </el-dropdown-menu>
      </el-dropdown>
    </div>
  </div>
</template>

<script>
import { appStore, toggleSidebar, setHealth } from "@/store/app";
import { userStore, logoutLocal } from "@/store/user";
import { getSystemHealth } from "@/api/system";
import { logout } from "@/api/auth";
import { LANGUAGES, setLanguage } from "@/locales";

export default {
  name: "AppHeader",
  data() {
    return {
      timer: null,
      languages: LANGUAGES.map((l) => ({ code: l.code, label: l.label }))
    };
  },
  computed: {
    collapsed() {
      return appStore.collapsed;
    },
    groupTitle() {
      const key = this.$route.meta?.groupKey;
      return key ? this.$t(key) : "";
    },
    pageTitle() {
      const key = this.$route.meta?.titleKey;
      return key ? this.$t(key) : "";
    },
    health() {
      return appStore.health;
    },
    healthLevel() {
      const { backend, core } = this.health;
      if (backend?.ok === null || core?.ok === null) return "unknown";
      if (backend?.ok && core?.ok) return "ok";
      return "down";
    },
    healthLabel() {
      const map = {
        ok: this.$t("system.health.ok"),
        down: this.$t("system.health.down"),
        unknown: this.$t("system.health.unknown")
      };
      return map[this.healthLevel];
    },
    healthText() {
      const { backend, core } = this.health;
      const fmt = (v) =>
        v?.ok === null
          ? this.$t("system.health.unknown_short")
          : v?.ok
            ? this.$t("system.health.ok_short")
            : v?.message || this.$t("system.health.down_short");
      return `Backend: ${fmt(backend)} / Core: ${fmt(core)}`;
    },
    displayName() {
      return userStore.user?.username || this.$t("auth.header.guest");
    },
    currentLang() {
      return this.$i18n.locale;
    },
    currentLangLabel() {
      return this.languages.find((l) => l.code === this.currentLang)?.label || this.currentLang;
    }
  },
  mounted() {
    this.refreshHealth();
    this.timer = setInterval(this.refreshHealth, 30000);
  },
  beforeDestroy() {
    if (this.timer) clearInterval(this.timer);
  },
  methods: {
    onToggle() {
      toggleSidebar();
    },
    handleLangCommand(code) {
      setLanguage(code);
    },
    async refreshHealth() {
      try {
        const data = await getSystemHealth();
        setHealth({
          backend: data?.backend || { ok: true, message: "ok" },
          core: data?.core || { ok: true, message: "ok" }
        });
      } catch (e) {
        setHealth({
          backend: { ok: false, message: this.$t("system.health.unreachable") },
          core: { ok: false, message: this.$t("system.health.unreachable") }
        });
      }
    },
    async handleCommand(command) {
      if (command === "about") {
        if (this.$route.name !== "about") this.$router.push({ name: "about" });
        return;
      }
      if (command === "logout") {
        try {
          await logout();
        } catch (e) {
          // 后端 logout 未实现时忽略，前端独立完成登出
        }
        logoutLocal();
        this.$router.replace({ name: "login" });
      }
    }
  }
};
</script>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 24px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
}

.left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.breadcrumb {
  font-size: 14px;
}

.health-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 12px;
  background: #f4f4f5;
  color: #909399;
  cursor: default;
}

.health-badge.health-ok {
  background: #f0f9eb;
  color: #67c23a;
}

.health-badge.health-down {
  background: #fef0f0;
  color: #f56c6c;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #303133;
  font-size: 14px;
}

.lang-trigger {
  color: #606266;
  font-size: 13px;
}

.user-avatar {
  font-size: 18px;
  color: #409eff;
}
</style>
