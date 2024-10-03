import React from "react";
import ReactDOM from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
//import { OpenAPI as OpenAPIConfig } from "../openapi/requests/core/OpenAPI";

import App from "./App.tsx";
import AppletViewPane from "./AppletViewPane.tsx";
import ChannelViewPane from "./ChannelViewPane.tsx";
import ErrorPage from "./ErrorPage.tsx";

//OpenAPIConfig.BASE='/api/';

const queryClient = new QueryClient();

const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    errorElement: <ErrorPage />,
    children: [
      {
        path: "/applets",
	element: <AppletViewPane />
      },
      {
        path: "/channels",
	element: <ChannelViewPane />
      },
    ],
  },
]);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>,
);
