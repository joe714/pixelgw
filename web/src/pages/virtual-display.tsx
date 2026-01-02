import { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { PixelDisplay } from '@/components/PixelDisplay'
import { usePixelWebSocket } from '@/hooks/usePixelWebSocket'

export default function VirtualDisplayPage() {
  const { mode, uuid } = useParams<{ mode: 'channel' | 'device'; uuid: string }>()

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
        scale={8}
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
