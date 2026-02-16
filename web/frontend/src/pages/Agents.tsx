import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Bot,
  Send,
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
  Zap,
} from 'lucide-react'
import { agents } from '../lib/api'
import type { AgentTask } from '../types'

const taskTypes = [
  { value: 'code_review', label: 'Code Review' },
  { value: 'fix', label: 'Bug Fix' },
  { value: 'generate', label: 'Generate Code' },
  { value: 'document', label: 'Documentation' },
  { value: 'design', label: 'Design' },
]

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-4 h-4 text-success" />
    case 'failed':
      return <XCircle className="w-4 h-4 text-danger" />
    case 'running':
      return <Loader2 className="w-4 h-4 text-info animate-spin" />
    default:
      return <Clock className="w-4 h-4 text-warning" />
  }
}

function TaskCard({ task }: { task: AgentTask }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div
      className="bg-bg-card border border-border rounded-xl p-4 cursor-pointer transition-colors hover:border-border-light"
      onClick={() => setExpanded(!expanded)}
    >
      <div className="flex items-start gap-3">
        <StatusIcon status={task.status} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="inline-flex px-2 py-0.5 rounded-full text-[11px] font-semibold bg-accent-light text-accent">
              {task.type}
            </span>
            <span className="text-xs text-text-muted">
              {task.provider} / {task.model}
            </span>
          </div>
          <p className="text-sm mt-1.5 text-text-secondary line-clamp-2">
            {task.prompt}
          </p>
          {task.tokens_used > 0 && (
            <span className="text-xs text-text-muted mt-1.5 inline-block">
              {task.tokens_used.toLocaleString()} tokens
            </span>
          )}
        </div>
      </div>

      {expanded && task.result && (
        <div className="mt-4 pt-4 border-t border-border">
          <pre className="text-sm font-mono text-text-secondary whitespace-pre-wrap bg-bg-input rounded-lg p-3 max-h-96 overflow-y-auto">
            {task.result}
          </pre>
        </div>
      )}
    </div>
  )
}

export default function Agents() {
  const { owner = '', repo = '' } = useParams()
  const queryClient = useQueryClient()
  const [prompt, setPrompt] = useState('')
  const [taskType, setTaskType] = useState('code_review')

  const { data: tasks, isLoading } = useQuery({
    queryKey: ['tasks', owner, repo],
    queryFn: () => agents.listTasks(owner, repo),
    enabled: !!owner && !!repo,
  })

  const mutation = useMutation({
    mutationFn: () =>
      agents.createTask(owner, repo, { type: taskType, prompt }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks', owner, repo] })
      setPrompt('')
    },
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bot className="w-6 h-6 text-accent" />
            AI Agents
          </h1>
          <p className="text-text-secondary text-sm mt-1">
            Use AI to review, fix, generate, and document your code
          </p>
        </div>
      </div>

      {/* New task form */}
      <div className="bg-bg-card border border-border rounded-xl overflow-hidden mb-6">
        <div className="px-4 py-3 border-b border-border bg-bg-secondary flex items-center gap-2">
          <Zap className="w-4 h-4 text-accent" />
          <span className="text-sm font-medium">New Agent Task</span>
        </div>

        <div className="p-4 space-y-3">
          <div className="flex gap-2 flex-wrap">
            {taskTypes.map((t) => (
              <button
                key={t.value}
                onClick={() => setTaskType(t.value)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                  taskType === t.value
                    ? 'bg-accent text-white'
                    : 'bg-bg-input text-text-secondary border border-border hover:bg-bg-hover'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          <div className="flex gap-2">
            <input
              type="text"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && prompt.trim()) mutation.mutate()
              }}
              placeholder="Describe what you want the agent to do..."
              className="flex-1 px-3.5 py-2.5 bg-bg-input border border-border rounded-lg text-sm text-text-primary outline-none focus:border-accent transition-colors"
            />
            <button
              onClick={() => mutation.mutate()}
              disabled={!prompt.trim() || mutation.isPending}
              className="inline-flex items-center gap-1.5 px-4 py-2.5 bg-accent text-white rounded-lg text-sm font-medium hover:bg-accent-hover transition-colors disabled:opacity-50"
            >
              {mutation.isPending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Send className="w-4 h-4" />
              )}
              Send
            </button>
          </div>

          {mutation.error && (
            <p className="text-sm text-danger">{mutation.error.message}</p>
          )}
        </div>
      </div>

      {/* Task list */}
      {isLoading && (
        <div className="text-center py-12 text-text-muted">
          <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
          Loading tasks...
        </div>
      )}

      {tasks && tasks.length === 0 && (
        <div className="text-center py-16 text-text-muted">
          <Bot className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <h3 className="text-lg text-text-secondary mb-1">No tasks yet</h3>
          <p className="text-sm">Create your first AI agent task above.</p>
        </div>
      )}

      {tasks && tasks.length > 0 && (
        <div className="space-y-3">
          {tasks.map((task) => (
            <TaskCard key={task.id} task={task} />
          ))}
        </div>
      )}
    </div>
  )
}
