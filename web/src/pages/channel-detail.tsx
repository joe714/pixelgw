import { useState, useEffect } from 'react'
import { useParams, Link, useRevalidator, useNavigate } from 'react-router-dom'
import { makeLoader, useLoaderData } from 'react-router-typesafe'
import { ArrowLeft, Plus, RefreshCw, Pencil, Trash2, ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { PixelDisplay } from '@/components/PixelDisplay'
import { ChannelAppletList } from '@/components/ChannelAppletList'
import { AddAppletModal } from '@/components/AddAppletModal'
import { EditAppletModal } from '@/components/EditAppletModal'
import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type ChannelDetail = components['schemas']['ChannelDetail']
type AppInstanceDetail = components['schemas']['AppInstanceDetail']

interface LoaderData {
  channel: ChannelDetail | null
}

export const channelDetailLoader = makeLoader(
  async ({ params }): Promise<LoaderData> => {
    const uuid = params.uuid
    if (!uuid) return { channel: null }

    const response = await restClient.GET('/channels/{uuid}', {
      params: { path: { uuid } },
    })

    // The API can return JSON or WebP based on Accept header
    // openapi-fetch defaults to JSON, so we expect ChannelDetail
    const data = response.data as ChannelDetail | undefined
    return { channel: data || null }
  }
)

export function ChannelDetail() {
  const { channel } = useLoaderData<typeof channelDetailLoader>()
  const { uuid } = useParams()
  const revalidator = useRevalidator()
  const navigate = useNavigate()
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [editModal, setEditModal] = useState<{ open: boolean; applet: AppInstanceDetail | null }>({
    open: false,
    applet: null,
  })
  const [editChannelModal, setEditChannelModal] = useState(false)
  const [channelName, setChannelName] = useState(channel?.name || '')
  const [channelComment, setChannelComment] = useState(channel?.comment || '')
  const [saving, setSaving] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [previewKey, setPreviewKey] = useState(0)

  // Update preview URL when channel changes
  useEffect(() => {
    if (uuid) {
      setPreviewUrl(`/api/channels/${uuid}`)
      setPreviewKey((k) => k + 1)
    }
  }, [uuid])

  // Sync channel name/comment when channel data changes
  useEffect(() => {
    if (channel) {
      setChannelName(channel.name || '')
      setChannelComment(channel.comment || '')
    }
  }, [channel])

  const handleRefreshPreview = () => {
    setPreviewKey((k) => k + 1)
  }

  const handleEditApplet = (applet: AppInstanceDetail) => {
    setEditModal({ open: true, applet })
  }

  const handleOpenEditChannel = () => {
    setChannelName(channel?.name || '')
    setChannelComment(channel?.comment || '')
    setEditChannelModal(true)
  }

  const handleSaveChannel = async () => {
    if (!uuid) return

    setSaving(true)
    try {
      await restClient.PATCH('/channels/{uuid}', {
        params: { path: { uuid } },
        body: {
          name: channelName,
          comment: channelComment,
        },
      })
      revalidator.revalidate()
      setEditChannelModal(false)
    } catch (error) {
      console.error('Failed to update channel:', error)
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteChannel = async () => {
    if (!uuid) return

    setDeleting(true)
    try {
      await restClient.DELETE('/channels/{uuid}', {
        params: { path: { uuid } },
      })
      setShowDeleteConfirm(false)
      setEditChannelModal(false)
      navigate('/')
    } catch (error) {
      console.error('Failed to delete channel:', error)
    } finally {
      setDeleting(false)
    }
  }

  const hasSubscribers = (channel?.subscribers?.length ?? 0) > 0

  if (!channel) {
    return (
      <div className="flex flex-col p-4">
        <div className="text-center py-8">
          <p className="text-slate-500">Channel not found.</p>
          <Link to="/" className="text-sky-500 hover:underline mt-2 inline-block">
            Back to channels
          </Link>
        </div>
      </div>
    )
  }

  const sortedApplets = [...(channel.applets || [])].sort((a, b) => (a.idx ?? 0) - (b.idx ?? 0))
  const nextIdx = sortedApplets.length > 0 ? (sortedApplets[sortedApplets.length - 1].idx ?? 0) + 1 : 0

  return (
    <div className="flex flex-col p-4 h-full">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Link to="/">
          <Button variant="ghost" size="sm" className="p-2">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold text-sky-500">{channel.name}</h1>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleOpenEditChannel}
              className="p-1 h-8 w-8"
            >
              <Pencil className="h-4 w-4" />
              <span className="sr-only">Edit channel</span>
            </Button>
          </div>
          {channel.comment && (
            <p className="text-sm text-slate-400">{channel.comment}</p>
          )}
        </div>
      </div>

      {/* Main content - side by side */}
      <div className="flex-1 flex gap-6 min-h-0">
        {/* Left: Preview */}
        <div className="flex-shrink-0 flex flex-col">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-slate-400">Live Preview</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleRefreshPreview}
              className="p-1 h-6 w-6"
            >
              <RefreshCw className="h-3 w-3" />
            </Button>
          </div>
          <PixelDisplay
            key={previewKey}
            src={previewUrl ? previewUrl + '?t=' + previewKey : ''}
            scale={4}
            showFrame={true}
            frameTheme="black"
            loading={!previewUrl}
          />
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={() => {
              window.open(`/display/channel/${uuid}`, '_blank')
            }}
          >
            <ExternalLink className="h-4 w-4 mr-2" />
            Open Virtual Display
          </Button>
          <div className="mt-4 text-xs text-slate-500">
            <div>Devices: {channel.subscribers?.length || 0}</div>
            <div>Applets: {channel.applets?.length || 0}</div>
          </div>
        </div>

        {/* Right: Applet list */}
        <div className="flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Applets</h2>
            <Button
              onClick={() => setAddModalOpen(true)}
              size="sm"
              className="bg-lime-700 hover:bg-lime-600"
            >
              <Plus className="h-4 w-4 mr-1" />
              Add Applet
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto pr-2">
            <ChannelAppletList
              channelUuid={uuid || ''}
              applets={sortedApplets}
              onEdit={handleEditApplet}
            />
          </div>
        </div>
      </div>

      {/* Add Applet Modal */}
      <AddAppletModal
        open={addModalOpen}
        onOpenChange={setAddModalOpen}
        channelUuid={uuid || ''}
        nextIdx={nextIdx}
      />

      {/* Edit Applet Modal */}
      <EditAppletModal
        open={editModal.open}
        onOpenChange={(open) => setEditModal({ open, applet: editModal.applet })}
        channelUuid={uuid || ''}
        applet={editModal.applet}
      />

      {/* Edit Channel Modal */}
      <Dialog open={editChannelModal} onOpenChange={setEditChannelModal}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Edit Channel</DialogTitle>
            <DialogDescription>
              Update the channel name and description.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="channel-name" className="text-right">
                Name
              </Label>
              <Input
                id="channel-name"
                value={channelName}
                onChange={(e) => setChannelName(e.target.value)}
                className="col-span-3"
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="channel-comment" className="text-right">
                Description
              </Label>
              <Input
                id="channel-comment"
                value={channelComment}
                onChange={(e) => setChannelComment(e.target.value)}
                className="col-span-3"
                placeholder="Optional description"
              />
            </div>
          </div>
          <DialogFooter className="sm:justify-between">
            <Button
              variant="destructive"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={hasSubscribers}
              title={hasSubscribers ? 'Remove all devices from this channel before deleting' : 'Delete this channel'}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete Channel
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setEditChannelModal(false)}>
                Cancel
              </Button>
              <Button onClick={handleSaveChannel} disabled={saving}>
                {saving ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Channel Confirmation */}
      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Channel</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{channel?.name}"? This will also remove all applets configured for this channel. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <Button
              variant="destructive"
              onClick={handleDeleteChannel}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
