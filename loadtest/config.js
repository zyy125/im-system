import { SharedArray } from 'k6/data';

function normalizeUsersFile(path) {
  if (!path) {
    return './data/users.json';
  }
  if (path.startsWith('./') || path.startsWith('../') || path.startsWith('/')) {
    return path;
  }
  // k6 resolves open() paths relative to the current script directory.
  // Accepting "loadtest/data/..." is convenient when commands are run from repo root.
  if (path.startsWith('loadtest/')) {
    return `../${path}`;
  }
  return path;
}

const rawUsers = new SharedArray('loadtest-users', () => {
  const path = normalizeUsersFile(__ENV.USERS_FILE);
  return JSON.parse(open(path));
});

function deriveWSOrigin(baseUrl) {
  const explicitOrigin = (__ENV.WS_ORIGIN || '').trim();
  if (explicitOrigin) {
    return explicitOrigin.replace(/\/+$/, '');
  }

  const match = baseUrl.match(/^https?:\/\/(localhost|127\.0\.0\.1)(?::\d+)?$/);
  if (!match) {
    return '';
  }

  return `http://${match[1]}:4174`;
}

const baseUrl = (__ENV.BASE_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
const wsBaseUrl = (__ENV.WS_URL || baseUrl.replace(/^http/, 'ws')).replace(/\/+$/, '');
const wsOrigin = deriveWSOrigin(baseUrl);

export const config = {
  baseUrl,
  wsBaseUrl,
  wsOrigin,
  users: rawUsers,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
    checks: ['rate>0.99'],
  },
};

const validationCache = new Set();

export function pickUserByVU() {
  if (!config.users.length) {
    throw new Error('no load test users configured');
  }
  const index = (__VU - 1) % config.users.length;
  return config.users[index];
}

export function requirePair(user) {
  if (!user || !user.peer) {
    throw new Error('scenario requires a user with peer metadata');
  }
  return user;
}

export function envNumber(name, fallback) {
  return Number(__ENV[name] || fallback);
}

export function ensureUserCapacity(requiredUsers, label) {
  if (requiredUsers > config.users.length) {
    throw new Error(
      `${label} requires at least ${requiredUsers} user entries, but only ${config.users.length} are configured`,
    );
  }
}

export function ensureDedicatedChatPairs(requiredPairs, label) {
  const cacheKey = `${label}:${requiredPairs}`;
  if (validationCache.has(cacheKey)) {
    return;
  }

  ensureUserCapacity(requiredPairs, label);

  const seen = new Set();
  for (let i = 0; i < requiredPairs; i += 1) {
    const user = requirePair(config.users[i]);
    const identities = [user.username, user.peer.username];
    const conversationId = user.peer.conversation_id || user.conversation_id;

    if (!conversationId) {
      throw new Error(`${label} user ${user.username} is missing conversation_id`);
    }

    for (const username of identities) {
      if (!username) {
        throw new Error(`${label} contains an empty username`);
      }
      if (seen.has(username)) {
        throw new Error(
          `${label} requires dedicated self/peer accounts per VU; duplicate username detected: ${username}`,
        );
      }
      seen.add(username);
    }
  }

  validationCache.add(cacheKey);
}
