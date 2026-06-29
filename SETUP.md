# Setup & Running Guide

This guide walks you through running Discord Kittens as a **Discord activity** — from
creating the Discord application to launching the game in a voice channel. There are two
ways to host it:

- **Locally** — run everything on your machine and expose it to Discord through a
  [Cloudflare tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/).
  Best for development.
- **Deployed** — host the frontend (e.g. Vercel) and backend (e.g. Render) on the
  public internet. Best for letting other people play.

> Just want to play in a browser without Discord? Skip to
> [Browser-only mode](#browser-only-mode-no-discord-required).

## Prerequisites

| Tool                                                                                                    | Used for                             | Notes                                      |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------ | ------------------------------------------ |
| [Go](https://go.dev/dl/) 1.25+                                                                          | Backend game server                  |                                            |
| [Node.js](https://nodejs.org/) 20+ & npm                                                                | Frontend (Vite)                      |                                            |
| [VS Code](https://code.visualstudio.com/)                                                               | One-key "run everything" task        | Optional but recommended for local hosting |
| [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) | Public tunnel to your local frontend | Only for **local** hosting                 |

A Discord account with permission to create applications.

## 1. Create a Discord application

1. Open the [Discord Developer Portal](https://discord.com/developers/applications).
2. Click **New Application**, give it a name, and create it.

### Get your client id and secret

1. In the sidebar, go to **OAuth2**.
2. Copy the **Client ID**.
3. Under **Client Secret**, click **Reset Secret** (or **Copy** if shown) and copy the value.
   You'll only see the secret once — keep it somewhere safe.

You'll paste both into your `.env` in [step 2](#2-clone-and-configure).

### Add a redirect URL

Still in the **OAuth2** section, under **Redirects**, add:

```
https://127.0.0.1
```

Click **Save Changes**.

### Enable the activity and platforms

1. In the sidebar, go to **Activities → Settings**.
2. Check **all supported platforms** (Desktop, etc.) so the activity can launch everywhere.
3. Save.

You'll set up **URL Mappings** in [step 3](#3-set-up-url-mappings) once your server is running.

## 2. Clone and configure

Clone the repo, then create your `.env` from the example:

```bash
cp .env.example .env
```

Open `.env` and fill in the two values from
[step 1](#get-your-client-id-and-secret):

```bash
VITE_DISCORD_CLIENT_ID=your_discord_app_client_id
DISCORD_CLIENT_SECRET=your_discord_app_client_secret
```

> The `.env` lives at the **repo root** and is shared by both the Go backend (via
> godotenv) and Vite (via `envDir: '..'`). The `VITE_`-prefixed id is exposed to the
> frontend; the secret stays backend-only (used by `/api/token`). `.env` is gitignored —
> never commit it.

Install the frontend dependencies once:

```bash
cd client
npm install
```

## 3. Set up URL Mappings

This is where the two hosting paths diverge. In the Developer Portal, go to
**Activities → URL Mappings**.

### Option A — Local hosting (Cloudflare tunnel)

First, start everything (see [Running locally](#4a-running-locally) below) so you have a
tunnel URL. The tunnel URL looks like `https://something-random.trycloudflare.com` and is
copied to your clipboard automatically.

Set these mappings:

| Prefix     | Target                                                                  |
| ---------- | ----------------------------------------------------------------------- |
| `/` (root) | your cloudflared tunnel URL (e.g. `something-random.trycloudflare.com`) |
| `/cdn`     | `cdn.discordapp.com`                                                    |

> Free Cloudflare quick tunnels get a **new random URL every time you restart them**, so
> you'll need to update the `/` mapping whenever the tunnel URL changes.

### Option B — Deployed hosting

| Prefix     | Target                                                         |
| ---------- | -------------------------------------------------------------- |
| `/` (root) | your frontend URL (e.g. your Vercel deployment)                |
| `/api`     | your backend API URL (e.g. `discord-kittens.onrender.com/api`) |
| `/cdn`     | `cdn.discordapp.com`                                           |

> Inside Discord the frontend always talks to the backend over **same-origin** `/api`
> (Discord's CSP blocks direct cross-origin connections), so the `/api` mapping is what
> routes WebSocket + HTTP traffic to your backend. The frontend's `VITE_WS_URL` build
> variable handles the standalone (non-Discord) case where Vercel can't proxy WebSockets.

## 4a. Running locally

The repo ships a VS Code task that starts the **backend**, **frontend**, and
**cloudflared tunnel** together.

1. Open the project in VS Code.
2. Press **Ctrl+Shift+B** (**Cmd+Shift+B** on macOS) to run the default build task,
   **Dev: All**.

This launches three dedicated terminal panels:

| Task               | What it does                      | Port                           |
| ------------------ | --------------------------------- | ------------------------------ |
| Backend (Go)       | `go run .` in `game/`             | `:8080`                        |
| Frontend (Vite)    | `npm run dev` in `client/`        | `:5173`                        |
| Cloudflared Tunnel | exposes `localhost:5173` publicly | random `trycloudflare.com` URL |

The tunnel task **auto-copies the public URL to your clipboard** and prints it in green in
its terminal panel. Paste it into the `/` URL mapping from
[Option A](#option-a--local-hosting-cloudflare-tunnel).

> **Prefer to run them by hand?** In three terminals:
>
> ```bash
> cd game && go run .          # backend on :8080
> cd client && npm run dev     # frontend on :5173
> cloudflared tunnel --url http://localhost:5173
> ```

> The VS Code tasks work on Windows, macOS, and Linux. On Linux, auto-copying the tunnel
> URL needs `xclip`, `xsel`, or `wl-copy` installed — otherwise the URL is just printed to
> the terminal.

## 4b. Running deployed

Deploy the two halves wherever you like:

- **Frontend** (`client/`) — a static Vite build. On Vercel, set `VITE_DISCORD_CLIENT_ID`
  and `VITE_WS_URL` (your backend's `wss://…/api/ws`) as build-time env vars.
- **Backend** (`game/`) — the Go server. Set `DISCORD_CLIENT_SECRET` (and
  `VITE_DISCORD_CLIENT_ID` if needed) in its environment.

Then fill in the [Option B](#option-b--deployed-hosting) URL mappings.

## 5. Launch the activity in Discord

1. Join a **voice channel** in a server where you can use activities.
2. Open the **activities launcher** (the rocket / controller icon) and pick your app.
3. The game creates and joins a lobby automatically from the activity instance — no lobby
   name needed. Other people in the call can join the same activity to play together.

## Browser-only mode (no Discord required)

You can play entirely in a browser without any Discord setup:

```bash
cd game && go run .          # terminal 1 — backend
cd client && npm run dev     # terminal 2 — frontend
```

Open the Vite URL it prints (usually `http://localhost:5173`), create a lobby, and share
the **lobby name** with other players so they can join the same game. (You don't need the
`.env`, redirect URL, or URL mappings for this — those are only for the Discord activity.)

## Troubleshooting

| Symptom                                          | Likely cause / fix                                                                                                                                                     |
| ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Activity hangs on a loading/handshake screen     | `VITE_DISCORD_CLIENT_ID` missing or wrong in `.env`. Restart the Vite dev server after editing `.env`.                                                                 |
| "Invalid OAuth2 redirect"                        | Make sure `https://127.0.0.1` is added under **OAuth2 → Redirects** and saved.                                                                                         |
| Activity loads but can't connect / no game state | Check your URL mappings. Locally, confirm the `/` mapping matches the **current** tunnel URL (it changes on restart). Deployed, confirm `/api` points at your backend. |
| Avatars don't load                               | Confirm the `/cdn → cdn.discordapp.com` mapping exists.                                                                                                                |
| Tunnel URL not copied to clipboard               | Expected on Linux without a clipboard tool — copy it from the tunnel terminal panel instead.                                                                           |
