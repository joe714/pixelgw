import { useState } from 'react'
import { useLoaderData, useRevalidator } from 'react-router-dom'
import { Trash2, Star, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { restClient } from '@/rest-client'
import { UploadFirmwareModal } from '@/components/UploadFirmwareModal'

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

export async function firmwaresLoader() {
  const response = await restClient.GET('/firmwares')
  return response.data || []
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function FirmwaresList() {
  const firmwares = useLoaderData() as FirmwareSummary[]
  const revalidator = useRevalidator()
  const [uploadModalOpen, setUploadModalOpen] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  // Group firmwares by platform
  const byPlatform = firmwares.reduce((acc, fw) => {
    if (!acc[fw.platform]) {
      acc[fw.platform] = []
    }
    acc[fw.platform].push(fw)
    return acc
  }, {} as Record<string, FirmwareSummary[]>)

  const handleDelete = async (uuid: string) => {
    setDeletingId(uuid)
    try {
      await restClient.DELETE('/firmwares/{uuid}', {
        params: { path: { uuid } }
      })
      revalidator.revalidate()
    } catch (error) {
      console.error('Failed to delete firmware:', error)
    } finally {
      setDeletingId(null)
    }
  }

  const handleSetDefault = async (uuid: string) => {
    try {
      await restClient.PATCH('/firmwares/{uuid}', {
        params: { path: { uuid } },
        body: { is_default: true }
      })
      revalidator.revalidate()
    } catch (error) {
      console.error('Failed to set default firmware:', error)
    }
  }

  const platforms = Object.keys(byPlatform).sort()

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-white">Firmwares</h1>
        <Button onClick={() => setUploadModalOpen(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Upload Firmware
        </Button>
      </div>

      {platforms.length === 0 ? (
        <div className="text-center py-12 text-slate-400">
          <p>No firmwares uploaded yet.</p>
          <p className="text-sm mt-2">Click "Upload Firmware" to add your first firmware image.</p>
        </div>
      ) : (
        <div className="space-y-6">
          {platforms.map((platform) => (
            <div key={platform} className="bg-slate-800 rounded-lg overflow-hidden">
              <div className="px-4 py-3 bg-slate-700 border-b border-slate-600">
                <h2 className="text-lg font-semibold text-white">{platform}</h2>
              </div>
              <div className="divide-y divide-slate-700">
                {byPlatform[platform].map((fw) => (
                  <div key={fw.uuid} className="p-4 hover:bg-slate-750">
                    <div className="flex justify-between items-start">
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-white">{fw.filename}</span>
                          {fw['is-default'] && (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-600 text-white">
                              <Star className="h-3 w-3 mr-1" />
                              default
                            </span>
                          )}
                        </div>
                        <div className="mt-1 text-sm text-slate-400">
                          Version: <span className="text-slate-300">{fw.version}</span>
                        </div>
                        <div className="mt-1 text-sm text-slate-500">
                          Built: {formatDate(fw['build-timestamp'])} | IDF {fw['idf-version']} | {formatFileSize(fw['file-size'])}
                        </div>
                        {fw.description && (
                          <div className="mt-2 text-sm text-slate-400 italic">
                            "{fw.description}"
                          </div>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {!fw['is-default'] && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleSetDefault(fw.uuid)}
                            title="Set as default"
                          >
                            <Star className="h-4 w-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(fw.uuid)}
                          disabled={deletingId === fw.uuid}
                          className="text-red-400 hover:text-red-300 hover:bg-red-900/20"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      <UploadFirmwareModal
        open={uploadModalOpen}
        onOpenChange={setUploadModalOpen}
      />
    </div>
  )
}
