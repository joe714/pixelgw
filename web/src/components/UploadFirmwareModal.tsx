import { useState, useRef } from 'react'
import { useRevalidator } from 'react-router-dom'
import { Upload } from 'lucide-react'
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

interface UploadFirmwareModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface ESP32Metadata {
  version: string
  buildDate: string
  buildTime: string
  idfVersion: string
  elfSHA256: string
}

// ESP32 binary format constants
const ESP32_MAGIC = 0xABCD5432
const MAGIC_OFFSET = 0x20
const VERSION_OFFSET = 0x30
const BUILD_TIME_OFFSET = 0x70
const BUILD_DATE_OFFSET = 0x80
const IDF_VERSION_OFFSET = 0x90
const ELF_SHA256_OFFSET = 0xB0

function extractNullTerminatedString(data: Uint8Array, offset: number, maxLen: number): string {
  let end = offset
  while (end < offset + maxLen && data[end] !== 0) {
    end++
  }
  return new TextDecoder().decode(data.slice(offset, end))
}

function extractESP32Metadata(data: ArrayBuffer): ESP32Metadata | null {
  if (data.byteLength < 0xD0) {
    return null
  }

  const view = new DataView(data)
  const bytes = new Uint8Array(data)

  // Check magic word
  const magic = view.getUint32(MAGIC_OFFSET, true)
  if (magic !== ESP32_MAGIC) {
    return null
  }

  // Extract strings
  const version = extractNullTerminatedString(bytes, VERSION_OFFSET, 32)
  const buildTime = extractNullTerminatedString(bytes, BUILD_TIME_OFFSET, 16)
  const buildDate = extractNullTerminatedString(bytes, BUILD_DATE_OFFSET, 16)
  const idfVersion = extractNullTerminatedString(bytes, IDF_VERSION_OFFSET, 32)

  // Extract ELF SHA256 as hex
  const sha256Bytes = bytes.slice(ELF_SHA256_OFFSET, ELF_SHA256_OFFSET + 32)
  const elfSHA256 = Array.from(sha256Bytes).map(b => b.toString(16).padStart(2, '0')).join('')

  return {
    version,
    buildDate,
    buildTime,
    idfVersion,
    elfSHA256,
  }
}

export function UploadFirmwareModal({ open, onOpenChange }: UploadFirmwareModalProps) {
  const [platform, setPlatform] = useState('esp32')
  const [description, setDescription] = useState('')
  const [isDefault, setIsDefault] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [metadata, setMetadata] = useState<ESP32Metadata | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const revalidator = useRevalidator()

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) {
      setSelectedFile(null)
      setMetadata(null)
      return
    }

    setSelectedFile(file)
    setError(null)

    // Read first 512 bytes for metadata preview
    try {
      const slice = file.slice(0, 512)
      const buffer = await slice.arrayBuffer()
      const meta = extractESP32Metadata(buffer)
      if (meta) {
        setMetadata(meta)
      } else {
        setMetadata(null)
        setError('Could not extract ESP32 metadata. File may not be a valid firmware binary.')
      }
    } catch (err) {
      console.error('Failed to read file:', err)
      setMetadata(null)
    }
  }

  const handleUpload = async () => {
    if (!selectedFile) return

    setUploading(true)
    setError(null)

    try {
      const formData = new FormData()
      formData.append('platform', platform)
      formData.append('firmware', selectedFile)
      if (description) {
        formData.append('description', description)
      }
      if (isDefault) {
        formData.append('is_default', 'true')
      }

      const response = await fetch('/api/firmwares', {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Upload failed')
      }

      revalidator.revalidate()
      onOpenChange(false)

      // Reset form
      setSelectedFile(null)
      setMetadata(null)
      setDescription('')
      setIsDefault(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    } catch (err) {
      console.error('Upload failed:', err)
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  const handleClose = () => {
    onOpenChange(false)
    setSelectedFile(null)
    setMetadata(null)
    setDescription('')
    setIsDefault(false)
    setError(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Upload Firmware</DialogTitle>
          <DialogDescription>
            Upload a new firmware binary for your devices.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="platform" className="text-right">
              Platform
            </Label>
            <Select value={platform} onValueChange={setPlatform}>
              <SelectTrigger className="col-span-3">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="esp32">esp32</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="file" className="text-right">
              File
            </Label>
            <div className="col-span-3">
              <Input
                id="file"
                type="file"
                accept=".bin"
                ref={fileInputRef}
                onChange={handleFileChange}
              />
            </div>
          </div>

          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="description" className="text-right">
              Description
            </Label>
            <Input
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="col-span-3"
              placeholder="Optional description"
            />
          </div>

          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="default" className="text-right">
              Default
            </Label>
            <div className="col-span-3 flex items-center">
              <input
                id="default"
                type="checkbox"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300"
              />
              <Label htmlFor="default" className="ml-2 text-sm text-slate-400">
                Set as default for this platform
              </Label>
            </div>
          </div>

          {metadata && (
            <div className="mt-2 p-3 bg-slate-800 rounded-lg text-sm">
              <div className="text-slate-400 text-xs uppercase tracking-wide mb-2">Preview</div>
              <div className="space-y-1">
                <div className="flex">
                  <span className="text-slate-500 w-20">Version:</span>
                  <span className="text-slate-300">{metadata.version}</span>
                </div>
                <div className="flex">
                  <span className="text-slate-500 w-20">Built:</span>
                  <span className="text-slate-300">{metadata.buildDate} {metadata.buildTime}</span>
                </div>
                <div className="flex">
                  <span className="text-slate-500 w-20">IDF:</span>
                  <span className="text-slate-300">{metadata.idfVersion}</span>
                </div>
                <div className="flex">
                  <span className="text-slate-500 w-20">SHA256:</span>
                  <span className="text-slate-300 font-mono text-xs">{metadata.elfSHA256.slice(0, 16)}...</span>
                </div>
                {selectedFile && (
                  <div className="flex">
                    <span className="text-slate-500 w-20">Size:</span>
                    <span className="text-slate-300">{(selectedFile.size / 1024).toFixed(1)} KB</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {error && (
            <div className="p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-400">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            onClick={handleUpload}
            disabled={!selectedFile || uploading}
          >
            <Upload className="h-4 w-4 mr-2" />
            {uploading ? 'Uploading...' : 'Upload'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
