// Import the SDK
import { DiscordSDK, Common } from '@discord/embedded-app-sdk';

function isDiscordActivity(): boolean {
  return new URLSearchParams(window.location.search).has('frame_id');
}

// Instantiate the SDK only inside a Discord activity.
const discordSdk = isDiscordActivity()
  ? new DiscordSDK(import.meta.env.VITE_DISCORD_CLIENT_ID)
  : null;

// Shape of a resolved authenticate() call.
type DiscordAuth = Awaited<
  ReturnType<NonNullable<typeof discordSdk>['commands']['authenticate']>
>;

// The authenticated session, populated once setupDiscordSdk() resolves.
export let auth: DiscordAuth | null = null;

// Instance id available as soon as sdk is instantiated
export function getInstanceId(): string | null {
  return discordSdk?.instanceId || null;
}

// Cached so the handshake runs exactly once even though both main.tsx (eager
// kickoff) and the game page (await-before-connect) call it. Callers share the
// same promise and therefore the same `auth` result.
let setupPromise: Promise<DiscordAuth | null> | null = null;

export function setupDiscordSdk(): Promise<DiscordAuth | null> {
  return (setupPromise ??= doSetupDiscordSdk());
}

async function doSetupDiscordSdk(): Promise<DiscordAuth | null> {
  // Outside a Discord activity there is no SDK and no auth to establish.
  if (!discordSdk) return null;

  // Wait for the host (Discord client) to be ready
  await discordSdk.ready();

  // Lock mobile to landscape (no-op / unsupported on desktop; harmless).
  try {
    await discordSdk.commands.setOrientationLockState({
      lock_state: Common.OrientationLockStateTypeObject.LANDSCAPE,
    });
  } catch {
    // Older clients or desktop may not support orientation locking.
  }

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
