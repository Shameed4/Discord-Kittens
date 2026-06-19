// Import the SDK
import { DiscordSDK } from '@discord/embedded-app-sdk';

// Instantiate the SDK
const discordSdk = new DiscordSDK(import.meta.env.VITE_DISCORD_CLIENT_ID);

// The authenticated session, populated once setupDiscordSdk() resolves.
export let auth: Awaited<
  ReturnType<typeof discordSdk.commands.authenticate>
> | null = null;

// Cached so the handshake runs exactly once even though both main.tsx (eager
// kickoff) and the game page (await-before-connect) call it. Callers share the
// same promise and therefore the same `auth` result.
let setupPromise: Promise<typeof auth> | null = null;

export function setupDiscordSdk() {
  if (!setupPromise) setupPromise = doSetupDiscordSdk();
  return setupPromise;
}

async function doSetupDiscordSdk() {
  // Wait for the host (Discord client) to be ready
  await discordSdk.ready();

  // Authorize with Discord to obtain a short-lived OAuth2 code
  const { code } = await discordSdk.commands.authorize({
    client_id: import.meta.env.VITE_DISCORD_CLIENT_ID,
    response_type: 'code',
    state: '',
    prompt: 'none',
    scope: ['identify'],
  });

  // Exchange the code for an access_token via our backend (/api/token -> :8080/token)
  const response = await fetch('/api/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  });
  const { access_token } = await response.json();

  // Authenticate the SDK with the access_token
  auth = await discordSdk.commands.authenticate({ access_token });

  return auth;
}

// Best available display name for the authenticated Discord user, or null when
// running outside an authenticated Discord context.
export function getUsername(): string | null {
  if (!auth) return null;
  return auth.user.global_name ?? auth.user.username;
}

// Stable Discord user id for the authenticated user, or null when running
// outside an authenticated Discord context. The backend uses this to reconnect
// a player to their existing seat when they rejoin a lobby they left.
export function getUserId(): string | null {
  if (!auth) return null;
  return auth.user.id;
}

export { discordSdk };
