const SESSION_COOKIE_NAME = 'oc_admin_session';
const SESSION_TTL_SECONDS = 60 * 60 * 12;
const LOGIN_WINDOW_MS = 15 * 60 * 1000;
const MAX_LOGIN_ATTEMPTS = 5;

type AdminSession = {
  sub: 'admin';
  username: string;
  iat: number;
  exp: number;
};

type FailedAttempt = {
  count: number;
  resetAt: number;
};

const loginAttempts = new Map<string, FailedAttempt>();

function getSessionSecret() {
  const explicitSecret = process.env.OPEN_CONTEXT_ADMIN_SESSION_SECRET?.trim();
  if (explicitSecret) {
    return explicitSecret;
  }

  const apiKey = process.env.OPEN_CONTEXT_API_KEY?.trim();
  if (apiKey && apiKey !== 'changeme') {
    return apiKey;
  }

  return null;
}

function encodeBase64Url(bytes: Uint8Array) {
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

function decodeBase64Url(input: string) {
  const padded = input.replace(/-/g, '+').replace(/_/g, '/');
  const normalized = padded.padEnd(Math.ceil(padded.length / 4) * 4, '=');
  const binary = atob(normalized);
  const bytes = new Uint8Array(binary.length);

  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }

  return bytes;
}

function getCryptoKey(secret: string) {
  return crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign', 'verify'],
  );
}

async function signPayload(payload: string, secret: string) {
  const key = await getCryptoKey(secret);
  const signature = await crypto.subtle.sign('HMAC', key, new TextEncoder().encode(payload));
  return encodeBase64Url(new Uint8Array(signature));
}

function decodeSessionPayload(encodedPayload: string) {
  try {
    const json = new TextDecoder().decode(decodeBase64Url(encodedPayload));
    return JSON.parse(json) as AdminSession;
  } catch {
    return null;
  }
}

export function getAdminCookieName() {
  return SESSION_COOKIE_NAME;
}

export function getAdminSessionTtlSeconds() {
  return SESSION_TTL_SECONDS;
}

export function getAdminAuthConfigError() {
  if (!getSessionSecret()) {
    return 'Set OPEN_CONTEXT_ADMIN_SESSION_SECRET or a non-default OPEN_CONTEXT_API_KEY.';
  }

  return null;
}

export async function createAdminSession(username: string) {
  const secret = getSessionSecret();
  if (!secret) {
    throw new Error('Admin auth session secret is not configured.');
  }

  const now = Math.floor(Date.now() / 1000);
  const payload = encodeBase64Url(
    new TextEncoder().encode(
      JSON.stringify({
        sub: 'admin',
        username,
        iat: now,
        exp: now + SESSION_TTL_SECONDS,
      } satisfies AdminSession),
    ),
  );
  const signature = await signPayload(payload, secret);

  return `${payload}.${signature}`;
}

export async function verifyAdminSession(token: string | null | undefined) {
  if (!token) {
    return null;
  }

  const [encodedPayload, signature, extra] = token.split('.');
  if (!encodedPayload || !signature || extra) {
    return null;
  }

  const secret = getSessionSecret();
  if (!secret) {
    return null;
  }

  const expectedSignature = await signPayload(encodedPayload, secret);
  if (signature !== expectedSignature) {
    return null;
  }

  const payload = decodeSessionPayload(encodedPayload);
  if (!payload || payload.sub !== 'admin' || !payload.username) {
    return null;
  }

  const now = Math.floor(Date.now() / 1000);
  if (!Number.isFinite(payload.exp) || payload.exp <= now) {
    return null;
  }

  return payload;
}

export function getLoginClientKey(request: Request) {
  const forwarded = request.headers.get('x-forwarded-for');
  if (forwarded) {
    return forwarded.split(',')[0]?.trim() || 'unknown';
  }

  return request.headers.get('x-real-ip') || 'unknown';
}

function getFailedAttempt(clientKey: string) {
  const existing = loginAttempts.get(clientKey);
  if (!existing) {
    return null;
  }

  if (existing.resetAt <= Date.now()) {
    loginAttempts.delete(clientKey);
    return null;
  }

  return existing;
}

export function getRemainingLoginCooldownMs(clientKey: string) {
  const failedAttempt = getFailedAttempt(clientKey);
  if (!failedAttempt || failedAttempt.count < MAX_LOGIN_ATTEMPTS) {
    return 0;
  }

  return Math.max(0, failedAttempt.resetAt - Date.now());
}

export function recordFailedLogin(clientKey: string) {
  const existing = getFailedAttempt(clientKey);
  if (!existing) {
    loginAttempts.set(clientKey, {
      count: 1,
      resetAt: Date.now() + LOGIN_WINDOW_MS,
    });
    return;
  }

  loginAttempts.set(clientKey, {
    count: existing.count + 1,
    resetAt: existing.resetAt,
  });
}

export function clearFailedLogin(clientKey: string) {
  loginAttempts.delete(clientKey);
}
