import { useMemo, useState, useEffect, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { PixelDisplay } from '@/components/PixelDisplay'
import { usePixelWebSocket } from '@/hooks/usePixelWebSocket'

// Native display resolution
const NATIVE_WIDTH = 64
const NATIVE_HEIGHT = 32

// Frame adds to dimensions:
// - p-4 = 16px padding each side = 32px total horizontal, 32px total vertical
// - border-2 = 2px each side = 4px total each dimension
// - p-1 = 4px each side = 8px total each dimension (bezel padding)
// - LED area: h-2 (8px) + mb-2 (8px) = 16px vertical only
// - Branding: mt-2 (8px) + text (~12px) = 20px vertical only
const FRAME_EXTRA_WIDTH = 32 + 4 + 8  // 44px
const FRAME_EXTRA_HEIGHT = 32 + 4 + 8 + 16 + 20  // 80px

export default function VirtualDisplayPage() {
  const { mode, uuid } = useParams<{ mode: 'channel' | 'device'; uuid: string }>()
  const [scale, setScale] = useState(8)

  // Calculate optimal scale to fill viewport
  const calculateScale = useCallback(() => {
    // Account for page padding (p-4 = 16px each side) and status bar (~50px)
    const pagePadding = 32
    const statusBarHeight = 50
    const availableWidth = window.innerWidth - pagePadding
    const availableHeight = window.innerHeight - pagePadding - statusBarHeight

    // Calculate max scale that fits both dimensions
    const maxScaleX = (availableWidth - FRAME_EXTRA_WIDTH) / NATIVE_WIDTH
    const maxScaleY = (availableHeight - FRAME_EXTRA_HEIGHT) / NATIVE_HEIGHT

    // Use the smaller of the two, floor it, minimum of 4
    const optimalScale = Math.max(4, Math.floor(Math.min(maxScaleX, maxScaleY)))
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
    <div className="h-screen w-screen bg-black flex flex-col items-center justify-center p-4 overflow-hidden">
      <div className="flex-1 flex items-center justify-center">
        <PixelDisplay
          src={imageUrl || ''}
          scale={scale}
          showFrame={true}
          frameTheme="black"
          powerOn={connected}
          loading={!imageUrl && !error}
          error={!!error}
        />
      </div>

      {/* Status bar */}
      <div className="flex-shrink-0 text-center py-2">
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
