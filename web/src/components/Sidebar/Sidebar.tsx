import { Grid, Monitor, Package, Bug } from 'lucide-react'
import { SidebarNavItem } from './SidebarNavItem'
import { Separator } from '@/components/ui/separator'

const primaryNavItems = [
  { label: 'Channels', path: '/', icon: Grid },
  { label: 'Devices', path: '/devices', icon: Monitor },
  { label: 'Applets', path: '/applets', icon: Package },
]

const secondaryNavItems = [
  { label: 'Debug', path: '/debug', icon: Bug },
]

export function Sidebar() {
  return (
    <div className="flex flex-col h-full w-64 bg-gray-900 border-r border-gray-800">
      <div className="p-4">
        <h1 className="text-xl font-bold text-white">PixelGW</h1>
      </div>
      
      <nav className="flex-1 px-4 pb-4">
        <div className="space-y-1">
          {primaryNavItems.map((item) => (
            <SidebarNavItem
              key={item.path}
              label={item.label}
              path={item.path}
              icon={item.icon}
            />
          ))}
        </div>
        
        <Separator className="my-4 bg-gray-800" />
        
        <div className="space-y-1">
          {secondaryNavItems.map((item) => (
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