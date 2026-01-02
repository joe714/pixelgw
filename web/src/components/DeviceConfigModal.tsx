import { useState, useEffect } from 'react'
import { useRevalidator } from 'react-router-dom'
import { Copy, Trash2, Eye } from 'lucide-react'
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
  const revalidator = useRevalidator()

  useEffect(() => {
    async function fetchChannels() {
      const response = await restClient.GET('/channels')
      if (response.data) {
        setChannels(response.data)
      }
    }
    if (open) {
      fetchChannels()
      setName(device.name || '')
      setSelectedChannelUuid(device.channel?.uuid || '')
    }
  }, [open, device])

  const handleSave = async () => {
    if (!device.uuid) return
    
    setLoading(true)
    try {
      const patchData: any = {}
      
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