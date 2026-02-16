import type {
  Repository,
  AgentTask,
  Document,
  TreeEntry,
  CommitInfo,
  BlobContent,
  BestPracticeReport,
  WorkflowRun,
  StepResult,
  AuthInfo,
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

// Workflows — plain-English file in the repo
export const workflows = {
  getFile: (owner: string, repo: string) =>
    request<{ path: string; content: string; exists: boolean }>(
      `/repos/${owner}/${repo}/workflows/file`
    ),

  saveFile: (owner: string, repo: string, content: string) =>
    request<{ path: string; commit_sha: string }>(
      `/repos/${owner}/${repo}/workflows/file`,
      { method: 'PUT', body: JSON.stringify({ content }) }
    ),

  listRuns: (owner: string, repo: string) =>
    request<WorkflowRun[]>(`/repos/${owner}/${repo}/workflows/runs`),

  createRun: (
    owner: string,
    repo: string,
    data: { triggered_by: string; section: string; step_results?: StepResult[] }
  ) =>
    request<WorkflowRun>(`/repos/${owner}/${repo}/workflows/runs`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getRun: (owner: string, repo: string, runId: number) =>
    request<WorkflowRun>(`/repos/${owner}/${repo}/workflows/runs/${runId}`),

  updateRun: (
    owner: string,
    repo: string,
    runId: number,
    data: { status?: string; step_results?: StepResult[]; summary?: string }
  ) =>
    request<WorkflowRun>(`/repos/${owner}/${repo}/workflows/runs/${runId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
}

// Best Practices
export const practices = {
  analyze: (owner: string, repo: string) =>
    request<BestPracticeReport>(`/repos/${owner}/${repo}/practices`),
}

// Auth
export const auth = {
  me: () => request<AuthInfo>('/auth/me'),
}

// System
export const system = {
  health: () => request<{ status: string }>('/health'),
  version: () => request<{ name: string; version: string }>('/version'),
}
