import { makeLoader, useLoaderData } from "react-router-typesafe"
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'

import { SquarePlus } from 'lucide-react'

import { restClient } from '@/rest-client'

export const channelListLoader = makeLoader(
  async () => await restClient.GET("/channels")
);

export function ChannelList() {
  const { data } = useLoaderData<typeof channelListLoader>();
  console.log(data)
  const rows = data?.map((e) => {
    return (
        <li>
          <div className="flex flex-col w-full">
            <Separator className="my-4" />
            <div className="flex flex-row justify-start">
              <div className="font-bold text-lg text-sky-500">{e.name}</div>
            </div>
            <div className="text-slate-300 my-1">{e.comment}</div>
            <div className="flex flex-row text-sm text-slate-400 space-x-2">
              <div>3 devices</div>
              <div>2 apps</div>
            </div>
          </div>
        </li>
      )
  })
            
  return (
    <div className="flex flex-col p-4">
      <div className="flex flex-row justify-end">
        <Button variant="ghost" className="p-2 space-x-1 bg-lime-700">
          <span className="sr-only">New Channel</span>
          <SquarePlus className="h-4 w-4" />
          <Label className="font-bold">New</Label>
        </Button>
      </div>
      <div>
        <ul>
          {rows}
        </ul>
        <Separator className="my-4" />
      </div>
    </div>
  )
}
