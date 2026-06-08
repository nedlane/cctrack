<template>
  <div class="metric-strip">
    <div
      class="metric"
      v-for="(m, i) in metrics"
      :key="m.label"
      :style="{ animationDelay: 40 + i * 60 + 'ms' }"
    >
      <div class="metric-label">{{ m.label }}</div>
      <div class="metric-value" :class="{ accent: m.accent }">{{ m.value }}</div>
      <div class="metric-sub">{{ m.sub }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface Metric {
  label: string
  value: string
  sub?: string
  accent?: boolean
}
defineProps<{ metrics: Metric[] }>()
</script>

<style scoped>
.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-8);
}
.metric {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-4) var(--space-5);
  animation: fadeSlideUp 0.45s ease both;
}
.metric-label {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  margin-bottom: var(--space-2);
}
.metric-value {
  font-family: 'JetBrains Mono', monospace;
  font-size: 22px;
  font-weight: 500;
  color: var(--text-primary);
  line-height: 1.1;
}
.metric-value.accent {
  color: var(--amber-400);
}
.metric-sub {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-tertiary);
}

@media (max-width: 720px) {
  .metric-strip {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
