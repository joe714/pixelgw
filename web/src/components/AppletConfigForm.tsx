import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DateTimePicker } from '@/components/DateTimePicker'
import type { components } from '@/openapi'

type SchemaField = components['schemas']['SchemaField']
type Schema = components['schemas']['Schema']

interface AppletConfigFormProps {
  schema: Schema | undefined
  config: Record<string, string>
  onChange: (config: Record<string, string>) => void
}

export function AppletConfigForm({ schema, config, onChange }: AppletConfigFormProps) {
  if (!schema?.schema || schema.schema.length === 0) {
    return (
      <div className="text-slate-500 text-sm py-4">
        This applet has no configurable options.
      </div>
    )
  }

  const handleFieldChange = (fieldId: string, value: string) => {
    onChange({ ...config, [fieldId]: value })
  }

  const renderField = (field: SchemaField) => {
    if (!field.id) return null

    const fieldValue = config[field.id] ?? field.default ?? ''

    switch (field.type) {
      case 'text':
        return (
          <div key={field.id} className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor={field.id} className="text-right text-sm">
              {field.name || field.id}
            </Label>
            <Input
              id={field.id}
              value={fieldValue}
              onChange={(e) => handleFieldChange(field.id!, e.target.value)}
              className="col-span-3"
              placeholder={field.name || field.id}
            />
          </div>
        )

      case 'dropdown':
        return (
          <div key={field.id} className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor={field.id} className="text-right text-sm">
              {field.name || field.id}
            </Label>
            <Select
              value={fieldValue}
              onValueChange={(value) => handleFieldChange(field.id!, value)}
            >
              <SelectTrigger className="col-span-3">
                <SelectValue placeholder={`Select ${field.name || field.id}`} />
              </SelectTrigger>
              <SelectContent>
                {field.options?.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.display || option.text}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )

      case 'onoff':
      case 'toggle':
        return (
          <div key={field.id} className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor={field.id} className="text-right text-sm">
              {field.name || field.id}
            </Label>
            <div className="col-span-3">
              <input
                type="checkbox"
                id={field.id}
                checked={fieldValue === 'true'}
                onChange={(e) => handleFieldChange(field.id!, e.target.checked ? 'true' : 'false')}
                className="h-4 w-4 rounded border-gray-300"
              />
            </div>
          </div>
        )

      case 'color':
        return (
          <div key={field.id} className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor={field.id} className="text-right text-sm">
              {field.name || field.id}
            </Label>
            <div className="col-span-3 flex items-center gap-2">
              <input
                type="color"
                id={field.id}
                value={fieldValue || '#ffffff'}
                onChange={(e) => handleFieldChange(field.id!, e.target.value)}
                className="h-8 w-12 rounded border border-gray-600 cursor-pointer"
              />
              <Input
                value={fieldValue}
                onChange={(e) => handleFieldChange(field.id!, e.target.value)}
                className="flex-1"
                placeholder="#ffffff"
              />
            </div>
          </div>
        )

      case 'datetime':
        return (
          <div key={field.id} className="grid grid-cols-4 items-start gap-4">
            <Label htmlFor={field.id} className="text-right text-sm pt-2">
              {field.name || field.id}
            </Label>
            <div className="col-span-3">
              <DateTimePicker
                id={field.id}
                value={fieldValue}
                onChange={(value) => handleFieldChange(field.id!, value)}
              />
            </div>
          </div>
        )

      default:
        // For unknown types, render as text input
        return (
          <div key={field.id} className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor={field.id} className="text-right text-sm">
              {field.name || field.id}
            </Label>
            <Input
              id={field.id}
              value={fieldValue}
              onChange={(e) => handleFieldChange(field.id!, e.target.value)}
              className="col-span-3"
              placeholder={field.name || field.id}
            />
          </div>
        )
    }
  }

  return (
    <div className="space-y-4">
      {schema.schema.map((field) => renderField(field))}
    </div>
  )
}
