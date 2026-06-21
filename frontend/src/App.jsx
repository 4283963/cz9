import React, { useState, useEffect, useCallback } from 'react'
import BatchForm from './components/BatchForm.jsx'
import RepoList from './components/RepoList.jsx'
import StatsBar from './components/StatsBar.jsx'
import { fetchRepos, fetchStats, deleteAllRepos } from './api/index.js'

function App() {
  const [repos, setRepos] = useState([])
  const [stats, setStats] = useState({ total: 0, pending: 0, running: 0, success: 0, failed: 0 })
  const [filter, setFilter] = useState('')
  const [toast, setToast] = useState(null)
  const [loading, setLoading] = useState(true)

  const showToast = useCallback((type, message) => {
    setToast({ type, message })
    setTimeout(() => setToast(null), 3000)
  }, [])

  const loadRepos = useCallback(async () => {
    try {
      const res = await fetchRepos(filter)
      setRepos(res.data.data || [])
    } catch (err) {
      console.error('Load repos error:', err)
    }
  }, [filter])

  const loadStats = useCallback(async () => {
    try {
      const res = await fetchStats()
      setStats(res.data.data || {})
    } catch (err) {
      console.error('Load stats error:', err)
    }
  }, [])

  const refreshAll = useCallback(async () => {
    await Promise.all([loadRepos(), loadStats()])
    setLoading(false)
  }, [loadRepos, loadStats])

  useEffect(() => {
    refreshAll()
  }, [refreshAll])

  useEffect(() => {
    loadRepos()
  }, [loadRepos])

  useEffect(() => {
    const hasRunning = repos.some(
      (r) => ['pending', 'local_init', 'remote_create', 'first_push'].includes(r.status),
    )
    if (hasRunning) {
      const timer = setInterval(refreshAll, 2000)
      return () => clearInterval(timer)
    }
  }, [repos, refreshAll])

  const handleBatchCreated = () => {
    showToast('success', '批量创建任务已提交')
    refreshAll()
  }

  const handleRepoChanged = () => {
    refreshAll()
  }

  const handleClearAll = async () => {
    if (!confirm('确定要清空所有记录吗？此操作不可恢复。')) return
    try {
      await deleteAllRepos()
      showToast('info', '已清空所有记录')
      refreshAll()
    } catch (err) {
      showToast('error', err.response?.data?.error || '清空失败')
    }
  }

  return (
    <div className="app">
      {toast && <div className={`toast ${toast.type}`}>{toast.message}</div>}

      <div className="header">
        <h1>🚀 GitHub 批量仓库管理</h1>
        <button className="btn btn-danger btn-sm" onClick={handleClearAll}>
          清空记录
        </button>
      </div>

      <StatsBar stats={stats} />

      <div className="main-content">
        <BatchForm onCreated={handleBatchCreated} showToast={showToast} />
        <RepoList
          repos={repos}
          loading={loading}
          filter={filter}
          setFilter={setFilter}
          onRepoChanged={handleRepoChanged}
          showToast={showToast}
        />
      </div>
    </div>
  )
}

export default App
