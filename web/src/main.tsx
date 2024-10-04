import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from "react-router-dom"

import App from './App.tsx'
import { ChannelList, channelListLoader } from "@/pages/channel-list"

import './index.css'

const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      {
        index: true,
	element: <ChannelList />,
	loader: channelListLoader,
      },
    ],
  },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
