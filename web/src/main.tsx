import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from "react-router-dom"

import App from './App.tsx'
import { ChannelList, channelListLoader } from "@/pages/channel-list"
import { ChannelDetail, channelDetailLoader } from "@/pages/channel-detail"
import { DevicesList, devicesLoader } from "@/pages/devices"
import { AppletsList, appletsLoader } from "@/pages/applets"
import { FirmwaresList, firmwaresLoader } from "@/pages/firmwares"
import { PixelDisplayDebug } from "@/pages/pixel-display-debug"
import VirtualDisplayPage from "@/pages/virtual-display"

import './index.css'

const router = createBrowserRouter([
  // Virtual display routes (no layout chrome)
  {
    path: "/display/:mode/:uuid",
    element: <VirtualDisplayPage />,
  },
  // Main app routes (with layout)
  {
    path: "/",
    element: <App />,
    children: [
      {
        index: true,
	element: <ChannelList />,
	loader: channelListLoader,
      },
      {
        path: "channels/:uuid",
        element: <ChannelDetail />,
        loader: channelDetailLoader,
      },
      {
        path: "devices",
        element: <DevicesList />,
        loader: devicesLoader,
      },
      {
        path: "firmwares",
        element: <FirmwaresList />,
        loader: firmwaresLoader,
      },
      {
        path: "applets",
        element: <AppletsList />,
        loader: appletsLoader,
      },
      {
        path: "debug",
        element: <PixelDisplayDebug />,
      },
    ],
  },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
