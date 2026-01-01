import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { makeLoader, useLoaderData } from 'react-router-typesafe'
import { ArrowLeft, Plus, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
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
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [editModal, setEditModal] = useState<{ open: boolean; applet: AppInstanceDetail | null }>({
    open: false,
    applet: null,
  })
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [previewKey, setPreviewKey] = useState(0)

  // Update preview URL when channel changes
  useEffect(() => {
    if (uuid) {
      setPreviewUrl(`/api/channels/${uuid}`)
      setPreviewKey((k) => k + 1)
    }
  }, [uuid])

  const handleRefreshPreview = () => {
    setPreviewKey((k) => k + 1)
  }

  const handleEditApplet = (applet: AppInstanceDetail) => {
    setEditModal({ open: true, applet })
  }

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
          <h1 className="text-2xl font-bold text-sky-500">{channel.name}</h1>
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
    </div>
  )
}
