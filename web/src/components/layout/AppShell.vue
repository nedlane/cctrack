<template>
  <div class="app-shell">
    <header class="mobile-topbar">
      <router-link class="mobile-logo" to="/" @click="navOpen = false">
        <div class="logo-glyph"></div>
        <span class="logo-text">CCTRACK</span>
      </router-link>
      <button
        class="nav-toggle"
        type="button"
        :aria-expanded="navOpen"
        aria-controls="app-sidebar"
        aria-label="Toggle navigation"
        @click="navOpen = !navOpen"
      >
        <svg v-if="!navOpen" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <path d="M3 6h18M3 12h18M3 18h18" stroke-linecap="round" />
        </svg>
        <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" />
        </svg>
      </button>
    </header>

    <div v-if="navOpen" class="nav-backdrop" @click="navOpen = false"></div>

    <SidebarNav
      id="app-sidebar"
      :class="{ open: navOpen }"
      :connectionStatus="connectionStatus"
      @navigate="navOpen = false"
    />
    <main class="main">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { ConnectionStatus } from '../../types'
import SidebarNav from './SidebarNav.vue'

defineProps<{ connectionStatus: ConnectionStatus }>()

// Mobile off-canvas nav state. Closes on any route change so the drawer never
// lingers over the page after navigating.
const navOpen = ref(false)
const route = useRoute()
watch(() => route.fullPath, () => { navOpen.value = false })
</script>

<style scoped>
.app-shell {
  display: flex;
  min-height: 100vh;
}
.main {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-10) var(--space-10) var(--space-16);
  max-width: calc(1200px + var(--space-10) * 2);
}
.main::-webkit-scrollbar { width: 6px; }
.main::-webkit-scrollbar-track { background: transparent; }
.main::-webkit-scrollbar-thumb { background: var(--border-default); }
.main::-webkit-scrollbar-thumb:hover { background: var(--border-strong); }

/* Mobile top bar + hamburger — hidden on desktop, where the sidebar is inline. */
.mobile-topbar { display: none; }
.nav-backdrop { display: none; }

@media (max-width: 768px) {
  .mobile-topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 52px;
    padding: 0 var(--space-4);
    background: var(--bg-surface);
    border-bottom: 1px solid var(--border-subtle);
    z-index: 60;
  }
  .mobile-logo {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    text-decoration: none;
  }
  .mobile-logo .logo-glyph {
    width: 20px;
    height: 20px;
    background: var(--amber-500);
    position: relative;
    flex-shrink: 0;
  }
  .mobile-logo .logo-glyph::after {
    content: '';
    position: absolute;
    inset: 3px;
    background: var(--bg-surface);
  }
  .mobile-logo .logo-text {
    font-family: 'Bebas Neue', sans-serif;
    font-size: 20px;
    letter-spacing: 0.06em;
    color: var(--text-primary);
    line-height: 1;
  }
  .nav-toggle {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    margin-right: -8px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .nav-toggle:active { color: var(--text-primary); }

  .nav-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    z-index: 70;
    animation: fadeIn 150ms ease;
  }

  .main {
    padding: calc(52px + var(--space-5)) var(--space-4) var(--space-12);
    max-width: 100%;
  }
}
</style>
