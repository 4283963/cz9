import React from 'react'

const cards = [
  { key: 'total', label: '总数', className: 'total' },
  { key: 'pending', label: '待处理', className: 'pending' },
  { key: 'running', label: '运行中', className: 'running' },
  { key: 'success', label: '成功', className: 'success' },
  { key: 'failed', label: '失败', className: 'failed' },
]

export default function StatsBar({ stats }) {
  return (
    <div className="stats-bar">
      {cards.map((c) => (
        <div key={c.key} className={`stat-card ${c.className}`}>
          <div className="label">{c.label}</div>
          <div className="value">{stats[c.key] || 0}</div>
        </div>
      ))}
    </div>
  )
}
