import { SharedArray } from 'k6/data';

const rawUsers = new SharedArray('loadtest-users', () => {
  const path = __ENV.USERS_FILE || './data/users.json';
  return JSON.parse(open(path));
});

const baseUrl = (__ENV.BASE_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '');
const wsBaseUrl = (__ENV.WS_URL || baseUrl.replace(/^http/, 'ws')).replace(/\/+$/, '');

export const config = {
  baseUrl,
  wsBaseUrl,
  users: rawUsers,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
    checks: ['rate>0.99'],
  },
};

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
