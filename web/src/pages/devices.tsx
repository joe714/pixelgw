import { useState } from 'react'
import { makeLoader, useLoaderData } from "react-router-typesafe"
import { useRevalidator } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Monitor, Wifi, WifiOff, Settings, Plus } from 'lucide-react'
import { restClient } from '@/rest-client'
import { DeviceConfigModal } from '@/components/DeviceConfigModal'
import { CreateVirtualDeviceModal } from '@/components/CreateVirtualDeviceModal'
import type { components } from '@/openapi'

type DeviceSummary = components['schemas']['DeviceSummary']

export const devicesLoader = makeLoader(
  async () => await restClient.GET("/devices")
);

function formatLastSeen(timestamp: string | undefined): string {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  return `${diffDays}d ago`
}

export function DevicesList() {
  const { data } = useLoaderData<typeof devicesLoader>();
  const revalidator = useRevalidator()
  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const handleDeviceCreated = (device: DeviceSummary) => {
    // Open the virtual display in a new window
    window.open(`/display/device/${device.uuid}`, '_blank')
    // Refresh the device list
    revalidator.revalidate()
  }

  return (
    <>
      <div className="flex flex-col p-4">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold">Devices</h1>
          <Button onClick={() => setCreateModalOpen(true)} className="bg-lime-700 hover:bg-lime-600">
            <Plus className="h-4 w-4 mr-2" />
            New Virtual Device
          </Button>
        </div>

        {data && data.length > 0 ? (
          <div className="grid gap-4">
            {data.map((device) => {
              const isOnline = device.connected ?? false
              const currentIP = device["current-ip"]
              const lastIP = device["last-ip"]
              const lastDisconnect = device["last-disconnect-time"]

              return (
                <div
                  key={device.uuid}
                  className="border rounded-lg p-4 bg-card hover:border-slate-600 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-lg font-semibold text-sky-500 truncate">
                          {device.name || 'Unnamed Device'}
                        </h3>
                        {isOnline ? (
                          <span className="flex items-center gap-1 text-xs text-green-500">
                            <Wifi className="h-3 w-3" />
                            Online
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-xs text-slate-500">
                            <WifiOff className="h-3 w-3" />
                            Offline
                          </span>
                        )}
                      </div>

                      <div className="space-y-1 text-sm">
                        <div className="text-slate-400">
                          <span className="text-slate-500">ID:</span> {device.uuid}
                        </div>
                        {isOnline && currentIP && (
                          <div className="text-slate-400">
                            <span className="text-slate-500">IP:</span> {currentIP}
                          </div>
                        )}
                        {!isOnline && lastDisconnect && lastIP && (
                          <div className="text-slate-500">
                            Last seen {formatLastSeen(lastDisconnect)} at {lastIP}
                          </div>
                        )}
                        {device.channel && (
                          <div className="text-slate-400">
                            <span className="text-slate-500">Channel:</span> {device.channel.name || device.channel.uuid?.slice(0, 8)}
                          </div>
                        )}
                      </div>
                    </div>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSelectedDevice(device)
                        setModalOpen(true)
                      }}
                    >
                      <Settings className="h-4 w-4" />
                      <span className="sr-only">Configure {device.name}</span>
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          <div className="text-center py-12 text-gray-500">
            <Monitor className="h-12 w-12 mx-auto mb-4 text-gray-600" />
            <p>No devices registered</p>
            <p className="text-sm mt-2">Devices will appear here when they connect via WebSocket</p>
          </div>
        )}
      </div>

      {selectedDevice && (
        <DeviceConfigModal
          open={modalOpen}
          onOpenChange={setModalOpen}
          device={selectedDevice}
        />
      )}

      <CreateVirtualDeviceModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onCreated={handleDeviceCreated}
      />
    </>
  )
}