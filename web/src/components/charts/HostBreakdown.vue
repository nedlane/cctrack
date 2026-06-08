<template>
  <div class="host-card">
    <div class="chart-header">
      <div class="chart-title">Spend by Host</div>
      <div class="chart-meta" v-if="totalCost > 0">{{ formatCostDisplay(totalCost) }} total</div>
    </div>
    <div class="host-bars">
      <div v-for="(h, i) in rows" :key="h.host" class="host-bar-row">
        <div class="bar-label">
          <span class="bar-host">{{ h.host }}</span>
          <span class="bar-cost">{{ formatCostDisplay(h.total_cost) }}</span>
        </div>
        <div class="bar-track">
          <div
            class="bar-fill"
            :style="{ width: barWidth(h.total_cost) + '%', background: hostColors[i % hostColors.length] }"
          ></div>
        </div>
        <div class="bar-meta">
          <span>{{ h.pct }}% of spend</span>
          <span class="bar-sub">{{ formatTokens(h.total_tokens) }} tok · {{ h.session_count }} sessions</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { HostSummary } from '../../types'
import { formatCostDisplay, formatTokens } from '../../composables/useFormatCost'

const props = defineProps<{ hosts: HostSummary[] }>()

const hostColors = ['#f59e0b', '#fbbf24', '#78716c', '#44403c', '#a8a29e', '#d6d3d1']

const totalCost = computed(() => props.hosts.reduce((s, h) => s + h.total_cost, 0))

const rows = computed(() =>
  [...props.hosts]
    .sort((a, b) => b.total_cost - a.total_cost)
    .map(h => ({
      ...h,
      pct: totalCost.value > 0 ? Math.round((h.total_cost / totalCost.value) * 100) : 0,
    }))
)

function barWidth(cost: number) {
  const max = rows.value[0]?.total_cost || 1
  return Math.max((cost / max) * 100, 2)
}
</script>

<style scoped>
.host-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  padding: var(--space-6);
  animation: fadeSlideUp 0.45s ease both;
  animation-delay: 420ms;
}
.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-5);
}
.chart-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.chart-meta {
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-tertiary);
}
.host-bars {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.host-bar-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.bar-label {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.bar-host {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  font-family: 'JetBrains Mono', monospace;
}
.bar-cost {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--amber-400);
}
.bar-track {
  height: 6px;
  background: var(--bg-subtle);
  overflow: hidden;
}
.bar-fill {
  height: 100%;
  transition: width 800ms cubic-bezier(0.16, 1, 0.3, 1);
}
.bar-meta {
  display: flex;
  justify-content: space-between;
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: var(--text-tertiary);
}
.bar-sub {
  color: var(--text-disabled);
}
</style>
