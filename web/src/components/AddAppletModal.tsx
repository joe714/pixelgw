import { useState, useEffect, useCallback } from 'react'
import { useRevalidator } from 'react-router-dom'
import { ArrowLeft, Search } from 'lucide-react'
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
import { PixelDisplay } from '@/components/PixelDisplay'
import { AppletConfigForm } from '@/components/AppletConfigForm'
import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type App = components['schemas']['App']

interface AddAppletModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelUuid: string
  nextIdx: number
}

export function AddAppletModal({ open, onOpenChange, channelUuid, nextIdx }: AddAppletModalProps) {
  const [step, setStep] = useState<'select' | 'configure'>('select')
  const [applets, setApplets] = useState<App[]>([])
  const [selectedApplet, setSelectedApplet] = useState<App | null>(null)
  const [config, setConfig] = useState<Record<string, string>>({})
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const revalidator = useRevalidator()

  // Fetch applets when modal opens
  useEffect(() => {
    async function fetchApplets() {
      const response = await restClient.GET('/applets')
      if (response.data) {
        setApplets(response.data)
      }
    }
    if (open) {
      fetchApplets()
      setStep('select')
      setSelectedApplet(null)
      setConfig({})
      setSearchQuery('')
      setPreviewUrl(null)
    }
  }, [open])

  // Fetch applet details (with schema) when selected
  const selectedAppletId = selectedApplet?.id
  useEffect(() => {
    async function fetchAppletDetails() {
      if (!selectedAppletId) return
      setLoading(true)
      try {
        const response = await restClient.GET('/applets/{id}', {
          params: { path: { id: selectedAppletId } },
        })
        if (response.data) {
          setSelectedApplet(response.data)
          // Initialize config with defaults
          const defaults: Record<string, string> = {}
          response.data.schema?.schema?.forEach((field) => {
            if (field.id && field.default) {
              defaults[field.id] = field.default
            }
          })
          setConfig(defaults)
        }
      } catch (error) {
        console.error('Failed to fetch applet details:', error)
      } finally {
        setLoading(false)
      }
    }
    if (step === 'configure' && selectedAppletId) {
      fetchAppletDetails()
    }
  }, [step, selectedAppletId])

  // Debounced preview rendering
  const updatePreview = useCallback(async () => {
    if (!selectedApplet?.id) return
    try {
      const configJson = JSON.stringify(config)
      const url = `/api/applets/${encodeURIComponent(selectedApplet.id)}/render?config=${encodeURIComponent(configJson)}`
      setPreviewUrl(url + '&t=' + Date.now()) // Cache bust
    } catch (error) {
      console.error('Failed to render preview:', error)
    }
  }, [selectedApplet?.id, config])

  useEffect(() => {
    if (step === 'configure' && selectedApplet) {
      const timer = setTimeout(updatePreview, 500)
      return () => clearTimeout(timer)
    }
  }, [step, selectedApplet, config, updatePreview])

  const handleSelectApplet = (applet: App) => {
    setSelectedApplet(applet)
    setStep('configure')
  }

  const handleBack = () => {
    setStep('select')
    setSelectedApplet(null)
    setConfig({})
    setPreviewUrl(null)
  }

  const handleSave = async () => {
    if (!selectedApplet?.id) return

    setSaving(true)
    try {
      await restClient.POST('/channels/{channelUUID}/applets', {
        params: { path: { channelUUID: channelUuid } },
        body: {
          'app-id': selectedApplet.id,
          idx: nextIdx,
          config: JSON.stringify(config),
        },
      })
      revalidator.revalidate()
      onOpenChange(false)
    } catch (error) {
      console.error('Failed to add applet:', error)
    } finally {
      setSaving(false)
    }
  }

  const filteredApplets = applets.filter(
    (applet) =>
      applet.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      applet.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      applet.summary.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[800px] max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {step === 'select' ? 'Add Applet' : (
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" onClick={handleBack} className="p-1">
                  <ArrowLeft className="h-4 w-4" />
                </Button>
                Configure {selectedApplet?.name}
              </div>
            )}
          </DialogTitle>
          <DialogDescription>
            {step === 'select'
              ? 'Select an applet to add to this channel.'
              : 'Configure the applet settings.'}
          </DialogDescription>
        </DialogHeader>

        {step === 'select' ? (
          <div className="flex-1 overflow-hidden flex flex-col">
            <div className="relative mb-4">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
              <Input
                placeholder="Search applets..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex-1 overflow-y-auto grid grid-cols-2 gap-2 pr-2">
              {filteredApplets.map((applet) => (
                <button
                  key={applet.id}
                  onClick={() => handleSelectApplet(applet)}
                  className="text-left p-3 rounded-lg border border-slate-700 hover:border-sky-500 hover:bg-slate-800 transition-colors"
                >
                  <div className="font-medium text-sm text-slate-200 truncate">
                    {applet.name}
                  </div>
                  <div className="text-xs text-slate-500 truncate">
                    {applet.summary}
                  </div>
                  <div className="text-xs text-slate-600 mt-1">
                    by {applet.author}
                  </div>
                </button>
              ))}
              {filteredApplets.length === 0 && (
                <div className="col-span-2 text-center py-8 text-slate-500">
                  No applets found.
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="flex-1 overflow-hidden flex gap-6">
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
                  schema={selectedApplet?.schema}
                  config={config}
                  onChange={setConfig}
                />
              )}
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          {step === 'configure' && (
            <Button onClick={handleSave} disabled={saving}>
              {saving ? 'Adding...' : 'Add Applet'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
