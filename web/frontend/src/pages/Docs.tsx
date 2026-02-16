import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { FileText, Loader2 } from 'lucide-react'
import { docs } from '../lib/api'

export default function Docs() {
  const { owner = '', repo = '' } = useParams()

  const { data: documents, isLoading } = useQuery({
    queryKey: ['docs', owner, repo],
    queryFn: () => docs.list(owner, repo),
    enabled: !!owner && !!repo,
  })

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <FileText className="w-6 h-6 text-accent" />
          Documents
        </h1>
        <p className="text-text-secondary text-sm mt-1">
          AI-generated documentation for{' '}
          <span className="text-text-primary">{owner}/{repo}</span>
        </p>
      </div>

      {isLoading && (
        <div className="text-center py-16 text-text-muted">
          <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
          Loading documents...
        </div>
      )}

      {documents && documents.length === 0 && (
        <div className="text-center py-16 text-text-muted">
          <FileText className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <h3 className="text-lg text-text-secondary mb-1">No documents yet</h3>
          <p className="text-sm">
            Use the AI Agent to generate documentation for this repository.
          </p>
        </div>
      )}

      {documents && documents.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {documents.map((doc) => (
            <div
              key={doc.id}
              className="bg-bg-card border border-border rounded-xl p-5"
            >
              <div className="flex items-center gap-2 mb-2">
                <FileText className="w-4 h-4 text-accent" />
                <h3 className="text-sm font-semibold">{doc.title}</h3>
                <span className="inline-flex px-2 py-0.5 rounded-full text-[10px] font-semibold bg-bg-input text-text-muted">
                  {doc.type}
                </span>
              </div>
              {doc.path && (
                <p className="text-xs text-text-muted font-mono mb-2">{doc.path}</p>
              )}
              <p className="text-sm text-text-secondary line-clamp-4">
                {doc.content.slice(0, 200)}...
              </p>
              <p className="text-xs text-text-muted mt-3">
                v{doc.version} &middot;{' '}
                {new Date(doc.updated_at).toLocaleDateString()}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
