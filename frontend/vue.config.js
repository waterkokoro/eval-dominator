const { defineConfig } = require('@vue/cli-service')

module.exports = defineConfig({
  transpileDependencies: true,
  devServer: {
    port: 8081,
    proxy: {
      '/api': {
        target: process.env.VUE_APP_API_BASE_URL || 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
