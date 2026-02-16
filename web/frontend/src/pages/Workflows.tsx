import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Workflow,
  Loader2,
  Save,
  RotateCcw,
  FileText,
  Play,
  CheckCircle2,
  XCircle,
  Clock,
  ChevronDown,
  ChevronRight,
  User,
  AlertCircle,
} from 'lucide-react'
import { workflows } from '../lib/api'
import type { WorkflowRun, StepResult } from '../types'

const starterTemplate = `# AetherDev Workflows
# This file describes CI/CD and DevOps tasks in plain English.
# Any AetherDev agent session can read this file and execute the steps
# inside its sandbox when asked to run workflows on this repository.

## On every pull request

1. Install all project dependencies
2. Run the full test suite and report any failures
3. Run the linter and fix auto-fixable issues
4. Check for known security vulnerabilities in dependencies
5. Post a summary comment on the pull request with results

## Before merging to main

1. Ensure all tests pass on the latest commit
2. Run a build to verify the project compiles cleanly
3. Generate updated API documentation if any endpoints changed
4. Verify no secrets or credentials are present in the diff

## On deploy

1. Build the production artifacts
2. Run smoke tests against the build output
3. Tag the release with the current version from package metadata
4. Deploy to the staging environment first
5. Run integration tests against staging
6. If all checks pass, promote the build to production
`

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    pending: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/30',
    running: 'bg-blue-500/10 text-blue-400 border-blue-500/30',
    completed: 'bg-green-500/10 text-green-400 border-green-500/30',
    failed: 'bg-red-500/10 text-red-400 border-red-500/30',
    skipped: 'bg-gray-500/10 text-gray-400 border-gray-500/30',
  }
  const icons: Record<string, typeof CheckCircle2> = {
    pending: Clock,
    running: Loader2,
    completed: CheckCircle2,
    failed: XCircle,
    skipped: AlertCircle,
  }
  const Icon = icons[status] || Clock

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold border ${styles[status] || styles.pending}`}
    >
      <Icon
        className={`w-3 h-3 ${status === 'running' ? 'animate-spin' : ''}`}
      />
      {status}
    </span>
  )
}

function StepResultRow({ result }: { result: StepResult }) {
  const [expanded, setExpanded] = useState(false)
  const hasOutput = result.output && result.output.length > 0

  return (
    <div className="border-b border-border last:border-b-0">
      <button
        onClick={() => hasOutput && setExpanded(!expanded)}
        className={`w-full flex items-center gap-3 px-4 py-2.5 text-left text-sm ${hasOutput ? 'hover:bg-bg-hover cursor-pointer' : 'cursor-default'} transition-colors`}
      >
        {hasOutput ? (
          expanded ? (
            <ChevronDown className="w-3.5 h-3.5 text-text-muted shrink-0" />
          ) : (
            <ChevronRight className="w-3.5 h-3.5 text-text-muted shrink-0" />
          )
        ) : (
          <span className="w-3.5" />
        )}
        <StatusBadge status={result.status} />
        <span className="text-text-primary flex-1">{result.step}</span>
        {result.task_id ? (
          <span className="text-[10px] text-text-muted font-mono">
            task #{result.task_id}
          </span>
        ) : null}
      </button>
      {expanded && hasOutput && (
        <div className="px-4 pb-3 pl-14">
          <pre className="text-xs text-text-secondary bg-bg-secondary rounded-lg p-3 whitespace-pre-wrap font-mono leading-relaxed overflow-x-auto">
            {result.output}
          </pre>
        </div>
      )}
    </div>
  )
}

function RunCard({ run }: { run: WorkflowRun }) {
  const [expanded, setExpanded] = useState(run.status === 'running')

  const duration =
    run.started_at && run.completed_at
      ? Math.round(
          (new Date(run.completed_at).getTime() -
            new Date(run.started_at).getTime()) /
            1000
        )
      : null

  return (
    <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-5 py-3.5 hover:bg-bg-hover transition-colors text-left"
      >
        {expanded ? (
          <ChevronDown className="w-4 h-4 text-text-muted shrink-0" />
        ) : (
          <ChevronRight className="w-4 h-4 text-text-muted shrink-0" />
        )}
        <StatusBadge status={run.status} />
        <div className="flex-1 min-w-0">
          <span className="text-sm font-medium text-text-primary">
            {run.section || 'Full workflow'}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-text-muted shrink-0">
          <span className="flex items-center gap-1">
            <User className="w-3 h-3" />
            {run.triggered_by || 'agent'}
          </span>
          {duration !== null && <span>{duration}s</span>}
          <span>{new Date(run.created_at).toLocaleString()}</span>
        </div>
      </button>

      {expanded && (
        <div className="border-t border-border">
          {/* Step results */}
          {run.step_results && run.step_results.length > 0 ? (
            <div>
              {run.step_results.map((result, i) => (
                <StepResultRow key={i} result={result} />
              ))}
            </div>
          ) : (
            <div className="px-5 py-3 text-sm text-text-muted">
              No step details recorded for this run.
            </div>
          )}

          {/* Summary */}
          {run.summary && (
            <div className="px-5 py-3 border-t border-border">
              <p className="text-xs font-semibold text-text-muted uppercase tracking-wider mb-1">
                Summary
              </p>
              <p className="text-sm text-text-secondary">{run.summary}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function Workflows() {
  const { owner = '', repo = '' } = useParams()
  const queryClient = useQueryClient()

  // Workflow file
  const { data: fileData, isLoading: fileLoading } = useQuery({
    queryKey: ['workflows', owner, repo],
    queryFn: () => workflows.getFile(owner, repo),
    enabled: !!owner && !!repo,
  })

  // Run history
  const { data: runs, isLoading: runsLoading } = useQuery({
    queryKey: ['workflowRuns', owner, repo],
    queryFn: () => workflows.listRuns(owner, repo),
    enabled: !!owner && !!repo,
    refetchInterval: 10_000, // poll for live updates on running workflows
  })

  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const [saved, setSaved] = useState(false)
  const [activeTab, setActiveTab] = useState<'file' | 'runs'>('runs')

  const saveMutation = useMutation({
    mutationFn: (content: string) => workflows.saveFile(owner, repo, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows', owner, repo] })
      setEditing(false)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    },
  })

  const triggerMutation = useMutation({
    mutationFn: (section: string) =>
      workflows.createRun(owner, repo, {
        triggered_by: 'admin',
        section,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['workflowRuns', owner, repo],
      })
      setActiveTab('runs')
    },
  })

  const content = fileData?.content || ''
  const exists = fileData?.exists ?? false

  // Parse sections from workflow file for the trigger dropdown
  const sections = content
    .split('\n')
    .filter((line) => line.startsWith('## '))
    .map((line) => line.replace('## ', '').trim())

  const runningCount =
    runs?.filter((r) => r.status === 'running' || r.status === 'pending')
      .length || 0

  function startEditing() {
    setDraft(exists ? content : starterTemplate)
    setEditing(true)
    setActiveTab('file')
  }

  function cancelEditing() {
    setEditing(false)
    setDraft('')
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Workflow className="w-6 h-6 text-accent" />
            Workflows
          </h1>
          <p className="text-text-secondary text-sm mt-1">
            Plain-English CI/CD steps for{' '}
            <span className="text-text-primary">
              {owner}/{repo}
            </span>{' '}
            &mdash; readable by humans and AI agents alike
          </p>
        </div>

        <div className="flex gap-2">
          {exists && !editing && sections.length > 0 && (
            <div className="relative group">
              <button className="flex items-center gap-1.5 px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-500 transition">
                <Play className="w-4 h-4" />
                Run Workflow
              </button>
              <div className="absolute right-0 top-full mt-1 w-64 bg-bg-card border border-border rounded-lg shadow-xl opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto transition-opacity z-10">
                {sections.map((section) => (
                  <button
                    key={section}
                    onClick={() => triggerMutation.mutate(section)}
                    disabled={triggerMutation.isPending}
                    className="w-full text-left px-4 py-2.5 text-sm text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors first:rounded-t-lg last:rounded-b-lg"
                  >
                    {section}
                  </button>
                ))}
              </div>
            </div>
          )}

          {!editing && (
            <button
              onClick={startEditing}
              className="flex items-center gap-1.5 px-4 py-2 bg-accent text-white text-sm font-medium rounded-lg hover:brightness-110 transition"
            >
              <FileText className="w-4 h-4" />
              {exists ? 'Edit' : 'Create Workflow File'}
            </button>
          )}
        </div>
      </div>

      {saved && (
        <div className="mb-4 px-4 py-2.5 rounded-lg bg-green-500/10 border border-green-500/30 text-green-400 text-sm">
          Workflow file saved and committed to the repository.
        </div>
      )}

      {/* Tabs */}
      {exists && !editing && (
        <div className="flex gap-0 border-b border-border mb-5">
          <button
            onClick={() => setActiveTab('runs')}
            className={`flex items-center gap-1.5 px-5 py-2.5 text-sm border-b-2 transition-colors ${
              activeTab === 'runs'
                ? 'border-accent text-accent'
                : 'border-transparent text-text-secondary hover:text-text-primary'
            }`}
          >
            <Play className="w-4 h-4" />
            Run History
            {runningCount > 0 && (
              <span className="ml-1 px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-blue-500/20 text-blue-400">
                {runningCount}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('file')}
            className={`flex items-center gap-1.5 px-5 py-2.5 text-sm border-b-2 transition-colors ${
              activeTab === 'file'
                ? 'border-accent text-accent'
                : 'border-transparent text-text-secondary hover:text-text-primary'
            }`}
          >
            <FileText className="w-4 h-4" />
            Workflow File
          </button>
        </div>
      )}

      {(fileLoading || runsLoading) && !fileData && !runs && (
        <div className="text-center py-16 text-text-muted">
          <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
          Loading workflows...
        </div>
      )}

      {/* Editor mode */}
      {editing && (
        <div>
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs text-text-muted font-mono">
              .aetherdev/workflows.md
            </p>
            <div className="flex gap-2">
              <button
                onClick={cancelEditing}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-text-secondary border border-border rounded-lg hover:bg-bg-hover transition"
              >
                <RotateCcw className="w-3.5 h-3.5" />
                Cancel
              </button>
              <button
                onClick={() => saveMutation.mutate(draft)}
                disabled={saveMutation.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium bg-accent text-white rounded-lg hover:brightness-110 transition disabled:opacity-50"
              >
                {saveMutation.isPending ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Save className="w-3.5 h-3.5" />
                )}
                Save &amp; Commit
              </button>
            </div>
          </div>

          {saveMutation.isError && (
            <div className="mb-3 px-4 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
              {saveMutation.error?.message || 'Failed to save workflow file.'}
            </div>
          )}

          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            rows={24}
            className="w-full bg-bg-card border border-border rounded-xl p-4 font-mono text-sm text-text-primary resize-y focus:outline-none focus:ring-2 focus:ring-accent/40 leading-relaxed"
            spellCheck={false}
          />
          <p className="text-xs text-text-muted mt-2">
            Write your workflows in plain English. Each section describes when
            the steps should run and what the agent should do. No YAML or
            scripting syntax needed.
          </p>
        </div>
      )}

      {/* Run history tab */}
      {!editing && activeTab === 'runs' && exists && (
        <div>
          {runs && runs.length === 0 && (
            <div className="text-center py-16 text-text-muted">
              <Play className="w-12 h-12 mx-auto mb-3 opacity-40" />
              <h3 className="text-lg text-text-secondary mb-1">
                No workflow runs yet
              </h3>
              <p className="text-sm max-w-md mx-auto">
                When an agent executes the workflow steps in this repository,
                each run and its results will appear here.
              </p>
            </div>
          )}

          {runs && runs.length > 0 && (
            <div className="space-y-3">
              {runs.map((run) => (
                <RunCard key={run.id} run={run} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Workflow file tab (read-only) */}
      {!editing && activeTab === 'file' && exists && (
        <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
          <div className="px-4 py-2.5 bg-bg-secondary border-b border-border flex items-center gap-2">
            <FileText className="w-4 h-4 text-text-muted" />
            <span className="text-xs font-mono text-text-muted">
              .aetherdev/workflows.md
            </span>
          </div>
          <div className="p-5">
            <pre className="whitespace-pre-wrap text-sm text-text-primary font-mono leading-relaxed">
              {content}
            </pre>
          </div>
        </div>
      )}

      {/* Empty state — no workflow file */}
      {!fileLoading && !editing && !exists && (
        <div className="text-center py-16 text-text-muted">
          <Workflow className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <h3 className="text-lg text-text-secondary mb-1">
            No workflow file yet
          </h3>
          <p className="text-sm max-w-md mx-auto mb-4">
            Create a{' '}
            <span className="font-mono text-text-primary">
              .aetherdev/workflows.md
            </span>{' '}
            file to define your CI/CD steps in plain English. Any AetherDev
            agent session can read this file and execute the steps in its
            sandbox.
          </p>
          <button
            onClick={startEditing}
            className="inline-flex items-center gap-1.5 px-4 py-2 bg-accent text-white text-sm font-medium rounded-lg hover:brightness-110 transition"
          >
            <FileText className="w-4 h-4" />
            Create Workflow File
          </button>
        </div>
      )}
    </div>
  )
}
