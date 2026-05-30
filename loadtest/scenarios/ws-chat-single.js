import { sleep } from 'k6';
import { randomUUID } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { config, pickUserByVU, requirePair } from '../config.js';
import { login } from '../lib/auth.js';
import { apiGet, apiPost, expectOK } from '../lib/http.js';
import {
  connect,
  safeParse,
  sendEnvelope,
  wsConnectFailure,
  wsConnectSuccess,
  wsErrors,
  wsMessageCreated,
  wsMessageDelivered,
  wsMessageRead,
  wsMessageSent,
  wsRoundTrip,
  wsSyncRequired,
} from '../lib/ws.js';

function openConversation(token, conversationId) {
  return expectOK(
    'open-conversation',
    apiPost(`/api/v1/conversations/${conversationId}/open`, token, null, {
      tags: { scenario_group: 'ws_chat', endpoint: 'conversation_open' },
    }),
  );
}

function history(token, conversationId) {
  return expectOK(
    'message-history',
    apiGet(`/api/v1/messages/history?conversation_id=${conversationId}`, token, {
      tags: { scenario_group: 'ws_chat', endpoint: 'message_history' },
    }),
  );
}

export const options = {
  scenarios: {
    ws_chat_single: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || '20s', target: Number(__ENV.TARGET_VUS || 10) },
        { duration: __ENV.HOLD || '40s', target: Number(__ENV.TARGET_VUS || 10) },
        { duration: __ENV.RAMP_DOWN || '10s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    ...config.thresholds,
    ws_message_round_trip_ms: ['p(95)<1000'],
  },
};

export default function () {
  const user = requirePair(pickUserByVU());
  const selfTokens = login(user.username, user.password);
  const peerTokens = login(user.peer.username, user.peer.password);
  const conversationId = user.peer.conversation_id || user.conversation_id;

  if (!conversationId) {
    throw new Error('ws-chat-single requires conversation_id in test user data');
  }

  openConversation(selfTokens.access_token, conversationId);
  history(selfTokens.access_token, conversationId);

  const msgId = randomUUID();
  const messageText = `k6-${msgId}`;
  const start = Date.now();

  let deliveredSeq = 0;

  const peerSession = connect(peerTokens.access_token, (socket) => {
    socket.on('open', () => {
      wsConnectSuccess.add(1);
    });

    socket.on('message', (raw) => {
      const env = safeParse(raw);
      if (!env || !env.type) {
        return;
      }

      if (env.type === 'message.created') {
        wsMessageCreated.add(1);
        if (env.data && env.data.msg_id === msgId) {
          deliveredSeq = env.data.seq;
          sendEnvelope(socket, 'message.delivered', {
            conversation_id: conversationId,
            delivered_seq: deliveredSeq,
          });
          sendEnvelope(socket, 'message.read', {
            conversation_id: conversationId,
            read_seq: deliveredSeq,
          });
        }
      } else if (env.type === 'sync.required') {
        wsSyncRequired.add(1);
      } else if (env.type === 'error') {
        wsErrors.add(1);
      }
    });

    socket.setTimeout(() => socket.close(), Number(__ENV.SOCKET_LIFETIME_MS || 20000));
  }, {
    tags: { scenario_group: 'ws_chat', endpoint: 'peer_ws' },
  });

  const selfSession = connect(selfTokens.access_token, (socket) => {
    socket.on('open', () => {
      wsConnectSuccess.add(1);
      sendEnvelope(socket, 'message.send', {
        msg_id: msgId,
        conversation_id: conversationId,
        content: messageText,
      });
      wsMessageSent.add(1);
    });

    socket.on('message', (raw) => {
      const env = safeParse(raw);
      if (!env || !env.type) {
        return;
      }

      if (env.type === 'message.delivered') {
        wsMessageDelivered.add(1);
      } else if (env.type === 'message.read') {
        wsMessageRead.add(1);
        if (env.data && env.data.read_seq >= deliveredSeq && deliveredSeq > 0) {
          wsRoundTrip.add(Date.now() - start);
        }
      } else if (env.type === 'sync.required') {
        wsSyncRequired.add(1);
      } else if (env.type === 'error') {
        wsErrors.add(1);
      }
    });

    socket.on('error', () => {
      wsConnectFailure.add(1);
    });

    socket.setTimeout(() => socket.close(), Number(__ENV.SOCKET_LIFETIME_MS || 20000));
  }, {
    tags: { scenario_group: 'ws_chat', endpoint: 'self_ws' },
  });

  if (peerSession && peerSession.error) {
    wsConnectFailure.add(1);
  }
  if (selfSession && selfSession.error) {
    wsConnectFailure.add(1);
  }

  sleep(Number(__ENV.SLEEP_SECONDS || 1));
}
