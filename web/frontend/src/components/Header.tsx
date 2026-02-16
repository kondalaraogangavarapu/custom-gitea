import { Search, Bell, User } from 'lucide-react'

interface HeaderProps {
  title?: string
}

export default function Header({ title }: HeaderProps) {
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
        <div className="w-8 h-8 rounded-full bg-accent flex items-center justify-center">
          <User className="w-4 h-4 text-white" />
        </div>
      </div>
    </header>
  )
}
