import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Session } from '../types'
import { fetchSessions, fetchSession } from '../api'

export interface ProjectGroup {
  project: string
  sessions: Session[]
  session_count: number
  total_cost: number
  total_tokens: number
  last_activity: string
}

function tokensOf(s: Session): number {
  return s.total_input + s.total_output + s.total_cache_read + s.total_cache_write
}

export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<Session[]>([])
  const total = ref(0)
  const limit = ref(25)
  const offset = ref(0)
  const sortBy = ref('cost')
  const sortDir = ref<'asc' | 'desc'>('desc')
  const selectedSession = ref<Session | null>(null)
  const loading = ref(false)

  // Grouping: collapse the flat session list into per-project groups.
  const groupByProject = ref(false)
  const allSessions = ref<Session[]>([])
  const expandedProjects = ref<Set<string>>(new Set())

  async function load() {
    if (groupByProject.value) {
      return loadAll()
    }
    loading.value = true
    try {
      const res = await fetchSessions(limit.value, offset.value, sortBy.value, sortDir.value)
      sessions.value = res.sessions || []
      total.value = res.total
    } finally {
      loading.value = false
    }
  }

  // Grouped mode needs every session at once so per-project aggregates are
  // correct across pages. The dataset is small (one row per Claude Code
  // session), so a single unpaginated fetch is fine.
  async function loadAll() {
    loading.value = true
    try {
      const res = await fetchSessions(100000, 0, sortBy.value, sortDir.value)
      allSessions.value = res.sessions || []
      total.value = res.total
    } finally {
      loading.value = false
    }
  }

  const projectGroups = computed<ProjectGroup[]>(() => {
    const map = new Map<string, ProjectGroup>()
    for (const s of allSessions.value) {
      let g = map.get(s.project)
      if (!g) {
        g = { project: s.project, sessions: [], session_count: 0, total_cost: 0, total_tokens: 0, last_activity: '' }
        map.set(s.project, g)
      }
      g.sessions.push(s)
      g.session_count++
      g.total_cost += s.total_cost
      g.total_tokens += tokensOf(s)
      if (s.last_activity > g.last_activity) g.last_activity = s.last_activity
    }
    // Sessions arrive already ordered by the active sort; only order the groups.
    return [...map.values()].sort((a, b) => b.total_cost - a.total_cost)
  })

  function setGroupByProject(on: boolean) {
    groupByProject.value = on
    offset.value = 0
    load()
  }

  function toggleProject(project: string) {
    const next = new Set(expandedProjects.value)
    if (next.has(project)) next.delete(project)
    else next.add(project)
    expandedProjects.value = next
  }

  function isExpanded(project: string): boolean {
    return expandedProjects.value.has(project)
  }

  function setSort(col: string) {
    if (sortBy.value === col) {
      sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
    } else {
      sortBy.value = col
      sortDir.value = 'desc'
    }
    offset.value = 0
    load()
  }

  function nextPage() {
    if (offset.value + limit.value < total.value) {
      offset.value += limit.value
      load()
    }
  }

  function prevPage() {
    if (offset.value > 0) {
      offset.value = Math.max(0, offset.value - limit.value)
      load()
    }
  }

  async function selectSession(id: string) {
    selectedSession.value = await fetchSession(id)
  }

  function clearSelection() {
    selectedSession.value = null
  }

  return {
    sessions, total, limit, offset, sortBy, sortDir,
    selectedSession, loading,
    groupByProject, projectGroups, expandedProjects,
    load, setSort, nextPage, prevPage, selectSession, clearSelection,
    setGroupByProject, toggleProject, isExpanded,
  }
})
