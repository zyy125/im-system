import { sleep } from 'k6';
import { config, pickUserByVU } from '../config.js';
import { login } from '../lib/auth.js';
import { apiGet, expectOK } from '../lib/http.js';

export const options = {
  vus: Number(__ENV.VUS || 5),
  duration: __ENV.DURATION || '30s',
  thresholds: config.thresholds,
};

export default function () {
  const user = pickUserByVU();
  const tokens = login(user.username, user.password);
  const token = tokens.access_token;

  expectOK(
    'conversation-list',
    apiGet('/api/v1/conversations', token, {
      tags: { scenario_group: 'http', endpoint: 'conversation_list' },
    }),
  );

  expectOK(
    'group-list',
    apiGet('/api/v1/conversations/groups', token, {
      tags: { scenario_group: 'http', endpoint: 'group_list' },
    }),
  );

  sleep(Number(__ENV.SLEEP_SECONDS || 1));
}
