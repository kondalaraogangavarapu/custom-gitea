export interface User {
  id: number
  username: string
  email: string
  full_name: string
  avatar_url: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

export interface Repository {
  id: number
  owner_id: number
  name: string
  description: string
  is_private: boolean
  default_branch: string
  language: string
  stars: number
  forks: number
  created_at: string
  updated_at: string
}

export interface AgentTask {
  id: number
  repo_id: number
  user_id: number
  type: 'code_review' | 'fix' | 'generate' | 'document' | 'mindmap' | 'design'
  status: 'pending' | 'running' | 'completed' | 'failed'
  prompt: string
  result: string
  provider: 'claude' | 'aws_bedrock'
  model: string
  tokens_used: number
  branch_name: string
  commit_sha: string
  started_at: string | null
  completed_at: string | null
  created_at: string
}

export interface Document {
  id: number
  repo_id: number
  task_id: number
  title: string
  type: 'markdown' | 'design' | 'mindmap' | 'diagram' | 'runbook'
  content: string
  path: string
  version: number
  created_at: string
  updated_at: string
}

export interface TreeEntry {
  name: string
  path: string
  type: 'blob' | 'tree'
  size: number
  mode: string
}

export interface CommitInfo {
  sha: string
  message: string
  author: string
  email: string
  date: string
}

export interface BlobContent {
  content: string
  path: string
}

export interface PracticeItem {
  rule: string
  description: string
  status: 'pass' | 'warn' | 'fail'
  severity: string
  suggestion: string
  auto_fixable: boolean
}

export interface PracticeCategory {
  name: string
  score: number
  items: PracticeItem[]
}

export interface BestPracticeReport {
  id: number
  repo_id: number
  overall_score: number
  categories: PracticeCategory[]
  generated_at: string
}
