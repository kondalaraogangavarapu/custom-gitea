import { Outlet, useParams } from 'react-router-dom'
import Sidebar from './Sidebar'
import Header from './Header'

export default function Layout() {
  const { owner, repo } = useParams()

  return (
    <div className="flex min-h-screen">
      <Sidebar owner={owner} repo={repo} />
      <div className="flex-1 ml-[260px] flex flex-col">
        <Header />
        <main className="flex-1 p-6 animate-fade-in">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
