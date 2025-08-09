import { Link, useLocation } from 'react-router-dom'
import { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface SidebarNavItemProps {
  label: string
  path: string
  icon: LucideIcon
  badge?: number
}

export function SidebarNavItem({ label, path, icon: Icon, badge }: SidebarNavItemProps) {
  const location = useLocation()
  const isActive = location.pathname === path

  return (
    <Link
      to={path}
      className={cn(
        "flex items-center gap-3 px-3 py-2 rounded-lg transition-colors",
        "hover:bg-gray-800",
        isActive && "bg-lime-700 text-white"
      )}
    >
      <Icon className="h-5 w-5" />
      <span className="text-sm font-medium">{label}</span>
      {badge !== undefined && badge > 0 && (
        <span className="ml-auto bg-gray-700 text-xs px-2 py-0.5 rounded-full">
          {badge}
        </span>
      )}
    </Link>
  )
}