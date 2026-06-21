import React, { useState } from 'react'
import { batchCreate } from '../api/index.js'

export default function BatchForm({ onCreated, showToast }) {
  const [prefix, setPrefix] = useState('cc')
  const [start, setStart] = useState(1)
  const [end, setEnd] = useState(10)
  const [description, setDescription] = useState('Initial commit')
  const [priv, setPriv] = useState(false)
  const [template, setTemplate] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!prefix.trim()) {
      showToast('error', '请输入前缀')
      return
    }
    if (end < start) {
      showToast('error', '结束序号必须大于等于开始序号')
      return
    }
    const count = end - start + 1
    if (count > 50) {
      showToast('error', '单次最多创建 50 个仓库')
      return
    }
    setSubmitting(true)
    try {
      await batchCreate({
        prefix: prefix.trim(),
        start: Number(start),
        end: Number(end),
        description: description.trim(),
        private: priv,
        template: template.trim(),
      })
      onCreated()
    } catch (err) {
      showToast('error', err.response?.data?.error || '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  const count = Math.max(0, end - start + 1)

  return (
    <div className="panel">
      <div className="panel-title">📦 批量创建仓库</div>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>仓库名前缀</label>
          <input
            type="text"
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
            placeholder="例如: cc、proj、lab"
          />
        </div>

        <div className="form-row">
          <div className="form-group">
            <label>开始序号</label>
            <input
              type="number"
              value={start}
              onChange={(e) => setStart(Number(e.target.value))}
              min="0"
            />
          </div>
          <div className="form-group">
            <label>结束序号</label>
            <input
              type="number"
              value={end}
              onChange={(e) => setEnd(Number(e.target.value))}
              min="0"
            />
          </div>
        </div>

        <div className="form-group">
          <label>描述</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows="2"
            placeholder="仓库描述"
          />
        </div>

        <div className="form-group">
          <label>模板仓库（可选）</label>
          <input
            type="text"
            value={template}
            onChange={(e) => setTemplate(e.target.value)}
            placeholder="例如: owner/template-repo"
          />
        </div>

        <label className="form-check">
          <input
            type="checkbox"
            checked={priv}
            onChange={(e) => setPriv(e.target.checked)}
          />
          <span>设为私有仓库</span>
        </label>

        <div className="btn-group">
          <button
            type="submit"
            className="btn btn-primary"
            disabled={submitting}
          >
            {submitting ? '提交中...' : `创建 ${count} 个仓库`}
          </button>
        </div>
      </form>
    </div>
  )
}
