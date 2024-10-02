import React from "react";
import ReactDOM from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";

import App from "./App.tsx";
import AppletViewPane from "./AppletViewPane.tsx";
import ChannelViewPane from "./ChannelViewPane.tsx";
import ErrorPage from "./ErrorPage.tsx";

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
    <RouterProvider router={router} />
  </React.StrictMode>,
);
