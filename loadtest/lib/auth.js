import { apiPost, expectOK } from './http.js';

const loginCache = new Map();

export function login(username, password) {
  const res = apiPost('/api/v1/auth/login', null, { username, password }, {
    tags: { scenario_group: 'auth', endpoint: 'login' },
  });
  const body = expectOK('login', res, {
    'login: has access token': (r) => {
      const json = r.json();
      return Boolean(json?.data?.access_token);
    },
  });
  return body.data;
}

function cacheKey(username, password) {
  return `${username}\n${password}`;
}

function isTokenFresh(entry) {
  if (!entry || !entry.tokens || !entry.expiresAt) {
    return false;
  }
  // Leave a small buffer so long-running scenarios refresh before expiry.
  return Date.now() + 60_000 < entry.expiresAt;
}

export function loginCached(username, password) {
  const key = cacheKey(username, password);
  const cached = loginCache.get(key);
  if (isTokenFresh(cached)) {
    return cached.tokens;
  }

  const tokens = login(username, password);
  const expiresAt = Date.now() + Number(tokens.expires_in || 0) * 1000;
  loginCache.set(key, { tokens, expiresAt });
  return tokens;
}
