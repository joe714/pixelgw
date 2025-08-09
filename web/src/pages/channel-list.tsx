import { makeLoader, useLoaderData } from "react-router-typesafe"
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

import { SquarePlus, Settings } from 'lucide-react'

import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type ChannelDetail = components['schemas']['ChannelDetail']

export const channelListLoader = makeLoader(
  async (): Promise<ChannelDetail[]> => {
    const channelsResponse = await restClient.GET("/channels")
    if (!channelsResponse.data) return []
    
    // Fetch details for each channel to get applets and subscribers
    const channelsWithDetails = await Promise.all(
      channelsResponse.data.map(async (channel) => {
        if (channel.uuid) {
          const detailResponse = await restClient.GET("/channels/{uuid}", {
            params: { path: { uuid: channel.uuid } }
          })
          return detailResponse.data as ChannelDetail || channel as ChannelDetail
        }
        return channel as ChannelDetail
      })
    )
    
    return channelsWithDetails.filter((channel): channel is ChannelDetail => channel !== undefined)
  }
);

interface ChannelConfigModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel: ChannelDetail | null
}

function ChannelConfigModal({ open, onOpenChange, channel }: ChannelConfigModalProps) {
  const [name, setName] = useState(channel?.name || '')
  const [comment, setComment] = useState(channel?.comment || '')
  
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Configure Channel</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="channel-name" className="text-right">Name</Label>
            <Input
              id="channel-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="col-span-3"
            />
          </div>
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="channel-comment" className="text-right">Description</Label>
            <Input
              id="channel-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              className="col-span-3"
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function ChannelList() {
  const data = useLoaderData<typeof channelListLoader>();
  const [configModal, setConfigModal] = useState<{ open: boolean; channel: ChannelDetail | null }>({ 
    open: false, 
    channel: null 
  })

  const renderDeviceList = (items: ChannelDetail['subscribers'], maxDisplay: number = 5) => {
    if (!items || items.length === 0) {
      return <div className="text-slate-500 italic ml-4">None</div>
    }
    
    const displayItems = items.slice(0, maxDisplay)
    const remaining = items.length - maxDisplay
    
    return (
      <div className="ml-4 space-y-1">
        {displayItems.map((item, index) => (
          <div key={item.uuid || index} className="text-slate-200">
            {item.name || 'Unknown Device'}
          </div>
        ))}
        {remaining > 0 && (
          <div className="text-slate-400 italic">
            (+{remaining} more)
          </div>
        )}
      </div>
    )
  }

  const renderAppletList = (items: ChannelDetail['applets'], maxDisplay: number = 5) => {
    if (!items || items.length === 0) {
      return <div className="text-slate-500 italic ml-4">None</div>
    }
    
    const displayItems = items.slice(0, maxDisplay)
    const remaining = items.length - maxDisplay
    
    return (
      <div className="ml-4 space-y-1">
        {displayItems.map((item, index) => (
          <div key={item.uuid || index} className="text-slate-200">
            {item['app-id'] || 'Unknown Applet'}
          </div>
        ))}
        {remaining > 0 && (
          <div className="text-slate-400 italic">
            (+{remaining} more)
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col p-4">
      <div className="flex flex-row justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Channels</h1>
        <Button variant="ghost" className="p-2 space-x-1 bg-lime-700">
          <span className="sr-only">New Channel</span>
          <SquarePlus className="h-4 w-4" />
          <Label className="font-bold">New</Label>
        </Button>
      </div>
      
      <div className="grid gap-4">
        {data.map((channel) => (
          <div key={channel.uuid} className="border rounded-lg p-6 bg-card">
            <div className="flex justify-between items-start">
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-sky-500 truncate">
                      {channel.name}
                    </h3>
                    <p className="text-slate-300 text-sm">
                      {channel.comment || 'No description'}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setConfigModal({ open: true, channel })}
                    className="ml-4 flex-shrink-0"
                  >
                    <Settings className="h-4 w-4" />
                    <span className="sr-only">Configure {channel.name}</span>
                  </Button>
                </div>
                
                <div className="space-y-4 text-sm">
                  <div>
                    <div className="font-medium text-slate-400 mb-2">Applets</div>
                    {renderAppletList(channel.applets)}
                  </div>
                  <div>
                    <div className="font-medium text-slate-400 mb-2">Devices</div>
                    {renderDeviceList(channel.subscribers)}
                  </div>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      <ChannelConfigModal
        open={configModal.open}
        onOpenChange={(open) => setConfigModal({ open, channel: configModal.channel })}
        channel={configModal.channel}
      />
    </div>
  )
}
