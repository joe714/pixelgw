import { Grid, Monitor } from 'lucide-react'
import { SidebarNavItem } from './SidebarNavItem'

const navItems = [
  { label: 'Channels', path: '/', icon: Grid },
  { label: 'Devices', path: '/devices', icon: Monitor },
]

export function Sidebar() {
  return (
    <div className="flex flex-col h-full w-64 bg-gray-900 border-r border-gray-800">
      <div className="p-4">
        <h1 className="text-xl font-bold text-white">PixelGW</h1>
      </div>

      <nav className="flex-1 px-4 pb-4">
        <div className="space-y-1">
          {navItems.map((item) => (
            <SidebarNavItem
              key={item.path}
              label={item.label}
              path={item.path}
              icon={item.icon}
            />
          ))}
        </div>
      </nav>
    </div>
  )
}