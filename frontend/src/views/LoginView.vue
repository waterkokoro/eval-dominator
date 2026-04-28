<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <div class="brand-mark">ED</div>
        <div class="brand-meta">
          <div class="brand-title">Eval Dominator</div>
          <div class="brand-sub">轻量级模型评测中心</div>
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
            placeholder="用户名"
            autocomplete="username"
            size="medium"
          />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            prefix-icon="el-icon-lock"
            placeholder="密码"
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
          登录
        </el-button>
      </el-form>
      <div class="login-tip">登录后可使用提交评测、任务管理等功能。</div>
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
      },
      rules: {
        username: [
          { required: true, message: "请输入用户名", trigger: "blur" }
        ],
        password: [
          { required: true, message: "请输入密码", trigger: "blur" }
        ]
      }
    };
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
        this.$message.success("登录成功");
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
