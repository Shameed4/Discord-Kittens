import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { setupDiscordSdk } from "./discord/sdk";

// Kick off the Discord OAuth handshake (authorize -> /api/token -> authenticate)
// as soon as the tab opens. Failures are logged so the app still renders outside
// of a Discord embedded context.
setupDiscordSdk().catch((err) =>
  console.error("Discord SDK setup failed", err),
);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
