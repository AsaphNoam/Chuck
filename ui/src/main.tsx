import "./lib/renamedStorage";
import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { router } from "./routes";
import { queryClient } from "./api/config";
import { sseClient } from "./api/sse";
import { installBrowserDiagnostics } from "./lib/browserDiagnostics";
import "./styles/index.css";
import { AppearanceRoot } from "./features/appearance/AppearanceRoot";

installBrowserDiagnostics();
sseClient.connect();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <AppearanceRoot />
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>,
);
