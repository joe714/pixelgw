import { makeLoader, useLoaderData } from "react-router-typesafe"
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'

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

export function ChannelList() {
  const data = useLoaderData<typeof channelListLoader>();

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
          <div key={channel.uuid} className="border rounded-lg p-6 bg-card hover:border-slate-600 transition-colors">
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
                  <div className="flex items-center gap-2 ml-4 flex-shrink-0">
                    <Link to={`/channels/${channel.uuid}`}>
                      <Button variant="ghost" size="sm">
                        <Settings className="h-4 w-4" />
                        <span className="sr-only">Manage {channel.name}</span>
                      </Button>
                    </Link>
                  </div>
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
    </div>
  )
}
