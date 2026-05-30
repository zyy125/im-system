import { sleep } from 'k6';
import { config, pickUserByVU } from '../config.js';
import { login } from '../lib/auth.js';
import { connect, wsConnectFailure, wsConnectSuccess } from '../lib/ws.js';

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
  thresholds: config.thresholds,
};

export default function () {
  const user = pickUserByVU();
  const tokens = login(user.username, user.password);

  const res = connect(tokens.access_token, (socket) => {
    socket.on('open', () => {
      wsConnectSuccess.add(1);
    });

    socket.on('error', () => {
      wsConnectFailure.add(1);
    });

    socket.on('close', () => {
      wsConnectFailure.add(0);
    });

    socket.setTimeout(() => {
      socket.close();
    }, Number(__ENV.SOCKET_LIFETIME_MS || 30000));
  }, {
    tags: { scenario_group: 'ws', endpoint: 'connect' },
  });

  if (res && res.error) {
    wsConnectFailure.add(1);
  }

  sleep(1);
}
