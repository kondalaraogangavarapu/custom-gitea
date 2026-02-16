import { GitBranch, LogIn } from 'lucide-react'

export default function Login() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary">
      <div className="w-full max-w-sm p-8">
        <div className="flex flex-col items-center mb-8">
          <div className="w-14 h-14 rounded-2xl bg-accent flex items-center justify-center mb-4">
            <GitBranch className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold bg-gradient-to-br from-accent to-[#a29bfe] bg-clip-text text-transparent">
            AetherDev
          </h1>
          <p className="text-sm text-text-muted mt-2">
            Sign in to your account
          </p>
        </div>

        <a
          href="/auth/login"
          className="flex items-center justify-center gap-2 w-full px-4 py-3 rounded-lg bg-accent text-white font-medium hover:bg-accent/90 transition-colors"
        >
          <LogIn className="w-5 h-5" />
          Sign in with SSO
        </a>

        <p className="text-xs text-text-muted text-center mt-6">
          You will be redirected to your company's identity provider.
        </p>
      </div>
    </div>
  )
}
