import { useState, useEffect } from 'react'
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
import type { components } from '@/openapi'

type DeviceSummary = components['schemas']['DeviceSummary']

interface CreateVirtualDeviceModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (device: DeviceSummary) => void
}

export function CreateVirtualDeviceModal({
  open,
  onOpenChange,
  onCreated,
}: CreateVirtualDeviceModalProps) {
  const [name, setName] = useState('')
  const [selectedChannelUuid, setSelectedChannelUuid] = useState('')
  const [channels, setChannels] = useState<Array<{ uuid?: string; name?: string }>>([])
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchChannels() {
      const response = await restClient.GET('/channels')
      if (response.data) {
        setChannels(response.data)
        // Select first channel by default
        if (response.data.length > 0) {
          setSelectedChannelUuid(response.data[0].uuid || '')
        }
      }
    }
    if (open) {
      fetchChannels()
      setName('')
      setSelectedChannelUuid('')
      setError(null)
    }
  }, [open])

  const handleCreate = async () => {
    if (!name.trim()) {
      setError('Device name is required')
      return
    }
    if (!selectedChannelUuid) {
      setError('Please select a channel')
      return
    }

    setCreating(true)
    setError(null)

    try {
      const response = await restClient.POST('/devices', {
        body: {
          name: name.trim(),
          channel: { uuid: selectedChannelUuid },
        },
      })

      if (response.error) {
        setError(response.error.message || 'Failed to create device')
        return
      }

      if (response.data) {
        onCreated(response.data)
        onOpenChange(false)
      }
    } catch (err) {
      console.error('Failed to create device:', err)
      setError('Failed to create device')
    } finally {
      setCreating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Virtual Device</DialogTitle>
          <DialogDescription>
            Create a new virtual device that can be used to display channel content in a browser.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="device-name" className="text-right">
              Name
            </Label>
            <Input
              id="device-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="col-span-3"
              placeholder="My Virtual Display"
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
          {error && (
            <div className="text-sm text-red-500 text-center">{error}</div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={creating}>
            {creating ? 'Creating...' : 'Create & Open'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
