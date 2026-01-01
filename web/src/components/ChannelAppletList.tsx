import { useState } from 'react'
import { useRevalidator } from 'react-router-dom'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Pencil, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type AppInstanceDetail = components['schemas']['AppInstanceDetail']

interface ChannelAppletListProps {
  channelUuid: string
  applets: AppInstanceDetail[]
  onEdit: (applet: AppInstanceDetail) => void
}

interface SortableAppletItemProps {
  applet: AppInstanceDetail
  channelUuid: string
  onEdit: (applet: AppInstanceDetail) => void
  onDelete: (applet: AppInstanceDetail) => void
  deleting: boolean
}

function SortableAppletItem({ applet, onEdit, onDelete, deleting }: SortableAppletItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: applet.uuid || '' })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-3 p-3 bg-slate-800 rounded-lg border border-slate-700"
    >
      <button
        {...attributes}
        {...listeners}
        className="cursor-grab active:cursor-grabbing p-1 text-slate-500 hover:text-slate-300"
      >
        <GripVertical className="h-4 w-4" />
      </button>

      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-slate-200 truncate">
          {applet['app-id']}
        </div>
        <div className="text-xs text-slate-500">
          Position: {applet.idx ?? 0}
        </div>
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onEdit(applet)}
          className="h-8 w-8 p-0"
        >
          <Pencil className="h-4 w-4" />
          <span className="sr-only">Edit</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDelete(applet)}
          disabled={deleting}
          className="h-8 w-8 p-0 text-red-500 hover:text-red-400"
        >
          <Trash2 className="h-4 w-4" />
          <span className="sr-only">Delete</span>
        </Button>
      </div>
    </div>
  )
}

export function ChannelAppletList({ channelUuid, applets, onEdit }: ChannelAppletListProps) {
  const [items, setItems] = useState(applets)
  const [deleting, setDeleting] = useState<string | null>(null)
  const revalidator = useRevalidator()

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event

    if (!over || active.id === over.id) return

    const oldIndex = items.findIndex((item) => item.uuid === active.id)
    const newIndex = items.findIndex((item) => item.uuid === over.id)

    const newItems = arrayMove(items, oldIndex, newIndex)
    setItems(newItems)

    // Update idx values for affected items
    const updates = newItems.map((item, index) => ({
      uuid: item.uuid!,
      idx: index,
    }))

    // Send PATCH requests for items that changed position
    try {
      await Promise.all(
        updates.map((update) =>
          restClient.PATCH('/channels/{channelUUID}/applets/{appletUUID}', {
            params: {
              path: {
                channelUUID: channelUuid,
                appletUUID: update.uuid,
              },
            },
            body: { idx: update.idx },
          })
        )
      )
      revalidator.revalidate()
    } catch (error) {
      console.error('Failed to update applet order:', error)
      // Revert on error
      setItems(applets)
    }
  }

  const handleDelete = async (applet: AppInstanceDetail) => {
    if (!applet.uuid) return

    setDeleting(applet.uuid)
    try {
      await restClient.DELETE('/channels/{channelUUID}/applets/{appletUUID}', {
        params: {
          path: {
            channelUUID: channelUuid,
            appletUUID: applet.uuid,
          },
        },
      })
      revalidator.revalidate()
    } catch (error) {
      console.error('Failed to delete applet:', error)
    } finally {
      setDeleting(null)
    }
  }

  // Sync local state with props when they change
  if (JSON.stringify(items.map(i => i.uuid)) !== JSON.stringify(applets.map(a => a.uuid))) {
    setItems(applets)
  }

  if (items.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500">
        <p>No applets in this channel.</p>
        <p className="text-sm mt-2">Add an applet to get started.</p>
      </div>
    )
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragEnd={handleDragEnd}
    >
      <SortableContext
        items={items.map((item) => item.uuid || '')}
        strategy={verticalListSortingStrategy}
      >
        <div className="space-y-2">
          {items.map((applet) => (
            <SortableAppletItem
              key={applet.uuid}
              applet={applet}
              channelUuid={channelUuid}
              onEdit={onEdit}
              onDelete={handleDelete}
              deleting={deleting === applet.uuid}
            />
          ))}
        </div>
      </SortableContext>
    </DndContext>
  )
}
