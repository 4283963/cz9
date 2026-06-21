import React, { useState } from 'react'
import RepoItem from './RepoItem.jsx'
import { retryRepo, deleteRepo } from '../api/index.js'

const filters = [
  { key: '', label: '全部' },
  { key: 'pending', label: '待处理' },
  { key: 'running', label: '运行中' },
  { key: 'success', label: '成功' },
  { key: 'failed', label: '失败' },
]

export default function RepoList({ repos, loading, filter, setFilter, onRepoChanged, showToast }) {
  const [expandedLogs, setExpandedLogs] = useState({})

  const toggleLog = (id) => {
    setExpandedLogs((prev) => ({ ...prev, [id]: !prev[id] }))
  }

  const handleRetry = async (repo) => {
    try {
      await retryRepo(repo.id)
      showToast('info', `已重新提交 ${repo.name}`)
      onRepoChanged()
    } catch (err) {
      showToast('error', err.response?.data?.error || '重试失败')
    }
  }

  const handleDelete = async (repo) => {
    if (!confirm(`确定删除记录 ${repo.name} 吗？（不会删除 GitHub 上的仓库）`)) return
    try {
      await deleteRepo(repo.id)
      showToast('info', `已删除 ${repo.name}`)
      onRepoChanged()
    } catch (err) {
      showToast('error', err.response?.data?.error || '删除失败')
    }
  }

  const filteredRepos = repos.filter((r) => {
    if (!filter) return true
    if (filter === 'running') {
      return ['local_init', 'remote_create', 'first_push'].includes(r.status)
    }
    return r.status === filter
  })

  return (
    <div className="panel">
      <div className="filter-bar">
        {filters.map((f) => (
          <button
            key={f.key || 'all'}
            className={`filter-chip ${filter === f.key ? 'active' : ''}`}
            onClick={() => setFilter(f.key)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="empty-state">
          <div className="empty-state-icon">⏳</div>
          <div className="empty-state-text">加载中...</div>
        </div>
      ) : filteredRepos.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">📭</div>
          <div className="empty-state-text">暂无仓库记录</div>
        </div>
      ) : (
        <div className="repo-list">
          {filteredRepos.map((repo) => (
            <RepoItem
              key={repo.id}
              repo={repo}
              expanded={!!expandedLogs[repo.id]}
              onToggleLog={() => toggleLog(repo.id)}
              onRetry={() => handleRetry(repo)}
              onDelete={() => handleDelete(repo)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
