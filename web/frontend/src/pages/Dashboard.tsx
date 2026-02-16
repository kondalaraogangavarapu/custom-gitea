import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { GitBranch, Star, GitFork, Lock, Plus, Globe } from 'lucide-react'
import { repos } from '../lib/api'
import type { Repository } from '../types'

function RepoCard({ repo }: { repo: Repository }) {
  return (
    <Link
      to={`/user/${repo.name}`}
      className="block bg-bg-card border border-border rounded-xl p-5 transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/40"
    >
      <div className="flex items-center gap-2 text-lg font-semibold">
        <GitBranch className="w-5 h-5 text-accent" />
        <span className="text-text-primary">{repo.name}</span>
        {repo.is_private ? (
          <Lock className="w-3.5 h-3.5 text-text-muted" />
        ) : (
          <Globe className="w-3.5 h-3.5 text-text-muted" />
        )}
      </div>

      {repo.description && (
        <p className="text-text-secondary text-[13px] mt-2 line-clamp-2">
          {repo.description}
        </p>
      )}

      <div className="flex gap-4 mt-3 text-xs text-text-muted">
        {repo.language && (
          <span className="flex items-center gap-1">
            <span className="w-2.5 h-2.5 rounded-full bg-accent" />
            {repo.language}
          </span>
        )}
        <span className="flex items-center gap-1">
          <Star className="w-3.5 h-3.5" />
          {repo.stars}
        </span>
        <span className="flex items-center gap-1">
          <GitFork className="w-3.5 h-3.5" />
          {repo.forks}
        </span>
      </div>
    </Link>
  )
}

export default function Dashboard() {
  const { data: repoList, isLoading, error } = useQuery({
    queryKey: ['repos'],
    queryFn: repos.list,
  })

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-text-secondary text-sm mt-1">
            Your repositories and recent activity
          </p>
        </div>
        <Link
          to="/new"
          className="inline-flex items-center gap-1.5 px-4 py-2 bg-accent text-white rounded-lg text-sm font-medium hover:bg-accent-hover transition-colors"
        >
          <Plus className="w-4 h-4" />
          New Repository
        </Link>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        <div className="bg-bg-card border border-border rounded-xl p-5">
          <p className="text-text-muted text-xs font-semibold uppercase tracking-wide">
            Repositories
          </p>
          <p className="text-3xl font-bold mt-1">{repoList?.length ?? 0}</p>
        </div>
        <div className="bg-bg-card border border-border rounded-xl p-5">
          <p className="text-text-muted text-xs font-semibold uppercase tracking-wide">
            AI Tasks
          </p>
          <p className="text-3xl font-bold mt-1 text-accent">--</p>
        </div>
        <div className="bg-bg-card border border-border rounded-xl p-5">
          <p className="text-text-muted text-xs font-semibold uppercase tracking-wide">
            Practice Score
          </p>
          <p className="text-3xl font-bold mt-1 text-success">--</p>
        </div>
      </div>

      {/* Repos grid */}
      {isLoading && (
        <div className="text-center py-16 text-text-muted">
          <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          Loading repositories...
        </div>
      )}

      {error && (
        <div className="text-center py-16 text-danger">
          Failed to load repositories. Is the backend running?
        </div>
      )}

      {repoList && repoList.length === 0 && (
        <div className="text-center py-16 text-text-muted">
          <GitBranch className="w-16 h-16 mx-auto mb-4 opacity-50" />
          <h3 className="text-lg text-text-secondary mb-2">No repositories yet</h3>
          <p className="text-sm mb-5">Create your first repository to get started.</p>
          <Link
            to="/new"
            className="inline-flex items-center gap-1.5 px-4 py-2 bg-accent text-white rounded-lg text-sm font-medium hover:bg-accent-hover transition-colors"
          >
            <Plus className="w-4 h-4" />
            Create Repository
          </Link>
        </div>
      )}

      {repoList && repoList.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
          {repoList.map((repo) => (
            <RepoCard key={repo.id} repo={repo} />
          ))}
        </div>
      )}
    </div>
  )
}
