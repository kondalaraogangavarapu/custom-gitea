import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ShieldCheck,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Loader2,
  Wrench,
} from 'lucide-react'
import { practices } from '../lib/api'
import type { PracticeItem } from '../types'

function ScoreBadge({ score }: { score: number }) {
  const color =
    score >= 80 ? 'text-success bg-success/15' :
    score >= 50 ? 'text-warning bg-warning/15' :
    'text-danger bg-danger/15'

  return (
    <div
      className={`inline-flex items-center justify-center w-12 h-12 rounded-full font-bold text-base ${color}`}
    >
      {score}
    </div>
  )
}

function StatusIcon({ status }: { status: PracticeItem['status'] }) {
  switch (status) {
    case 'pass':
      return <CheckCircle className="w-4 h-4 text-success shrink-0" />
    case 'warn':
      return <AlertTriangle className="w-4 h-4 text-warning shrink-0" />
    case 'fail':
      return <XCircle className="w-4 h-4 text-danger shrink-0" />
  }
}

export default function Practices() {
  const { owner = '', repo = '' } = useParams()

  const { data: report, isLoading, error } = useQuery({
    queryKey: ['practices', owner, repo],
    queryFn: () => practices.analyze(owner, repo),
    enabled: !!owner && !!repo,
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-accent" />
            Best Practices
          </h1>
          <p className="text-text-secondary text-sm mt-1">
            Code quality and best practice analysis for{' '}
            <span className="text-text-primary">{owner}/{repo}</span>
          </p>
        </div>
        {report && <ScoreBadge score={report.overall_score} />}
      </div>

      {isLoading && (
        <div className="text-center py-16 text-text-muted">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-3" />
          Analyzing repository...
        </div>
      )}

      {error && (
        <div className="text-center py-16 text-danger">
          Failed to analyze repository.
        </div>
      )}

      {report && (
        <div className="space-y-6">
          {report.categories.map((category) => (
            <div
              key={category.name}
              className="bg-bg-card border border-border rounded-xl overflow-hidden"
            >
              <div className="flex items-center justify-between px-5 py-3 border-b border-border bg-bg-secondary">
                <h3 className="text-sm font-semibold uppercase tracking-wide">
                  {category.name}
                </h3>
                <ScoreBadge score={category.score} />
              </div>

              <div className="divide-y divide-border">
                {category.items.map((item, i) => (
                  <div key={i} className="flex items-start gap-3 px-5 py-3">
                    <StatusIcon status={item.status} />
                    <div className="flex-1">
                      <p className="text-sm font-medium">{item.rule}</p>
                      {item.suggestion && (
                        <p className="text-xs text-text-muted mt-0.5">
                          {item.suggestion}
                        </p>
                      )}
                    </div>
                    {item.auto_fixable && (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-accent-light text-accent">
                        <Wrench className="w-3 h-3" />
                        Auto-fixable
                      </span>
                    )}
                    <span
                      className={`text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded ${
                        item.severity === 'critical' || item.severity === 'high'
                          ? 'bg-danger/15 text-danger'
                          : item.severity === 'medium'
                          ? 'bg-warning/15 text-warning'
                          : 'bg-bg-input text-text-muted'
                      }`}
                    >
                      {item.severity}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
