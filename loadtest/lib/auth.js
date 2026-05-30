import { apiPost, expectOK } from './http.js';

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
