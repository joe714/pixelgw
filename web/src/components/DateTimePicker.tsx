import { useState, useEffect, useMemo } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface DateTimePickerProps {
  value: string // ISO string like "2025-11-12T14:45:00-05:00"
  onChange: (value: string) => void
  label?: string
  id?: string
}

// Common timezone offsets with display names
const TIMEZONE_OPTIONS = [
  { offset: '-12:00', label: 'UTC-12:00 (Baker Island)' },
  { offset: '-11:00', label: 'UTC-11:00 (Samoa)' },
  { offset: '-10:00', label: 'UTC-10:00 (Hawaii)' },
  { offset: '-09:00', label: 'UTC-09:00 (Alaska)' },
  { offset: '-08:00', label: 'UTC-08:00 (Pacific)' },
  { offset: '-07:00', label: 'UTC-07:00 (Mountain)' },
  { offset: '-06:00', label: 'UTC-06:00 (Central)' },
  { offset: '-05:00', label: 'UTC-05:00 (Eastern)' },
  { offset: '-04:00', label: 'UTC-04:00 (Atlantic)' },
  { offset: '-03:00', label: 'UTC-03:00 (Argentina)' },
  { offset: '-02:00', label: 'UTC-02:00' },
  { offset: '-01:00', label: 'UTC-01:00 (Azores)' },
  { offset: '+00:00', label: 'UTC+00:00 (London)' },
  { offset: '+01:00', label: 'UTC+01:00 (Paris)' },
  { offset: '+02:00', label: 'UTC+02:00 (Cairo)' },
  { offset: '+03:00', label: 'UTC+03:00 (Moscow)' },
  { offset: '+04:00', label: 'UTC+04:00 (Dubai)' },
  { offset: '+05:00', label: 'UTC+05:00 (Karachi)' },
  { offset: '+05:30', label: 'UTC+05:30 (India)' },
  { offset: '+06:00', label: 'UTC+06:00 (Dhaka)' },
  { offset: '+07:00', label: 'UTC+07:00 (Bangkok)' },
  { offset: '+08:00', label: 'UTC+08:00 (Singapore)' },
  { offset: '+09:00', label: 'UTC+09:00 (Tokyo)' },
  { offset: '+10:00', label: 'UTC+10:00 (Sydney)' },
  { offset: '+11:00', label: 'UTC+11:00' },
  { offset: '+12:00', label: 'UTC+12:00 (Auckland)' },
]

// Get user's local timezone offset in ±HH:MM format
function getLocalTimezoneOffset(): string {
  const offset = new Date().getTimezoneOffset()
  const sign = offset <= 0 ? '+' : '-'
  const absOffset = Math.abs(offset)
  const hours = Math.floor(absOffset / 60).toString().padStart(2, '0')
  const minutes = (absOffset % 60).toString().padStart(2, '0')
  return `${sign}${hours}:${minutes}`
}

// Parse ISO string to components
function parseISOString(isoString: string): { date: string; time: string; offset: string } {
  if (!isoString) {
    const now = new Date()
    const localOffset = getLocalTimezoneOffset()
    return {
      date: now.toISOString().split('T')[0],
      time: now.toTimeString().slice(0, 5),
      offset: localOffset,
    }
  }

  // Handle formats like "2025-11-12T14:45:00-05:00" or "2025-11-12T14:45:00Z"
  const match = isoString.match(/^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2})(:\d{2})?([+-]\d{2}:\d{2}|Z)?$/)
  if (match) {
    const [, date, time, , tz] = match
    let offset = tz || getLocalTimezoneOffset()
    if (offset === 'Z') offset = '+00:00'
    return { date, time, offset }
  }

  // Fallback: try to parse as date
  try {
    const d = new Date(isoString)
    if (!isNaN(d.getTime())) {
      return {
        date: d.toISOString().split('T')[0],
        time: d.toTimeString().slice(0, 5),
        offset: getLocalTimezoneOffset(),
      }
    }
  } catch {
    // ignore
  }

  // Default to now
  const now = new Date()
  return {
    date: now.toISOString().split('T')[0],
    time: now.toTimeString().slice(0, 5),
    offset: getLocalTimezoneOffset(),
  }
}

// Build ISO string from components
function buildISOString(date: string, time: string, offset: string): string {
  return `${date}T${time}:00${offset}`
}

export function DateTimePicker({ value, onChange, label, id }: DateTimePickerProps) {
  const parsed = useMemo(() => parseISOString(value), [value])

  const [date, setDate] = useState(parsed.date)
  const [time, setTime] = useState(parsed.time)
  const [offset, setOffset] = useState(parsed.offset)

  // Sync internal state when value prop changes
  useEffect(() => {
    const newParsed = parseISOString(value)
    setDate(newParsed.date)
    setTime(newParsed.time)
    setOffset(newParsed.offset)
  }, [value])

  const handleDateChange = (newDate: string) => {
    setDate(newDate)
    if (newDate && time) {
      onChange(buildISOString(newDate, time, offset))
    }
  }

  const handleTimeChange = (newTime: string) => {
    setTime(newTime)
    if (date && newTime) {
      onChange(buildISOString(date, newTime, offset))
    }
  }

  const handleOffsetChange = (newOffset: string) => {
    setOffset(newOffset)
    if (date && time) {
      onChange(buildISOString(date, time, newOffset))
    }
  }

  // Find closest matching timezone option
  const selectedOffset = useMemo(() => {
    const found = TIMEZONE_OPTIONS.find(tz => tz.offset === offset)
    return found ? offset : getLocalTimezoneOffset()
  }, [offset])

  return (
    <div className="space-y-2">
      {label && (
        <Label htmlFor={id} className="text-sm">
          {label}
        </Label>
      )}
      <div className="flex gap-2">
        <Input
          type="date"
          id={id}
          value={date}
          onChange={(e) => handleDateChange(e.target.value)}
          className="flex-1"
        />
        <Input
          type="time"
          value={time}
          onChange={(e) => handleTimeChange(e.target.value)}
          className="w-28"
        />
      </div>
      <Select value={selectedOffset} onValueChange={handleOffsetChange}>
        <SelectTrigger className="w-full text-xs">
          <SelectValue placeholder="Select timezone" />
        </SelectTrigger>
        <SelectContent>
          {TIMEZONE_OPTIONS.map((tz) => (
            <SelectItem key={tz.offset} value={tz.offset} className="text-xs">
              {tz.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
