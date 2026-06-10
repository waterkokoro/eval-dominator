<template>
  <div class="eval-steps-wrapper">
    <div class="eval-steps-track">
      <div
        v-for="(step, idx) in steps"
        :key="step.key"
        class="step-item"
        :class="{
          'step-active': step.state === 'active',
          'step-done': step.state === 'done',
          'step-error': step.state === 'error',
          'step-wait': step.state === 'wait',
        }"
      >
        <!-- Connecting line (before node, not for first) -->
        <div
          v-if="idx > 0"
          class="step-line"
          :class="{ 'line-done': step.state !== 'wait', 'line-error': step.state === 'error' }"
        >
          <div class="step-line-base" />
          <div class="step-line-fill" />
          <!-- 流动光带：仅在 active 节点的左侧线上显示 -->
          <template v-if="step.state === 'active' && !isTerminal">
            <div class="step-line-flow" />
            <div class="step-line-particles">
              <span class="particle" />
              <span class="particle" />
              <span class="particle" />
            </div>
          </template>
        </div>

        <!-- Node -->
        <div class="step-node">
          <!-- 四角科技定位点（仅 active） -->
          <template v-if="step.state === 'active' && !isTerminal">
            <span class="corner corner-tl" />
            <span class="corner corner-tr" />
            <span class="corner corner-bl" />
            <span class="corner corner-br" />
          </template>

          <!-- 外层光晕（仅 active 非终态） -->
          <div v-if="step.state === 'active' && !isTerminal" class="step-halo" />
          <!-- 双层旋转环（仅 active 非终态） -->
          <div v-if="step.state === 'active' && !isTerminal" class="orbit orbit-outer" />
          <div v-if="step.state === 'active' && !isTerminal" class="orbit orbit-inner" />

          <!-- Progress ring（科技感渐变） -->
          <svg v-if="step.state === 'active' && hasRealProgress" class="progress-ring" viewBox="0 0 60 60">
            <defs>
              <linearGradient :id="ringGradId(step.key)" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#67e8f9" />
                <stop offset="50%" stop-color="#409eff" />
                <stop offset="100%" stop-color="#a78bfa" />
              </linearGradient>
            </defs>
            <circle class="ring-bg" cx="30" cy="30" r="26" fill="none" />
            <circle
              class="ring-fill"
              cx="30" cy="30" r="26"
              fill="none"
              :stroke="`url(#${ringGradId(step.key)})`"
              :stroke-dasharray="ringCircumference"
              :stroke-dashoffset="ringOffset"
            />
          </svg>

          <!-- 主节点圆 -->
          <div class="step-circle" :class="{ 'circle-pulse': step.state === 'active' && !isTerminal }">
            <span class="circle-shine" />
            <transition name="icon-fade" mode="out-in">
              <i v-if="step.state === 'active' && !isTerminal" key="spin" class="el-icon-loading step-icon" />
              <i v-else-if="step.state === 'done'" key="done" class="el-icon-check step-icon" />
              <i v-else-if="step.state === 'error'" key="err" class="el-icon-close step-icon" />
              <i v-else :class="step.icon" key="default" class="step-icon" />
            </transition>
          </div>

          <!-- 序号徽标 -->
          <span class="step-index">{{ stepIndexLabel(idx) }}</span>
        </div>

        <!-- Label -->
        <div class="step-label">
          <span class="step-title">{{ step.title }}</span>
          <span v-if="step.state === 'active' && hasRealProgress" class="step-pct">{{ progressLabel }}</span>
          <span
            v-else-if="step.state === 'active' && !hasRealProgress && !isTerminal"
            class="step-pct step-pct-pulse"
          >…</span>
          <span v-else-if="step.subText" class="step-sub">{{ step.subText }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { getEvalStatusText, isEvalStatusFinal } from "@/constants/eval-status";

// 5 节点：prepare(0) -> build(1) -> infer(2) -> evaluate(3) -> finish(4)
// running 状态下根据 runningPhase 决定停在 infer 还是 evaluate；
// parsing 阶段视为已经离开 evaluate（停在 evaluate 完成态）。
const STATUS_BASE_STEP = {
  pending: 0,
  validating: 0,
  building: 1,
  running: 2, // 默认 infer，下面会按 runningPhase 修正
  parsing: 3,
  succeeded: 4,
  failed: 4,
  timeout: 4,
  cancelled: 4,
};

export default {
  name: "EvalSteps",
  props: {
    status: { type: String, default: "" },
    progress: { type: Number, default: -1 },
    progressText: { type: String, default: "" },
    runningPhase: { type: String, default: "" }, // "infer" / "eval" / ""
  },
  computed: {
    activeStep() {
      const base = STATUS_BASE_STEP[this.status] ?? 0;
      if (this.status === "running" && this.runningPhase === "eval") return 3;
      if (this.status === "running") return 2;
      return base;
    },
    isTerminal() {
      return isEvalStatusFinal(this.status);
    },
    isTerminalError() {
      return this.status === "failed" || this.status === "timeout" || this.status === "cancelled";
    },
    hasRealProgress() {
      return this.progress >= 0 && this.progress <= 100;
    },
    runningPercent() {
      return this.hasRealProgress ? this.progress : 0;
    },
    progressLabel() {
      if (this.progressText) return this.progressText;
      return this.runningPercent + "%";
    },
    ringCircumference() {
      return 2 * Math.PI * 26; // r=26
    },
    ringOffset() {
      return this.ringCircumference * (1 - this.runningPercent / 100);
    },
    steps() {
      const t = (key) => this.$t(`eval.steps.${key}`);
      const defs = [
        { key: "prepare", title: t("prepare"), icon: "el-icon-document" },
        { key: "build", title: t("build"), icon: "el-icon-set-up" },
        { key: "infer", title: t("infer"), icon: "el-icon-cpu" },
        { key: "evaluate", title: t("evaluate"), icon: "el-icon-data-analysis" },
        {
          key: "finish",
          title: t("finish"),
          icon: this.isTerminalError ? "el-icon-circle-close" : "el-icon-circle-check",
        },
      ];
      return defs.map((d, idx) => ({
        ...d,
        state: this.stepState(idx),
        subText: this.stepSub(idx),
      }));
    },
  },
  methods: {
    ringGradId(key) {
      return `eval-ring-grad-${key}`;
    },
    stepIndexLabel(idx) {
      return String(idx + 1).padStart(2, "0");
    },
    stepState(idx) {
      const current = this.activeStep;
      if (idx < current) return "done";
      if (idx === current) {
        if (this.isTerminalError) return "error";
        if (this.isTerminal && idx === 4) return this.status === "succeeded" ? "done" : "error";
        return "active";
      }
      return "wait";
    },
    stepSub(idx) {
      if (this.stepState(idx) !== "error") return "";
      // Error 节点展示状态文案
      return getEvalStatusText(this.status);
    },
  },
};
</script>

<style scoped>
.eval-steps-wrapper {
  width: 100%;
  padding: 16px 4px 6px;
}

.eval-steps-track {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  position: relative;
}

/* ---- Step item ---- */
.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
  min-width: 0;
}

/* ---- Connecting line ---- */
.step-line {
  position: absolute;
  top: 24px;
  right: 50%;
  width: 100%;
  height: 4px;
  z-index: 0;
  border-radius: 4px;
  overflow: hidden;
}
.step-line-base {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, rgba(148, 163, 184, 0.18), rgba(148, 163, 184, 0.28), rgba(148, 163, 184, 0.18));
  border-radius: 4px;
}
.step-line-fill {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  width: 0;
  border-radius: 4px;
  transition: width 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}
.step-line.line-done .step-line-fill {
  width: 100%;
  background: linear-gradient(90deg, #22d3ee 0%, #3b82f6 50%, #8b5cf6 100%);
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.55);
}
.step-line.line-error .step-line-fill {
  width: 100%;
  background: linear-gradient(90deg, #f59e0b, #ef4444);
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.5);
}
/* 流动光带：active 节点左侧线条上跑一道亮光 */
.step-line-flow {
  position: absolute;
  top: 0;
  left: -45%;
  width: 45%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(186, 248, 255, 0.95), transparent);
  filter: blur(0.4px);
  animation: flow-stream 1.8s linear infinite;
}
@keyframes flow-stream {
  0% { left: -45%; }
  100% { left: 100%; }
}

/* 流动粒子：3 颗有错落感 */
.step-line-particles {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.step-line-particles .particle {
  position: absolute;
  top: 50%;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #e0f7ff;
  box-shadow: 0 0 8px #67e8f9, 0 0 14px rgba(103, 232, 249, 0.6);
  transform: translate(-50%, -50%);
  animation: particle-fly 2.4s linear infinite;
}
.step-line-particles .particle:nth-child(2) { animation-delay: 0.8s; }
.step-line-particles .particle:nth-child(3) { animation-delay: 1.6s; }
@keyframes particle-fly {
  0% { left: 0%; opacity: 0; transform: translate(-50%, -50%) scale(0.6); }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { left: 100%; opacity: 0; transform: translate(-50%, -50%) scale(0.6); }
}

/* ---- Node ---- */
.step-node {
  position: relative;
  z-index: 1;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
}

.step-circle {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  box-sizing: border-box;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.35s ease, border-color 0.35s ease, box-shadow 0.35s ease;
  position: relative;
  z-index: 2;
  border: 2px solid transparent;
  overflow: hidden;
}

/* 高光斜面 */
.circle-shine {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.45) 0%, rgba(255, 255, 255, 0) 45%);
  pointer-events: none;
  mix-blend-mode: screen;
}

/* Wait */
.step-wait .step-circle {
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e7ed 100%);
  border-color: rgba(148, 163, 184, 0.45);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.6);
}
.step-wait .step-icon {
  color: #9ca3af;
  font-size: 16px;
}
.step-wait .circle-shine { opacity: 0.4; }

/* Active：等离子渐变 + 内圈高光 */
.step-active .step-circle {
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.55), rgba(255, 255, 255, 0) 55%),
    conic-gradient(from 0deg, #22d3ee, #3b82f6, #8b5cf6, #22d3ee);
  border-color: rgba(186, 230, 253, 0.85);
  box-shadow:
    0 0 0 4px rgba(59, 130, 246, 0.16),
    0 4px 14px rgba(59, 130, 246, 0.35),
    0 0 18px rgba(34, 211, 238, 0.55),
    inset 0 0 10px rgba(255, 255, 255, 0.35);
  animation: spin-conic 6s linear infinite;
}
@keyframes spin-conic {
  from {
    background:
      radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.55), rgba(255, 255, 255, 0) 55%),
      conic-gradient(from 0deg, #22d3ee, #3b82f6, #8b5cf6, #22d3ee);
  }
  to {
    background:
      radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.55), rgba(255, 255, 255, 0) 55%),
      conic-gradient(from 360deg, #22d3ee, #3b82f6, #8b5cf6, #22d3ee);
  }
}
.step-active .step-icon {
  color: #fff;
  font-size: 19px;
  text-shadow: 0 0 8px rgba(255, 255, 255, 0.85), 0 0 14px rgba(103, 232, 249, 0.7);
  position: relative;
  z-index: 1;
}

/* Done */
.step-done .step-circle {
  background: linear-gradient(135deg, #10b981 0%, #34d399 60%, #6ee7b7 100%);
  border-color: rgba(110, 231, 183, 0.8);
  box-shadow:
    0 2px 10px rgba(16, 185, 129, 0.35),
    inset 0 0 8px rgba(255, 255, 255, 0.3);
}
.step-done .step-icon {
  color: #fff;
  font-size: 19px;
  font-weight: bold;
  text-shadow: 0 0 6px rgba(16, 185, 129, 0.6);
  position: relative;
  z-index: 1;
}

/* Error */
.step-error .step-circle {
  background: linear-gradient(135deg, #ef4444 0%, #f87171 100%);
  border-color: rgba(252, 165, 165, 0.85);
  box-shadow:
    0 0 0 4px rgba(239, 68, 68, 0.14),
    0 2px 10px rgba(239, 68, 68, 0.4),
    inset 0 0 8px rgba(255, 255, 255, 0.25);
  animation: shake-error 0.5s ease-in-out 1;
}
.step-error .step-icon {
  color: #fff;
  font-size: 19px;
  font-weight: bold;
  text-shadow: 0 0 6px rgba(239, 68, 68, 0.6);
  position: relative;
  z-index: 1;
}
@keyframes shake-error {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-2px); }
  75% { transform: translateX(2px); }
}

/* Pulse animation for active non-terminal */
.circle-pulse {
  animation: spin-conic 6s linear infinite, pulse-ring 2.4s ease-in-out infinite;
}
@keyframes pulse-ring {
  0% {
    box-shadow:
      0 0 0 0 rgba(59, 130, 246, 0.45),
      0 4px 14px rgba(59, 130, 246, 0.35),
      0 0 18px rgba(34, 211, 238, 0.55),
      inset 0 0 10px rgba(255, 255, 255, 0.35);
  }
  50% {
    box-shadow:
      0 0 0 14px rgba(59, 130, 246, 0),
      0 4px 18px rgba(139, 92, 246, 0.5),
      0 0 24px rgba(34, 211, 238, 0.7),
      inset 0 0 10px rgba(255, 255, 255, 0.4);
  }
  100% {
    box-shadow:
      0 0 0 0 rgba(59, 130, 246, 0),
      0 4px 14px rgba(59, 130, 246, 0.35),
      0 0 18px rgba(34, 211, 238, 0.55),
      inset 0 0 10px rgba(255, 255, 255, 0.35);
  }
}

/* 双层旋转环 */
.orbit {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
}
.orbit-outer {
  top: -10px;
  left: -10px;
  width: 68px;
  height: 68px;
  border: 1.5px dashed rgba(34, 211, 238, 0.55);
  z-index: 1;
  animation: halo-spin 7s linear infinite;
}
.orbit-inner {
  top: -4px;
  left: -4px;
  width: 56px;
  height: 56px;
  border: 1px dashed rgba(167, 139, 250, 0.55);
  z-index: 1;
  animation: halo-spin 4s linear infinite reverse;
}
@keyframes halo-spin {
  from { transform: rotate(0); }
  to { transform: rotate(360deg); }
}

/* 外层光晕（柔和发光） */
.step-halo {
  position: absolute;
  top: -14px;
  left: -14px;
  width: 76px;
  height: 76px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.28) 0%, rgba(59, 130, 246, 0.18) 35%, rgba(139, 92, 246, 0) 70%);
  z-index: 0;
  pointer-events: none;
  animation: halo-breath 2.8s ease-in-out infinite;
}
@keyframes halo-breath {
  0%, 100% { transform: scale(1); opacity: 0.85; }
  50% { transform: scale(1.12); opacity: 1; }
}

/* 四角科技定位点 */
.corner {
  position: absolute;
  width: 8px;
  height: 8px;
  border-style: solid;
  border-color: rgba(34, 211, 238, 0.85);
  z-index: 4;
  filter: drop-shadow(0 0 3px rgba(34, 211, 238, 0.7));
  animation: corner-blink 2s ease-in-out infinite;
}
.corner-tl { top: -10px; left: -10px; border-width: 1.5px 0 0 1.5px; }
.corner-tr { top: -10px; right: -10px; border-width: 1.5px 1.5px 0 0; }
.corner-bl { bottom: -10px; left: -10px; border-width: 0 0 1.5px 1.5px; }
.corner-br { bottom: -10px; right: -10px; border-width: 0 1.5px 1.5px 0; }
@keyframes corner-blink {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
}

/* ---- Progress ring ---- */
.progress-ring {
  position: absolute;
  top: -6px;
  left: -6px;
  width: 60px;
  height: 60px;
  z-index: 3;
  transform: rotate(-90deg);
}
.ring-bg {
  stroke: rgba(148, 163, 184, 0.18);
  stroke-width: 3;
}
.ring-fill {
  stroke-width: 3.5;
  stroke-linecap: round;
  filter: drop-shadow(0 0 5px rgba(103, 232, 249, 0.85));
  transition: stroke-dashoffset 0.6s cubic-bezier(0.4, 0, 0.2, 1);
}

/* 序号徽标 */
.step-index {
  position: absolute;
  top: -8px;
  right: -10px;
  z-index: 5;
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.5px;
  padding: 1px 5px;
  border-radius: 8px;
  font-family: "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace;
  line-height: 1.4;
  background: rgba(255, 255, 255, 0.92);
  color: #94a3b8;
  border: 1px solid rgba(148, 163, 184, 0.35);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
  transition: all 0.3s ease;
}
.step-active .step-index {
  background: linear-gradient(135deg, #22d3ee, #3b82f6);
  color: #fff;
  border-color: rgba(186, 230, 253, 0.9);
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.55);
}
.step-done .step-index {
  background: linear-gradient(135deg, #10b981, #34d399);
  color: #fff;
  border-color: rgba(110, 231, 183, 0.85);
}
.step-error .step-index {
  background: linear-gradient(135deg, #ef4444, #f87171);
  color: #fff;
  border-color: rgba(252, 165, 165, 0.9);
}

/* ---- Label ---- */
.step-label {
  margin-top: 14px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
}
.step-title {
  font-size: 13px;
  font-weight: 600;
  color: #475569;
  transition: color 0.3s;
  white-space: nowrap;
  letter-spacing: 0.2px;
}
.step-active .step-title {
  background: linear-gradient(90deg, #22d3ee, #3b82f6, #8b5cf6);
  background-size: 200% 100%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: title-gradient 3s ease-in-out infinite;
  font-weight: 700;
}
@keyframes title-gradient {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}
.step-done .step-title { color: #10b981; }
.step-error .step-title { color: #ef4444; }
.step-wait .step-title { color: #94a3b8; }

.step-pct {
  font-size: 12px;
  font-weight: 700;
  color: #3b82f6;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.5px;
  font-family: "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace;
  padding: 1px 6px;
  border-radius: 6px;
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.12), rgba(139, 92, 246, 0.12));
  border: 1px solid rgba(59, 130, 246, 0.25);
}
.step-pct-pulse {
  animation: pct-blink 1.4s ease-in-out infinite;
}
@keyframes pct-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.step-sub {
  font-size: 11px;
  color: #94a3b8;
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ---- Icon transition ---- */
.icon-fade-enter-active,
.icon-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.icon-fade-enter,
.icon-fade-leave-to {
  opacity: 0;
  transform: scale(0.6);
}
</style>
