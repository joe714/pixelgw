import { useState, useEffect, useRef, useCallback } from 'react'

interface UsePixelWebSocketResult {
  imageUrl: string | null
  connected: boolean
  error: string | null
  reconnect: () => void
}

export function usePixelWebSocket(url: string | null): UsePixelWebSocketResult {
  const [imageUrl, setImageUrl] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const previousUrlRef = useRef<string | null>(null)

  const connect = useCallback(() => {
    if (!url) return

    // Clean up previous connection
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setError(null)

    const ws = new WebSocket(url)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      setConnected(true)
      setError(null)
    }

    ws.onclose = (event) => {
      setConnected(false)
      if (!event.wasClean) {
        setError('Connection closed unexpectedly')
      }
    }

    ws.onerror = () => {
      setError('Connection failed')
      setConnected(false)
    }

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        // Revoke previous URL to prevent memory leaks
        if (previousUrlRef.current) {
          URL.revokeObjectURL(previousUrlRef.current)
        }
        // Convert to blob URL for img src
        const blob = new Blob([event.data], { type: 'image/webp' })
        const blobUrl = URL.createObjectURL(blob)
        previousUrlRef.current = blobUrl
        setImageUrl(blobUrl)
      }
    }
  }, [url])

  useEffect(() => {
    connect()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      // Clean up blob URL on unmount
      if (previousUrlRef.current) {
        URL.revokeObjectURL(previousUrlRef.current)
        previousUrlRef.current = null
      }
    }
  }, [connect])

  const reconnect = useCallback(() => {
    connect()
  }, [connect])

  return { imageUrl, connected, error, reconnect }
}
