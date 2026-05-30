import { sleep } from 'k6';
import { config, pickUserByVU } from '../config.js';
import { login } from '../lib/auth.js';

export const options = {
  vus: Number(__ENV.VUS || 5),
  duration: __ENV.DURATION || '30s',
  thresholds: config.thresholds,
};

export default function () {
  const user = pickUserByVU();
  login(user.username, user.password);
  sleep(Number(__ENV.SLEEP_SECONDS || 1));
}
