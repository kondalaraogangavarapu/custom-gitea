import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  GitBranch,
  Bot,
  FileText,
  ShieldCheck,
  Workflow,
  Plus,
  Settings,
} from 'lucide-react'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/new', icon: Plus, label: 'New Repository' },
]

const repoNavItems = [
  { suffix: '', icon: GitBranch, label: 'Code' },
  { suffix: '/agents', icon: Bot, label: 'AI Agents' },
  { suffix: '/docs', icon: FileText, label: 'Documents' },
  { suffix: '/practices', icon: ShieldCheck, label: 'Best Practices' },
  { suffix: '/pipelines', icon: Workflow, label: 'Pipelines' },
]

interface SidebarProps {
  owner?: string
  repo?: string
}

export default function Sidebar({ owner, repo }: SidebarProps) {
  const repoBase = owner && repo ? `/${owner}/${repo}` : null

  return (
    <aside className="w-[260px] bg-bg-secondary border-r border-border fixed top-0 left-0 bottom-0 z-50 flex flex-col">
      {/* Logo */}
      <NavLink
        to="/"
        className="flex items-center gap-3 px-5 py-4 border-b border-border"
      >
        <div className="w-8 h-8 rounded-lg bg-accent flex items-center justify-center">
          <GitBranch className="w-5 h-5 text-white" />
        </div>
        <h1 className="text-lg font-bold bg-gradient-to-br from-accent to-[#a29bfe] bg-clip-text text-transparent">
          AetherDev
        </h1>
      </NavLink>

      {/* Navigation */}
      <nav className="flex-1 p-3 overflow-y-auto">
        <div className="mb-6">
          <p className="text-[11px] font-semibold uppercase tracking-wider text-text-muted px-2 mb-2">
            Navigation
          </p>
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end
              className={({ isActive }) =>
                `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors ${
                  isActive
                    ? 'bg-accent-light text-accent'
                    : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
                }`
              }
            >
              <item.icon className="w-[18px] h-[18px] shrink-0" />
              {item.label}
            </NavLink>
          ))}
        </div>

        {repoBase && (
          <div className="mb-6">
            <p className="text-[11px] font-semibold uppercase tracking-wider text-text-muted px-2 mb-2">
              {owner}/{repo}
            </p>
            {repoNavItems.map((item) => (
              <NavLink
                key={item.suffix}
                to={`${repoBase}${item.suffix}`}
                end={item.suffix === ''}
                className={({ isActive }) =>
                  `flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors ${
                    isActive
                      ? 'bg-accent-light text-accent'
                      : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
                  }`
                }
              >
                <item.icon className="w-[18px] h-[18px] shrink-0" />
                {item.label}
              </NavLink>
            ))}
          </div>
        )}
      </nav>

      {/* Footer */}
      <div className="p-3 border-t border-border">
        <NavLink
          to="/settings"
          className="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm text-text-secondary hover:bg-bg-hover hover:text-text-primary transition-colors"
        >
          <Settings className="w-[18px] h-[18px] shrink-0" />
          Settings
        </NavLink>
      </div>
    </aside>
  )
}
