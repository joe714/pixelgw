import { useState, useEffect, useCallback, useRef } from 'react'
import { Input } from '@/components/ui/input'
import { restClient } from '@/rest-client'
import type { components } from '@/openapi'

type Location = components['schemas']['Location']

interface LocationPickerProps {
  value: string // place_id
  onChange: (value: string) => void
  id?: string
}

export function LocationPicker({ value, onChange, id }: LocationPickerProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Location[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [selectedLocation, setSelectedLocation] = useState<Location | null>(null)
  const [loading, setLoading] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Fetch location details when value changes (for existing config)
  useEffect(() => {
    async function fetchLocation() {
      if (!value) {
        setSelectedLocation(null)
        return
      }
      try {
        const response = await restClient.GET('/locations/{placeId}', {
          params: { path: { placeId: value } },
        })
        if (response.data) {
          setSelectedLocation(response.data)
          setQuery(response.data.description)
        }
      } catch {
        // Location not found, clear selection
        setSelectedLocation(null)
      }
    }
    fetchLocation()
  }, [value])

  // Search for locations as user types
  const searchLocations = useCallback(async (searchQuery: string) => {
    if (searchQuery.length < 2) {
      setResults([])
      return
    }

    setLoading(true)
    try {
      const response = await restClient.GET('/locations', {
        params: { query: { q: searchQuery, limit: 8 } },
      })
      if (response.data) {
        setResults(response.data)
      }
    } catch (error) {
      console.error('Failed to search locations:', error)
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [])

  // Debounce search
  useEffect(() => {
    if (!isOpen) return

    const timer = setTimeout(() => {
      searchLocations(query)
    }, 300)

    return () => clearTimeout(timer)
  }, [query, isOpen, searchLocations])

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value)
    setIsOpen(true)
  }

  const handleInputFocus = () => {
    setIsOpen(true)
    if (query.length >= 2) {
      searchLocations(query)
    }
  }

  const handleSelectLocation = (location: Location) => {
    setSelectedLocation(location)
    setQuery(location.description)
    onChange(location.place_id)
    setIsOpen(false)
  }

  const handleClear = () => {
    setSelectedLocation(null)
    setQuery('')
    onChange('')
    inputRef.current?.focus()
  }

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Input
          ref={inputRef}
          id={id}
          type="text"
          value={query}
          onChange={handleInputChange}
          onFocus={handleInputFocus}
          placeholder="Search for a city..."
          className="pr-8"
        />
        {selectedLocation && (
          <button
            type="button"
            onClick={handleClear}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200 text-sm"
          >
            ✕
          </button>
        )}
      </div>

      {isOpen && (results.length > 0 || loading) && (
        <div className="absolute z-50 w-full mt-1 bg-slate-800 border border-slate-600 rounded-md shadow-lg max-h-60 overflow-auto">
          {loading && (
            <div className="px-3 py-2 text-slate-400 text-sm">Searching...</div>
          )}
          {!loading && results.map((location) => (
            <button
              key={location.place_id}
              type="button"
              onClick={() => handleSelectLocation(location)}
              className="w-full text-left px-3 py-2 hover:bg-slate-700 focus:bg-slate-700 focus:outline-none"
            >
              <div className="text-sm text-slate-200">{location.description}</div>
              <div className="text-xs text-slate-500">{location.timezone}</div>
            </button>
          ))}
          {!loading && results.length === 0 && query.length >= 2 && (
            <div className="px-3 py-2 text-slate-400 text-sm">No locations found</div>
          )}
        </div>
      )}

      {selectedLocation && (
        <div className="mt-1 text-xs text-slate-500">
          {selectedLocation.lat}, {selectedLocation.lng} ({selectedLocation.timezone})
        </div>
      )}
    </div>
  )
}
