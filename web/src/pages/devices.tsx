import { useState } from 'react'
import { makeLoader, useLoaderData } from "react-router-typesafe"
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Monitor, Wifi, WifiOff, Plus } from 'lucide-react'
import { restClient } from '@/rest-client'
import { DeviceConfigModal } from '@/components/DeviceConfigModal'

export const devicesLoader = makeLoader(
  async () => await restClient.GET("/devices")
);

export function DevicesList() {
  const { data } = useLoaderData<typeof devicesLoader>();
  const [selectedDevice, setSelectedDevice] = useState<any>(null)
  const [modalOpen, setModalOpen] = useState(false)
  
  const devices = data?.map((device) => {
    // Devices don't have a status field in the API, we'll assume online if they appear in the list
    const isOnline = true
    
    return (
      <li key={device.uuid}>
        <Separator className="my-4" />
        <div className="flex flex-row w-full justify-between items-center">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-gray-800 rounded-lg">
              <Monitor className="h-6 w-6 text-gray-400" />
            </div>
            <div className="flex flex-col">
              <div className="font-bold text-lg text-sky-500">{device.name || device.uuid}</div>
              <div className="text-slate-300 text-sm">Device ID: {device.uuid}</div>
              <div className="flex items-center gap-2 mt-1">
                {isOnline ? (
                  <>
                    <Wifi className="h-4 w-4 text-green-500" />
                    <span className="text-xs text-green-500">Connected</span>
                  </>
                ) : (
                  <>
                    <WifiOff className="h-4 w-4 text-gray-500" />
                    <span className="text-xs text-gray-500">Offline</span>
                  </>
                )}
                {device.channel && (
                  <span className="text-xs text-slate-400">• Channel: {device.channel.name || device.channel.uuid?.slice(0, 8)}</span>
                )}
              </div>
            </div>
          </div>
          <div className="flex gap-2">
            <Button 
              variant="outline" 
              size="sm"
              onClick={() => {
                setSelectedDevice(device)
                setModalOpen(true)
              }}
            >
              Configure
            </Button>
          </div>
        </div>
      </li>
    )
  })

  return (
    <>
      <div className="flex flex-col p-4">
        <div className="flex flex-row justify-between items-center mb-4">
          <h2 className="text-2xl font-bold text-white">Devices</h2>
          <Button variant="ghost" className="p-2 space-x-1 bg-lime-700">
            <Plus className="h-4 w-4" />
            <Label className="font-bold">Add Device</Label>
          </Button>
        </div>
        <div>
          {devices && devices.length > 0 ? (
            <ul>
              {devices}
              <Separator className="my-4" />
            </ul>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <Monitor className="h-12 w-12 mx-auto mb-4 text-gray-600" />
              <p>No devices connected</p>
              <p className="text-sm mt-2">Devices will appear here when they connect via WebSocket</p>
            </div>
          )}
        </div>
      </div>
      
      {selectedDevice && (
        <DeviceConfigModal
          open={modalOpen}
          onOpenChange={setModalOpen}
          device={selectedDevice}
        />
      )}
    </>
  )
}