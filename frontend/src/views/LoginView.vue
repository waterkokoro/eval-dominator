<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <div class="brand-mark">ED</div>
        <div class="brand-meta">
          <div class="brand-title">Eval Dominator</div>
          <div class="brand-sub">{{ $t("auth.login.brandSub") }}</div>
        </div>
      </div>
      <el-form
        ref="form"
        :model="form"
        :rules="rules"
        label-width="0"
        @submit.native.prevent
      >
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            prefix-icon="el-icon-user"
            :placeholder="$t('auth.login.username')"
            autocomplete="username"
            size="medium"
          />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            prefix-icon="el-icon-lock"
            :placeholder="$t('auth.login.password')"
            type="password"
            autocomplete="current-password"
            size="medium"
            show-password
          />
        </el-form-item>
        <el-button
          type="primary"
          :loading="loading"
          class="login-button"
          @click="handleLogin"
        >
          {{ $t("auth.login.submit") }}
        </el-button>
      </el-form>
      <div class="login-tip">{{ $t("auth.login.tip") }}</div>
    </div>
  </div>
</template>

<script>
import { login } from "@/api/auth";
import { loginSuccess, setUser } from "@/store/user";
import { fetchCurrentUser } from "@/api/auth";

export default {
  name: "LoginView",
  data() {
    return {
      loading: false,
      form: {
        username: "",
        password: ""
      }
    };
  },
  computed: {
    rules() {
      return {
        username: [
          { required: true, message: this.$t("auth.login.usernameRequired"), trigger: "blur" }
        ],
        password: [
          { required: true, message: this.$t("auth.login.passwordRequired"), trigger: "blur" }
        ]
      };
    }
  },
  methods: {
    async handleLogin() {
      const valid = await this.$refs.form.validate().catch(() => false);
      if (!valid) return;
      this.loading = true;
      try {
        const response = await login(this.form);
        loginSuccess(response.token);
        try {
          const me = await fetchCurrentUser();
          if (me) setUser(me);
          else setUser({ username: this.form.username });
        } catch (e) {
          setUser({ username: this.form.username });
        }
        this.$message.success(this.$t("auth.login.success"));
        const redirect = this.$route.query.redirect || "/eval/tasks";
        this.$router.replace(redirect);
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #1f2329 0%, #2d3a4d 100%);
}

.login-card {
  width: 420px;
  padding: 36px 40px 32px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.18);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
}

.brand-mark {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: linear-gradient(135deg, #409eff, #336cff);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 16px;
}

.brand-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.brand-sub {
  font-size: 13px;
  color: #909399;
  margin-top: 2px;
}

.login-button {
  width: 100%;
}

.login-tip {
  margin-top: 16px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}
</style>
