import { makeLoader, useLoaderData } from "react-router-typesafe"
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Package, Plus, Star, Clock, Code } from 'lucide-react'
import { restClient } from '@/rest-client'

export const appletsLoader = makeLoader(
  async () => await restClient.GET("/applets")
);

export function AppletsList() {
  const { data } = useLoaderData<typeof appletsLoader>();
  
  const applets = data?.map((applet) => {
    return (
      <div key={applet.id} className="bg-gray-900 rounded-lg p-4 border border-gray-800 hover:border-gray-700 transition-colors">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-3">
            <div className="p-2 bg-gray-800 rounded-lg">
              <Package className="h-5 w-5 text-lime-500" />
            </div>
            <div className="flex flex-col">
              <h3 className="font-bold text-lg text-white">{applet.name || applet.id}</h3>
              <p className="text-sm text-gray-400 mt-1">{applet.description || 'No description available'}</p>
              <div className="flex gap-4 mt-3 text-xs text-gray-500">
                {applet.author && (
                  <span className="flex items-center gap-1">
                    <Code className="h-3 w-3" />
                    {applet.author}
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  ID: {applet.id}
                </span>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <Button variant="outline" size="sm">View</Button>
            <Button variant="ghost" size="sm">
              <Star className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    )
  })

  return (
    <div className="flex flex-col p-4">
      <div className="flex flex-row justify-between items-center mb-4">
        <div>
          <h2 className="text-2xl font-bold text-white">Applets</h2>
          <p className="text-sm text-gray-500 mt-1">Available Pixlet applications from community and local sources</p>
        </div>
        <Button variant="ghost" className="p-2 space-x-1 bg-lime-700">
          <Plus className="h-4 w-4" />
          <Label className="font-bold">Upload Applet</Label>
        </Button>
      </div>
      
      <div className="mt-4">
        {applets && applets.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {applets}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500">
            <Package className="h-12 w-12 mx-auto mb-4 text-gray-600" />
            <p>No applets available</p>
            <p className="text-sm mt-2">Applets will be loaded from /apps directory</p>
          </div>
        )}
      </div>
    </div>
  )
}