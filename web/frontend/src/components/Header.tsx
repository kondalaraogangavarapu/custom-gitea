import { Search, Bell, User, LogOut } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { auth } from '../lib/api'

interface HeaderProps {
  title?: string
}

export default function Header({ title }: HeaderProps) {
  const { data } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: auth.me,
    staleTime: 60_000,
    retry: false,
  })

  const user = data?.user

  return (
    <header className="h-14 bg-bg-secondary border-b border-border flex items-center justify-between px-6 sticky top-0 z-40">
      {title && <h2 className="text-base font-semibold">{title}</h2>}

      {/* Search */}
      <div className="flex items-center gap-2 bg-bg-input border border-border rounded-lg px-3 py-1.5 w-[400px]">
        <Search className="w-4 h-4 text-text-muted shrink-0" />
        <input
          type="text"
          placeholder="Search repositories, files, agents..."
          className="bg-transparent border-none text-sm text-text-primary w-full outline-none placeholder:text-text-muted"
        />
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button className="p-2 rounded-lg text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors">
          <Bell className="w-[18px] h-[18px]" />
        </button>

        {user ? (
          <div className="flex items-center gap-2">
            {user.avatar_url ? (
              <img
                src={user.avatar_url}
                alt={user.full_name || user.username}
                className="w-8 h-8 rounded-full"
              />
            ) : (
              <div className="w-8 h-8 rounded-full bg-accent flex items-center justify-center text-white text-xs font-semibold">
                {(user.full_name || user.username || '?')[0].toUpperCase()}
              </div>
            )}
            <span className="text-sm text-text-secondary hidden lg:block">
              {user.full_name || user.username}
            </span>
            <a
              href="/auth/logout"
              title="Sign out"
              className="p-1.5 rounded-lg text-text-muted hover:bg-bg-hover hover:text-text-primary transition-colors"
            >
              <LogOut className="w-4 h-4" />
            </a>
          </div>
        ) : (
          <div className="w-8 h-8 rounded-full bg-accent flex items-center justify-center">
            <User className="w-4 h-4 text-white" />
          </div>
        )}
      </div>
    </header>
  )
}
