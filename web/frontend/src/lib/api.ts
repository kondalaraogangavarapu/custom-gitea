import type {
  Repository,
  AgentTask,
  Document,
  TreeEntry,
  CommitInfo,
  BlobContent,
  BestPracticeReport,
} from '../types'

const BASE = '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }
  return res.json()
}

// Repositories
export const repos = {
  list: () => request<Repository[]>('/repos'),

  get: (owner: string, repo: string) =>
    request<Repository>(`/repos/${owner}/${repo}`),

  create: (data: { name: string; description: string; is_private: boolean }) =>
    request<Repository>('/repos', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  branches: (owner: string, repo: string) =>
    request<string[]>(`/repos/${owner}/${repo}/branches`),

  commits: (owner: string, repo: string, ref: string) =>
    request<CommitInfo[]>(`/repos/${owner}/${repo}/commits/${ref}`),

  tree: (owner: string, repo: string, ref: string, path = '') =>
    request<TreeEntry[]>(`/repos/${owner}/${repo}/tree/${ref}/${path}`),

  blob: (owner: string, repo: string, ref: string, path: string) =>
    request<BlobContent>(`/repos/${owner}/${repo}/blob/${ref}/${path}`),
}

// Agent Tasks
export const agents = {
  createTask: (
    owner: string,
    repo: string,
    data: { type: string; prompt: string; provider?: string }
  ) =>
    request<AgentTask>(`/repos/${owner}/${repo}/agent/tasks`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  listTasks: (owner: string, repo: string) =>
    request<AgentTask[]>(`/repos/${owner}/${repo}/agent/tasks`),
}

// Documents
export const docs = {
  list: (owner: string, repo: string) =>
    request<Document[]>(`/repos/${owner}/${repo}/documents`),
}

// Best Practices
export const practices = {
  analyze: (owner: string, repo: string) =>
    request<BestPracticeReport>(`/repos/${owner}/${repo}/practices`),
}

// System
export const system = {
  health: () => request<{ status: string }>('/health'),
  version: () => request<{ name: string; version: string }>('/version'),
}
