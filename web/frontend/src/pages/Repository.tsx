import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Folder,
  File,
  GitBranch,
  Clock,
  Bot,
  FileText,
  ShieldCheck,
  Workflow,
} from 'lucide-react'
import { repos } from '../lib/api'
import type { TreeEntry } from '../types'

function FileIcon({ entry }: { entry: TreeEntry }) {
  if (entry.type === 'tree') {
    return <Folder className="w-4 h-4 text-accent shrink-0" />
  }
  return <File className="w-4 h-4 text-text-muted shrink-0" />
}

export default function Repository() {
  const { owner = '', repo = '' } = useParams()

  const { data: repoInfo } = useQuery({
    queryKey: ['repo', owner, repo],
    queryFn: () => repos.get(owner, repo),
    enabled: !!owner && !!repo,
  })

  const { data: tree, isLoading: treeLoading } = useQuery({
    queryKey: ['tree', owner, repo, 'main', ''],
    queryFn: () => repos.tree(owner, repo, 'main'),
    enabled: !!owner && !!repo,
  })

  const { data: commits } = useQuery({
    queryKey: ['commits', owner, repo, 'main'],
    queryFn: () => repos.commits(owner, repo, 'main'),
    enabled: !!owner && !!repo,
  })

  const latestCommit = commits?.[0]

  return (
    <div>
      {/* Repo header */}
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-1">
          <GitBranch className="w-5 h-5 text-accent" />
          <h1 className="text-2xl font-bold">
            <Link to={`/${owner}`} className="text-text-secondary hover:text-accent">
              {owner}
            </Link>
            <span className="text-text-muted mx-1">/</span>
            <span className="text-text-primary">{repo}</span>
          </h1>
        </div>
        {repoInfo?.description && (
          <p className="text-text-secondary text-sm mt-1">{repoInfo.description}</p>
        )}
      </div>

      {/* Quick nav tabs */}
      <div className="flex gap-0 border-b border-border mb-5">
        {[
          { to: `/${owner}/${repo}`, label: 'Code', icon: GitBranch, end: true },
          { to: `/${owner}/${repo}/agents`, label: 'AI Agents', icon: Bot },
          { to: `/${owner}/${repo}/docs`, label: 'Docs', icon: FileText },
          { to: `/${owner}/${repo}/practices`, label: 'Practices', icon: ShieldCheck },
          { to: `/${owner}/${repo}/workflows`, label: 'Workflows', icon: Workflow },
        ].map((tab) => (
          <Link
            key={tab.to}
            to={tab.to}
            className="flex items-center gap-1.5 px-5 py-2.5 text-sm text-text-secondary border-b-2 border-transparent hover:text-text-primary transition-colors first:border-accent first:text-accent"
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </Link>
        ))}
      </div>

      {/* Branch selector + latest commit */}
      <div className="flex items-center gap-4 mb-4">
        <div className="flex items-center gap-1.5 px-3 py-1.5 bg-bg-input border border-border rounded-lg text-sm">
          <GitBranch className="w-3.5 h-3.5 text-text-muted" />
          <span className="font-medium">main</span>
        </div>

        {latestCommit && (
          <div className="flex items-center gap-2 text-sm text-text-muted truncate">
            <Clock className="w-3.5 h-3.5 shrink-0" />
            <span className="text-text-secondary font-medium truncate">
              {latestCommit.message}
            </span>
            <span className="shrink-0">
              {latestCommit.author} committed{' '}
              {new Date(latestCommit.date).toLocaleDateString()}
            </span>
          </div>
        )}
      </div>

      {/* File tree */}
      <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
        {treeLoading && (
          <div className="p-8 text-center text-text-muted">
            <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin mx-auto mb-2" />
            Loading files...
          </div>
        )}

        {tree && tree.length === 0 && (
          <div className="p-12 text-center text-text-muted">
            <Folder className="w-12 h-12 mx-auto mb-3 opacity-40" />
            <p className="text-sm">This repository is empty</p>
          </div>
        )}

        {tree &&
          tree.length > 0 &&
          [...tree]
            .sort((a, b) => {
              if (a.type !== b.type) return a.type === 'tree' ? -1 : 1
              return a.name.localeCompare(b.name)
            })
            .map((entry) => (
              <Link
                key={entry.name}
                to={
                  entry.type === 'tree'
                    ? `/${owner}/${repo}/tree/main/${entry.path || entry.name}`
                    : `/${owner}/${repo}/blob/main/${entry.path || entry.name}`
                }
                className="flex items-center gap-2 px-4 py-2 border-b border-border hover:bg-bg-hover transition-colors last:border-b-0"
              >
                <FileIcon entry={entry} />
                <span className="text-sm flex-1">{entry.name}</span>
                {entry.size > 0 && (
                  <span className="text-xs text-text-muted">
                    {entry.size > 1024
                      ? `${(entry.size / 1024).toFixed(1)} KB`
                      : `${entry.size} B`}
                  </span>
                )}
              </Link>
            ))}
      </div>
    </div>
  )
}
