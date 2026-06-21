import React from 'react'

const statusMap = {
  pending: { text: '待处理', badge: 'pending', step: 0 },
  local_init: { text: '本地初始化中', badge: 'running', step: 1 },
  remote_create: { text: '远程创建中', badge: 'running', step: 2 },
  first_push: { text: '首次推送中', badge: 'running', step: 3 },
  success: { text: '成功', badge: 'success', step: 4 },
  failed: { text: '失败', badge: 'failed', step: -1 },
}

const steps = ['待处理', '本地初始化', '远程创建', '首次推送', '完成']

export default function RepoItem({ repo, expanded, onToggleLog, onRetry, onDelete }) {
  const info = statusMap[repo.status] || statusMap.pending
  const totalSteps = 4

  const getProgress = () => {
    if (repo.status === 'success') return 100
    if (repo.status === 'failed') return 100
    if (repo.status === 'pending') return 0
    return (info.step / totalSteps) * 100
  }

  const formatTime = (t) => {
    if (!t) return ''
    return new Date(t).toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const getStepClass = (index) => {
    if (repo.status === 'failed' && index === info.step) return 'failed'
    if (repo.status === 'success') return 'done'
    if (info.step === -1 && index === 0) return ''
    if (index < info.step) return 'done'
    if (index === info.step) return 'current'
    return ''
  }

  const isFinished = repo.status === 'success' || repo.status === 'failed'
  const canRetry = isFinished

  return (
    <div className="repo-card">
      <div className="repo-header">
        <div>
          <div className="repo-name">{repo.name}</div>
          {repo.description && (
            <div className="repo-desc">{repo.description}</div>
          )}
        </div>
        <span className={`status-badge ${info.badge}`}>
          <span className="status-dot" />
          {info.text}
        </span>
      </div>

      <div className="progress-track">
        <div
          className={`progress-bar ${info.badge}`}
          style={{ width: `${getProgress()}%` }}
        />
      </div>

      <div className="progress-steps">
        {steps.map((s, i) => (
          <div key={s} className={`step ${getStepClass(i)}`}>
            {s}
          </div>
        ))}
      </div>

      <div className="repo-actions">
        <div className="repo-meta">
          创建于 {formatTime(repo.created_at)}
          {repo.finished_at && ` · 耗时 ${((new Date(repo.finished_at) - new Date(repo.created_at)) / 1000).toFixed(1)}s`}
        </div>
        <div className="repo-actions-right">
          {repo.log && (
            <button className="toggle-log-btn" onClick={onToggleLog}>
              {expanded ? '收起日志' : '查看日志'}
            </button>
          )}
          {canRetry && (
            <button className="btn btn-secondary btn-sm" onClick={onRetry}>
              🔄 重试
            </button>
          )}
          <button className="btn btn-danger btn-sm" onClick={onDelete}>
            🗑️ 删除
          </button>
        </div>
      </div>

      {expanded && repo.log && <div className="log-viewer">{repo.log}</div>}
    </div>
  )
}
