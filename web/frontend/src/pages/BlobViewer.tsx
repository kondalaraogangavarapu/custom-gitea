import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { File, Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { repos } from '../lib/api'

export default function BlobViewer() {
  const { owner = '', repo = '', '*': filePath = '' } = useParams()
  const [copied, setCopied] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ['blob', owner, repo, 'main', filePath],
    queryFn: () => repos.blob(owner, repo, 'main', filePath),
    enabled: !!owner && !!repo && !!filePath,
  })

  const lines = data?.content?.split('\n') ?? []

  const handleCopy = async () => {
    if (data?.content) {
      await navigator.clipboard.writeText(data.content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div>
      {/* File path breadcrumb */}
      <div className="flex items-center gap-2 mb-4">
        <File className="w-4 h-4 text-text-muted" />
        <span className="text-sm text-text-secondary">
          {owner}/{repo}/
          <span className="text-text-primary font-medium">{filePath}</span>
        </span>
      </div>

      {/* File content */}
      <div className="bg-bg-card border border-border rounded-xl overflow-hidden">
        {/* File header */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-bg-secondary">
          <span className="text-sm text-text-muted">{lines.length} lines</span>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 px-2 py-1 text-xs text-text-secondary hover:text-text-primary rounded transition-colors"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-success" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                Copy
              </>
            )}
          </button>
        </div>

        {isLoading && (
          <div className="p-8 text-center text-text-muted">Loading file...</div>
        )}

        {error && (
          <div className="p-8 text-center text-danger">Failed to load file.</div>
        )}

        {data && (
          <div className="overflow-x-auto">
            <pre className="text-sm font-mono leading-relaxed">
              <code>
                {lines.map((line, i) => (
                  <div key={i} className="flex hover:bg-bg-hover">
                    <span className="inline-block w-12 text-right pr-4 text-text-muted select-none shrink-0 border-r border-border">
                      {i + 1}
                    </span>
                    <span className="pl-4 pr-4">{line || '\u00A0'}</span>
                  </div>
                ))}
              </code>
            </pre>
          </div>
        )}
      </div>
    </div>
  )
}
