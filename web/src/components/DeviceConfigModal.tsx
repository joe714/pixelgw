import { useState, useEffect } from 'react'
import { useRevalidator } from 'react-router-dom'
import { Copy } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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

  return (
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
              <Copy
                className="absolute right-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-500 cursor-pointer hover:text-gray-700"
                title="Copy"
                onClick={() => {
                  if (device.uuid) {
                    navigator.clipboard.writeText(device.uuid)
                  }
                }}
              />
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
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={loading}>
            {loading ? 'Saving...' : 'Save changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}