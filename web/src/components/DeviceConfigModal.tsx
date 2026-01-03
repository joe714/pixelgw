import { useState, useEffect } from 'react'
import { useRevalidator } from 'react-router-dom'
import { Copy, Trash2, Eye, ChevronDown, ChevronRight, RefreshCw, Star } from 'lucide-react'
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
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { restClient } from '@/rest-client'

type FirmwareSummary = {
  uuid: string
  platform: string
  filename: string
  description?: string
  version: string
  'build-timestamp': string
  'elf-sha256': string
  'idf-version': string
  'file-size': number
  'is-default': boolean
  'uploaded-at': string
}

interface DeviceConfigModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  device: {
    uuid?: string
    name?: string
    connected?: boolean
    'current-ip'?: string
    'last-ip'?: string
    channel?: {
      uuid?: string
      name?: string
    }
  }
}

export function DeviceConfigModal({ open, onOpenChange, device }: DeviceConfigModalProps) {
  const [name, setName] = useState(device.name || '')
  const [selectedChannelUuid, setSelectedChannelUuid] = useState(device.channel?.uuid || '')
  const [channels, setChannels] = useState<Array<{ uuid?: string; name?: string }>>([])
  const [loading, setLoading] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [identifying, setIdentifying] = useState(false)
  const [deviceInfo, setDeviceInfo] = useState<Record<string, unknown> | null>(null)
  const [deviceInfoUpdated, setDeviceInfoUpdated] = useState<string | null>(null)
  const [showDeviceInfo, setShowDeviceInfo] = useState(false)
  const [showFirmwareUpdate, setShowFirmwareUpdate] = useState(false)
  const [firmwares, setFirmwares] = useState<FirmwareSummary[]>([])
  const [selectedFirmwareUuid, setSelectedFirmwareUuid] = useState<string>('')
  const [updating, setUpdating] = useState(false)
  const revalidator = useRevalidator()

  useEffect(() => {
    async function fetchData() {
      // Fetch channels
      const channelsResponse = await restClient.GET('/channels')
      if (channelsResponse.data) {
        setChannels(channelsResponse.data)
      }

      // Fetch device with device-info
      if (device.uuid) {
        const deviceResponse = await restClient.GET('/devices/{uuid}', {
          params: {
            path: { uuid: device.uuid },
            query: { fields: '*' }
          }
        })
        if (deviceResponse.data) {
          setDeviceInfo(deviceResponse.data['device-info'] as Record<string, unknown> | null)
          setDeviceInfoUpdated(deviceResponse.data['device-info-updated'] ?? null)
        }
      }

      // Fetch firmwares for esp32 platform
      const firmwaresResponse = await restClient.GET('/firmwares', {
        params: { query: { platform: 'esp32' } }
      })
      if (firmwaresResponse.data) {
        setFirmwares(firmwaresResponse.data as FirmwareSummary[])
      }
    }
    if (open) {
      fetchData()
      setName(device.name || '')
      setSelectedChannelUuid(device.channel?.uuid || '')
      setShowDeviceInfo(false)
      setShowFirmwareUpdate(false)
      setSelectedFirmwareUuid('')
    }
  }, [open, device])

  const handleSave = async () => {
    if (!device.uuid) return
    
    setLoading(true)
    try {
      const patchData: { name?: string; channel?: { uuid: string } } = {}
      
      if (name !== device.name) {
        patchData.name = name
      }
      
      if (selectedChannelUuid && selectedChannelUuid !== device.channel?.uuid) {
        patchData.channel = { uuid: selectedChannelUuid }
      }
      
      if (Object.keys(patchData).length > 0) {
        await restClient.PATCH('/devices/{uuid}', {
          params: { path: { uuid: device.uuid } },
          body: patchData,
        })
        revalidator.revalidate()
      }
      
      onOpenChange(false)
    } catch (error) {
      console.error('Failed to update device:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async () => {
    if (!device.uuid) return

    setDeleting(true)
    try {
      await restClient.DELETE('/devices/{uuid}', {
        params: { path: { uuid: device.uuid } },
      })
      revalidator.revalidate()
      setShowDeleteConfirm(false)
      onOpenChange(false)
    } catch (error) {
      console.error('Failed to delete device:', error)
    } finally {
      setDeleting(false)
    }
  }

  const handleIdentify = async () => {
    if (!device.uuid) return

    setIdentifying(true)
    try {
      const deviceName = device.name || 'Unknown Device'
      const ipAddress = device['current-ip'] || device['last-ip'] || 'No IP'
      // Shorten UUID to fit on display (first 8 chars)
      const shortUuid = device.uuid.slice(0, 8)

      await restClient.POST('/devices/{uuid}/push', {
        params: { path: { uuid: device.uuid } },
        body: {
          applet: 'desk-name-tag',
          duration: 30,
          config: {
            name: deviceName,
            line_one: ipAddress,
            line_two: shortUuid,
            text_color: '#FFFFFF',
            background_color: '#FF0000',
          },
        },
      })
    } catch (error) {
      console.error('Failed to identify device:', error)
    } finally {
      setIdentifying(false)
    }
  }

  const handleFirmwareUpdate = async () => {
    if (!device.uuid || !selectedFirmwareUuid) return

    setUpdating(true)
    try {
      await restClient.POST('/devices/{uuid}/ota', {
        params: { path: { uuid: device.uuid } },
        body: { firmware_uuid: selectedFirmwareUuid }
      })
      // Close the section after triggering update
      setShowFirmwareUpdate(false)
      setSelectedFirmwareUuid('')
    } catch (error) {
      console.error('Failed to trigger firmware update:', error)
    } finally {
      setUpdating(false)
    }
  }

  // Get current device firmware SHA256 from device info
  const currentFirmwareSHA = deviceInfo?.sha256 as string | undefined

  // Find if current firmware matches any known firmware
  const currentFirmware = currentFirmwareSHA
    ? firmwares.find(fw => fw['elf-sha256'] === currentFirmwareSHA)
    : undefined

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Configure Device</DialogTitle>
            <DialogDescription>
              Update the device name and channel subscription.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="device-id" className="text-right">
                Device ID
              </Label>
              <div className="col-span-3 relative">
                <Input
                  id="device-id"
                  value={device.uuid || ''}
                  className="pr-10"
                  disabled
                />
                <div
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 cursor-pointer"
                  title="Copy"
                  onClick={() => {
                    if (device.uuid) {
                      navigator.clipboard.writeText(device.uuid)
                    }
                  }}
                >
                  <Copy className="h-4 w-4 text-gray-500 hover:text-gray-700" />
                </div>
              </div>
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="name" className="text-right">
                Name
              </Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="col-span-3"
                placeholder="Enter device name"
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="channel" className="text-right">
                Channel
              </Label>
              <Select value={selectedChannelUuid} onValueChange={setSelectedChannelUuid}>
                <SelectTrigger className="col-span-3">
                  <SelectValue placeholder="Select a channel" />
                </SelectTrigger>
                <SelectContent>
                  {channels.map((channel) => (
                    <SelectItem key={channel.uuid} value={channel.uuid || ''}>
                      {channel.name || channel.uuid}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Device Info Section */}
            {deviceInfo && Object.keys(deviceInfo).length > 0 && (
              <div className="mt-2">
                <button
                  type="button"
                  className="flex items-center text-sm font-medium text-slate-400 hover:text-slate-300"
                  onClick={() => setShowDeviceInfo(!showDeviceInfo)}
                >
                  {showDeviceInfo ? (
                    <ChevronDown className="h-4 w-4 mr-1" />
                  ) : (
                    <ChevronRight className="h-4 w-4 mr-1" />
                  )}
                  Device Info
                  {deviceInfoUpdated && (
                    <span className="ml-2 text-xs text-slate-500">
                      (updated {new Date(deviceInfoUpdated).toLocaleDateString()})
                    </span>
                  )}
                </button>
                {showDeviceInfo && (
                  <div className="mt-2 text-sm space-y-1 pl-5">
                    {Object.entries(deviceInfo).map(([key, value]) => (
                      typeof value !== 'object' && (
                        <div key={key} className="flex">
                          <span className="text-slate-500 w-24 flex-shrink-0">{key}:</span>
                          <span className="text-slate-300">{String(value)}</span>
                        </div>
                      )
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Firmware Update Section */}
            {device.connected && firmwares.length > 0 && (
              <div className="mt-2">
                <button
                  type="button"
                  className="flex items-center text-sm font-medium text-slate-400 hover:text-slate-300"
                  onClick={() => setShowFirmwareUpdate(!showFirmwareUpdate)}
                >
                  {showFirmwareUpdate ? (
                    <ChevronDown className="h-4 w-4 mr-1" />
                  ) : (
                    <ChevronRight className="h-4 w-4 mr-1" />
                  )}
                  Firmware Update
                </button>
                {showFirmwareUpdate && (
                  <div className="mt-2 pl-5 space-y-3">
                    <div className="text-sm">
                      <span className="text-slate-500">Current: </span>
                      <span className="text-slate-300">
                        {currentFirmware
                          ? `${currentFirmware.version} (${new Date(currentFirmware['build-timestamp']).toLocaleDateString()})`
                          : deviceInfo?.version
                            ? String(deviceInfo.version)
                            : 'Unknown'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Select value={selectedFirmwareUuid} onValueChange={setSelectedFirmwareUuid}>
                        <SelectTrigger className="flex-1">
                          <SelectValue placeholder="Select firmware..." />
                        </SelectTrigger>
                        <SelectContent>
                          {firmwares.map((fw) => (
                            <SelectItem key={fw.uuid} value={fw.uuid}>
                              <div className="flex items-center gap-2">
                                <span>{fw.version}</span>
                                <span className="text-slate-500 text-xs">
                                  ({new Date(fw['build-timestamp']).toLocaleDateString()})
                                </span>
                                {fw['is-default'] && (
                                  <Star className="h-3 w-3 text-amber-500" />
                                )}
                                {currentFirmwareSHA === fw['elf-sha256'] && (
                                  <span className="text-xs text-slate-500">current</span>
                                )}
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <Button
                      size="sm"
                      onClick={handleFirmwareUpdate}
                      disabled={!selectedFirmwareUuid || updating || !device.connected}
                    >
                      <RefreshCw className={`h-4 w-4 mr-2 ${updating ? 'animate-spin' : ''}`} />
                      {updating ? 'Updating...' : 'Update Firmware'}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>
          <DialogFooter className="sm:justify-between">
            <div className="flex gap-2">
              <Button
                variant="destructive"
                onClick={() => setShowDeleteConfirm(true)}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete Device
              </Button>
              <Button
                variant="secondary"
                onClick={handleIdentify}
                disabled={identifying || !device.connected}
                title={!device.connected ? 'Device must be connected to identify' : 'Flash device screen for 30 seconds'}
              >
                <Eye className="h-4 w-4 mr-2" />
                {identifying ? 'Identifying...' : 'Identify'}
              </Button>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={loading}>
                {loading ? 'Saving...' : 'Save changes'}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Device</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{device.name || device.uuid}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}