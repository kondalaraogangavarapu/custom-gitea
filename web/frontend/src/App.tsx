import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import NewRepo from './pages/NewRepo'
import Repository from './pages/Repository'
import BlobViewer from './pages/BlobViewer'
import Agents from './pages/Agents'
import Docs from './pages/Docs'
import Practices from './pages/Practices'
import Workflows from './pages/Workflows'
import Login from './pages/Login'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Login page lives outside the authenticated Layout */}
          <Route path="/auth/login-page" element={<Login />} />
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/new" element={<NewRepo />} />
            <Route path="/:owner/:repo" element={<Repository />} />
            <Route path="/:owner/:repo/blob/:ref/*" element={<BlobViewer />} />
            <Route path="/:owner/:repo/agents" element={<Agents />} />
            <Route path="/:owner/:repo/docs" element={<Docs />} />
            <Route path="/:owner/:repo/practices" element={<Practices />} />
            <Route path="/:owner/:repo/workflows" element={<Workflows />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
