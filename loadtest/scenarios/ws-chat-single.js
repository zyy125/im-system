import { uuidv4 as randomUUID } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import {
  config,
  ensureDedicatedChatPairs,
  envNumber,
  pickUserByVU,
  requirePair,
} from '../config.js';
import { login } from '../lib/auth.js';
import { apiGet, apiPost, expectOK } from '../lib/http.js';
import {
  closeSocket,
  connect,
  safeParse,
  sendEnvelope,
  wsConnectFailure,
  wsConnectSuccess,
  wsErrors,
  wsMessageCreated,
  wsMessageDeliveredAck,
  wsMessageRead,
  wsMessageSent,
  wsMessageSentAck,
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

function syncConversation(token, conversationId, afterSeq) {
  return expectOK(
    'message-sync',
    apiGet(`/api/v1/messages/sync?conversation_id=${conversationId}&after_seq=${afterSeq}`, token, {
      tags: { scenario_group: 'ws_chat', endpoint: 'message_sync' },
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
    ws_connect_failure_total: ['count==0'],
    ws_error_events_total: ['count==0'],
    ws_sync_required_total: ['count==0'],
    ws_message_sent_ack_total: [`count>=${envNumber('TARGET_VUS', 10) * 0.95}`],
    ws_message_created_total: [`count>=${envNumber('TARGET_VUS', 10) * 0.95}`],
    ws_message_read_total: [`count>=${envNumber('TARGET_VUS', 10) * 0.95}`],
    ws_message_round_trip_ms: ['p(95)<1000'],
  },
};

export function setup() {
  ensureDedicatedChatPairs(envNumber('TARGET_VUS', 10), 'ws-chat-single');
  return null;
}

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
  openConversation(peerTokens.access_token, conversationId);
  history(peerTokens.access_token, conversationId);

  const msgId = randomUUID();
  const messageText = `k6-${msgId}`;
  const startedAt = Date.now();
  const socketLifetimeMs = envNumber('SOCKET_LIFETIME_MS', 20000);

  const state = {
    selfOpen: false,
    peerOpen: false,
    messageSent: false,
    messageSentAck: false,
    messageCreated: false,
    messageRead: false,
    deliveredSeq: 0,
    done: false,
  };

  const selfSocket = connect(selfTokens.access_token, {
    tags: { scenario_group: 'ws_chat', endpoint: 'self_ws' },
  });

  const peerSocket = connect(peerTokens.access_token, {
    tags: { scenario_group: 'ws_chat', endpoint: 'peer_ws' },
  });
  let timeoutId = null;

  function clearScenarioTimeout() {
    if (timeoutId !== null) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  }

  function markFailure() {
    if (state.done) {
      return;
    }
    wsConnectFailure.add(1);
    state.done = true;
    clearScenarioTimeout();
    closeSocket(selfSocket, 4000, 'scenario_failed');
    closeSocket(peerSocket, 4000, 'scenario_failed');
  }

  function finishIfDone() {
    if (state.done || !state.messageSentAck || !state.messageCreated || !state.messageRead) {
      return;
    }
    wsRoundTrip.add(Date.now() - startedAt);
    state.done = true;
    clearScenarioTimeout();
    closeSocket(selfSocket);
    closeSocket(peerSocket);
  }

  timeoutId = setTimeout(markFailure, socketLifetimeMs);

  selfSocket.addEventListener('open', () => {
    state.selfOpen = true;
    wsConnectSuccess.add(1);
    if (state.peerOpen && !state.messageSent) {
      sendEnvelope(selfSocket, 'message.send', {
        msg_id: msgId,
        conversation_id: conversationId,
        content: messageText,
      });
      state.messageSent = true;
      wsMessageSent.add(1);
    }
  });

  selfSocket.addEventListener('message', (event) => {
    const env = safeParse(event.data);
    if (!env || !env.type) {
      return;
    }

    if (env.type === 'message.sent') {
      if (env.data && env.data.msg_id === msgId) {
        state.messageSentAck = true;
        wsMessageSentAck.add(1);
        finishIfDone();
      }
      return;
    }

    if (env.type === 'message.read') {
      if (env.data && env.data.read_seq >= state.deliveredSeq && state.deliveredSeq > 0) {
        state.messageRead = true;
        wsMessageRead.add(1);
        finishIfDone();
      }
      return;
    }

    if (env.type === 'sync.required') {
      wsSyncRequired.add(1);
      syncConversation(selfTokens.access_token, conversationId, state.deliveredSeq);
      return;
    }

    if (env.type === 'error') {
      wsErrors.add(1);
      markFailure();
    }
  });

  selfSocket.addEventListener('error', markFailure);
  selfSocket.addEventListener('close', () => {
    if (!state.done) {
      markFailure();
    }
  });

  peerSocket.addEventListener('open', () => {
    state.peerOpen = true;
    wsConnectSuccess.add(1);
    if (state.selfOpen && !state.messageSent) {
      sendEnvelope(selfSocket, 'message.send', {
        msg_id: msgId,
        conversation_id: conversationId,
        content: messageText,
      });
      state.messageSent = true;
      wsMessageSent.add(1);
    }
  });

  peerSocket.addEventListener('message', (event) => {
    const env = safeParse(event.data);
    if (!env || !env.type) {
      return;
    }

    if (env.type === 'message.created') {
      if (env.data && env.data.msg_id === msgId) {
        state.messageCreated = true;
        state.deliveredSeq = env.data.seq;
        wsMessageCreated.add(1);
        sendEnvelope(peerSocket, 'message.delivered', {
          conversation_id: conversationId,
          delivered_seq: state.deliveredSeq,
        });
        wsMessageDeliveredAck.add(1);
        sendEnvelope(peerSocket, 'message.read', {
          conversation_id: conversationId,
          read_seq: state.deliveredSeq,
        });
        finishIfDone();
      }
      return;
    }

    if (env.type === 'sync.required') {
      wsSyncRequired.add(1);
      syncConversation(peerTokens.access_token, conversationId, state.deliveredSeq);
      return;
    }

    if (env.type === 'error') {
      wsErrors.add(1);
      markFailure();
    }
  });

  peerSocket.addEventListener('error', markFailure);
  peerSocket.addEventListener('close', () => {
    if (!state.done) {
      markFailure();
    }
  });
}
