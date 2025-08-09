import { Outlet } from 'react-router-dom'
import { Sidebar } from '@/components/Sidebar'

function App() {
  return (
    <div className="flex w-screen h-screen overflow-hidden bg-gray-950">
      <Sidebar />
      <div className="flex-1 overflow-auto">
        <Outlet />
      </div>
    </div>
  )
}

export default App
