import { useState, useEffect, useCallback } from 'react'
import { useRevalidator } from 'react-router-dom'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { PixelDisplay } from '@/components/PixelDisplay'
import { AppletConfigForm } from '@/components/AppletConfigForm'
import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type AppInstanceDetail = components['schemas']['AppInstanceDetail']
type App = components['schemas']['App']

interface EditAppletModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelUuid: string
  applet: AppInstanceDetail | null
}

export function EditAppletModal({ open, onOpenChange, channelUuid, applet }: EditAppletModalProps) {
  const [appletDetails, setAppletDetails] = useState<App | null>(null)
  const [config, setConfig] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const revalidator = useRevalidator()

  // Fetch applet schema when modal opens
  useEffect(() => {
    async function fetchAppletDetails() {
      if (!applet?.['app-id']) return
      setLoading(true)
      try {
        const response = await restClient.GET('/applets/{id}', {
          params: { path: { id: applet['app-id'] } },
        })
        if (response.data) {
          setAppletDetails(response.data)
        }
      } catch (error) {
        console.error('Failed to fetch applet details:', error)
      } finally {
        setLoading(false)
      }
    }

    if (open && applet) {
      fetchAppletDetails()
      // Parse existing config
      try {
        const existingConfig = applet.config ? JSON.parse(applet.config) : {}
        setConfig(existingConfig)
      } catch {
        setConfig({})
      }
    }
  }, [open, applet])

  // Debounced preview rendering
  const updatePreview = useCallback(async () => {
    if (!applet?.['app-id']) return
    try {
      const configJson = JSON.stringify(config)
      const url = `/api/applets/${encodeURIComponent(applet['app-id'])}/render?config=${encodeURIComponent(configJson)}`
      setPreviewUrl(url + '&t=' + Date.now()) // Cache bust
    } catch (error) {
      console.error('Failed to render preview:', error)
    }
  }, [applet, config])

  useEffect(() => {
    if (open && applet) {
      const timer = setTimeout(updatePreview, 500)
      return () => clearTimeout(timer)
    }
  }, [open, applet, config, updatePreview])

  const handleSave = async () => {
    if (!applet?.uuid) return

    setSaving(true)
    try {
      await restClient.PATCH('/channels/{channelUUID}/applets/{appletUUID}', {
        params: {
          path: {
            channelUUID: channelUuid,
            appletUUID: applet.uuid,
          },
        },
        body: {
          config: JSON.stringify(config),
        },
      })
      revalidator.revalidate()
      onOpenChange(false)
    } catch (error) {
      console.error('Failed to update applet:', error)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Edit {applet?.['app-id']}</DialogTitle>
          <DialogDescription>
            Update the applet configuration.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-hidden flex gap-6 py-4">
          {/* Preview panel */}
          <div className="flex-shrink-0 flex flex-col items-center">
            <div className="text-xs text-slate-500 mb-2">Preview</div>
            <PixelDisplay
              src={previewUrl || ''}
              scale={3}
              showFrame={true}
              frameTheme="black"
              loading={!previewUrl}
            />
          </div>

          {/* Config form */}
          <div className="flex-1 overflow-y-auto pr-2">
            {loading ? (
              <div className="text-center py-4 text-slate-500">Loading schema...</div>
            ) : (
              <AppletConfigForm
                schema={appletDetails?.schema}
                config={config}
                onChange={setConfig}
              />
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
