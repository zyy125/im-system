import { config, envNumber, pickUserByVU } from '../config.js';
import { login } from '../lib/auth.js';
import {
  closeSocket,
  connect,
  wsConnectDuration,
  wsConnectFailure,
  wsConnectSuccess,
} from '../lib/ws.js';

export const options = {
  scenarios: {
    ws_connect: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || '20s', target: Number(__ENV.TARGET_VUS || 20) },
        { duration: __ENV.HOLD || '40s', target: Number(__ENV.TARGET_VUS || 20) },
        { duration: __ENV.RAMP_DOWN || '10s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    ...config.thresholds,
    ws_connect_failure_total: ['count==0'],
    ws_connect_success_total: [`count>=${envNumber('TARGET_VUS', 20) * 0.95}`],
    ws_connect_duration_ms: ['p(95)<1000'],
  },
};

export default function () {
  const user = pickUserByVU();
  const tokens = login(user.username, user.password);
  const startedAt = Date.now();
  const socketLifetimeMs = envNumber('SOCKET_LIFETIME_SECONDS', 30) * 1000;
  let opened = false;
  let failureRecorded = false;

  const socket = connect(tokens.access_token, {
    tags: { scenario_group: 'ws', endpoint: 'connect' },
  });

  const timeoutId = setTimeout(() => {
    closeSocket(socket);
  }, socketLifetimeMs);

  function recordFailure() {
    if (failureRecorded) {
      return;
    }
    failureRecorded = true;
    wsConnectFailure.add(1);
  }

  socket.addEventListener('open', () => {
    opened = true;
    wsConnectSuccess.add(1);
    wsConnectDuration.add(Date.now() - startedAt);
  });

  socket.addEventListener('error', () => {
    recordFailure();
  });

  socket.addEventListener('close', () => {
    clearTimeout(timeoutId);
    if (!opened) {
      recordFailure();
    }
  });
}
