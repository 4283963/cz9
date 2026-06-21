import axios from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL || '/api'

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
})

export const fetchRepos = (status = '') => {
  return api.get('/repos', { params: { status } })
}

export const fetchRepo = (id) => {
  return api.get(`/repos/${id}`)
}

export const batchCreate = (data) => {
  return api.post('/repos/batch', data)
}

export const retryRepo = (id, data = {}) => {
  return api.post(`/repos/${id}/retry`, data)
}

export const deleteRepo = (id) => {
  return api.delete(`/repos/${id}`)
}

export const deleteAllRepos = () => {
  return api.delete('/repos')
}

export const fetchStats = () => {
  return api.get('/stats')
}

export default api
