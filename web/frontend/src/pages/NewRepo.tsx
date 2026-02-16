import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { GitBranch, Lock, Globe } from 'lucide-react'
import { repos } from '../lib/api'

export default function NewRepo() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)

  const mutation = useMutation({
    mutationFn: () =>
      repos.create({ name, description, is_private: isPrivate }),
    onSuccess: (repo) => {
      queryClient.invalidateQueries({ queryKey: ['repos'] })
      navigate(`/user/${repo.name}`)
    },
  })

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold">Create a new repository</h1>
        <p className="text-text-secondary text-sm mt-1">
          A repository contains all project files, including the revision history.
        </p>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault()
          mutation.mutate()
        }}
        className="space-y-5"
      >
        <div>
          <label className="block text-sm font-medium text-text-secondary mb-1.5">
            Repository name <span className="text-danger">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-awesome-project"
            className="w-full px-3.5 py-2.5 bg-bg-input border border-border rounded-lg text-sm text-text-primary outline-none focus:border-accent transition-colors"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-text-secondary mb-1.5">
            Description
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Short description of your project"
            rows={3}
            className="w-full px-3.5 py-2.5 bg-bg-input border border-border rounded-lg text-sm text-text-primary outline-none focus:border-accent transition-colors resize-y font-[inherit]"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-text-secondary mb-3">
            Visibility
          </label>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setIsPrivate(false)}
              className={`flex-1 flex items-center gap-3 p-4 rounded-xl border transition-colors ${
                !isPrivate
                  ? 'border-accent bg-accent-light'
                  : 'border-border bg-bg-card hover:bg-bg-hover'
              }`}
            >
              <Globe
                className={`w-5 h-5 ${!isPrivate ? 'text-accent' : 'text-text-muted'}`}
              />
              <div className="text-left">
                <p className="text-sm font-medium">Public</p>
                <p className="text-xs text-text-muted">Anyone can see this repository</p>
              </div>
            </button>
            <button
              type="button"
              onClick={() => setIsPrivate(true)}
              className={`flex-1 flex items-center gap-3 p-4 rounded-xl border transition-colors ${
                isPrivate
                  ? 'border-accent bg-accent-light'
                  : 'border-border bg-bg-card hover:bg-bg-hover'
              }`}
            >
              <Lock
                className={`w-5 h-5 ${isPrivate ? 'text-accent' : 'text-text-muted'}`}
              />
              <div className="text-left">
                <p className="text-sm font-medium">Private</p>
                <p className="text-xs text-text-muted">Only you can access this repository</p>
              </div>
            </button>
          </div>
        </div>

        {mutation.error && (
          <div className="p-3 bg-danger/10 border border-danger/30 rounded-lg text-sm text-danger">
            {mutation.error.message}
          </div>
        )}

        <div className="flex items-center gap-3 pt-4 border-t border-border">
          <button
            type="submit"
            disabled={!name || mutation.isPending}
            className="inline-flex items-center gap-1.5 px-5 py-2.5 bg-accent text-white rounded-lg text-sm font-medium hover:bg-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <GitBranch className="w-4 h-4" />
            {mutation.isPending ? 'Creating...' : 'Create Repository'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/')}
            className="px-5 py-2.5 bg-bg-input border border-border text-text-primary rounded-lg text-sm hover:bg-bg-hover transition-colors"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
