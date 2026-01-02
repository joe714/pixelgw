import { useMemo, useState, useEffect, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { PixelDisplay } from '@/components/PixelDisplay'
import { usePixelWebSocket } from '@/hooks/usePixelWebSocket'

// Native display resolution
const NATIVE_WIDTH = 64
const NATIVE_HEIGHT = 32

// Frame adds padding: p-4 (16px each side) + border (2px each side) + bezel padding (4px each side)
// Plus LED area (~24px) and branding (~24px) at top/bottom
const FRAME_PADDING_X = 16 + 2 + 4  // 22px each side = 44px total
const FRAME_PADDING_Y = 16 + 2 + 4 + 24 + 24  // ~70px total

export default function VirtualDisplayPage() {
  const { mode, uuid } = useParams<{ mode: 'channel' | 'device'; uuid: string }>()
  const [scale, setScale] = useState(8)

  // Calculate optimal scale to fill viewport
  const calculateScale = useCallback(() => {
    // Leave some margin around the display
    const margin = 32
    const availableWidth = window.innerWidth - margin * 2
    const availableHeight = window.innerHeight - margin * 2 - 60 // 60px for status bar

    // Calculate max scale that fits both dimensions
    const maxScaleX = Math.floor((availableWidth - FRAME_PADDING_X * 2) / NATIVE_WIDTH)
    const maxScaleY = Math.floor((availableHeight - FRAME_PADDING_Y) / NATIVE_HEIGHT)

    // Use the smaller of the two, with a minimum of 4 and maximum of 16
    const optimalScale = Math.max(4, Math.min(16, Math.min(maxScaleX, maxScaleY)))
    setScale(optimalScale)
  }, [])

  // Calculate scale on mount and window resize
  useEffect(() => {
    calculateScale()
    window.addEventListener('resize', calculateScale)
    return () => window.removeEventListener('resize', calculateScale)
  }, [calculateScale])

  // Build WebSocket URL based on mode
  const wsUrl = useMemo(() => {
    if (!uuid) return null

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const base = `${protocol}//${window.location.host}`

    if (mode === 'channel') {
      return `${base}/ws/display?channel=${uuid}`
    } else {
      return `${base}/ws?device=${uuid}`
    }
  }, [mode, uuid])

  const { imageUrl, connected, error, reconnect } = usePixelWebSocket(wsUrl)

  return (
    <div className="min-h-screen bg-black flex flex-col items-center justify-center p-4">
      <PixelDisplay
        src={imageUrl || ''}
        scale={scale}
        showFrame={true}
        frameTheme="black"
        powerOn={connected}
        loading={!imageUrl && !error}
        error={!!error}
      />

      {/* Status bar */}
      <div className="mt-4 text-center">
        {error && (
          <div className="text-red-500 text-sm mb-2">
            {error}
          </div>
        )}
        <div className="flex items-center justify-center gap-4 text-xs text-gray-500">
          <span className="flex items-center gap-1">
            <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
            {connected ? 'Connected' : 'Disconnected'}
          </span>
          <span>|</span>
          <span className="capitalize">{mode}</span>
          <span>|</span>
          <span className="font-mono text-xs">{uuid?.slice(0, 8)}...</span>
        </div>
        {!connected && (
          <button
            onClick={reconnect}
            className="mt-2 px-3 py-1 text-xs bg-gray-800 hover:bg-gray-700 text-gray-300 rounded"
          >
            Reconnect
          </button>
        )}
      </div>
    </div>
  )
}
