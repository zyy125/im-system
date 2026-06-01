import { sleep } from 'k6';
import { uuidv4 as randomUUID } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { config, ensureDedicatedChatPairs, envNumber, pickUserByVU, requirePair } from '../config.js';
import { loginCached } from '../lib/auth.js';
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
  wsMessageRead,
  wsMessageSent,
  wsMessageSentAck,
  wsRoundTrip,
  wsSyncRequired,
} from '../lib/ws.js';

const openedConversationCache = new Map();

function openConversation(token, conversationId) {
  return expectOK(
    'open-conversation',
    apiPost(`/api/v1/conversations/${conversationId}/open`, token, null, {
      tags: { scenario_group: 'mixed', endpoint: 'conversation_open' },
    }),
  );
}

function openConversationCacheKey(token, conversationId) {
  return `${token}\n${conversationId}`;
}

function ensureConversationOpen(token, conversationId) {
  const key = openConversationCacheKey(token, conversationId);
  if (openedConversationCache.has(key)) {
    return;
  }
  openConversation(token, conversationId);
  openedConversationCache.set(key, true);
}

function syncConversation(token, conversationId, afterSeq) {
  return expectOK(
    'message-sync',
    apiGet(`/api/v1/messages/sync?conversation_id=${conversationId}&after_seq=${afterSeq}`, token, {
      tags: { scenario_group: 'mixed', endpoint: 'message_sync' },
    }),
  );
}

function everyNIterations(n) {
  const interval = Math.max(1, n);
  return (__ITER - 1) % interval === 0;
}

export const options = {
  scenarios: {
    idle_ws: {
      executor: 'ramping-vus',
      exec: 'idleWs',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || '30s', target: envNumber('IDLE_WS_VUS', 30) },
        { duration: __ENV.HOLD || '2m', target: envNumber('IDLE_WS_VUS', 30) },
        { duration: __ENV.RAMP_DOWN || '15s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
    conversation_poll: {
      executor: 'constant-vus',
      exec: 'conversationPoll',
      vus: envNumber('HTTP_VUS', 5),
      duration: __ENV.DURATION || '3m',
    },
    ws_chat_flow: {
      executor: 'ramping-vus',
      exec: 'wsChatFlow',
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || '30s', target: envNumber('CHAT_VUS', 8) },
        { duration: __ENV.HOLD || '2m', target: envNumber('CHAT_VUS', 8) },
        { duration: __ENV.RAMP_DOWN || '15s', target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    ...config.thresholds,
    ws_connect_failure_total: ['count==0'],
    ws_error_events_total: ['count==0'],
    ws_sync_required_total: ['count<5'],
    ws_message_sent_ack_total: [`count>=${envNumber('CHAT_VUS', 8) * 0.9}`],
    ws_message_created_total: [`count>=${envNumber('CHAT_VUS', 8) * 0.9}`],
    ws_message_read_total: [`count>=${envNumber('CHAT_VUS', 8) * 0.9}`],
    ws_message_round_trip_ms: ['p(95)<1500'],
  },
};

export function setup() {
  ensureDedicatedChatPairs(envNumber('CHAT_VUS', 8), 'mixed-dev');
  return null;
}

export function idleWs() {
  const user = pickUserByVU();
  const tokens = loginCached(user.username, user.password);
  const socket = connect(tokens.access_token, {
    tags: { scenario_group: 'mixed', endpoint: 'idle_ws' },
  });
  const timeoutId = setTimeout(() => {
    closeSocket(socket);
  }, envNumber('IDLE_SOCKET_LIFETIME_SECONDS', 60) * 1000);

  let opened = false;
  let failureRecorded = false;

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
  });
  socket.addEventListener('message', (event) => {
    const env = safeParse(event.data);
    if (!env || !env.type) {
      return;
    }
    if (env.type === 'sync.required') {
      wsSyncRequired.add(1);
    } else if (env.type === 'error') {
      wsErrors.add(1);
    }
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

export function conversationPoll() {
  const user = pickUserByVU();
  const tokens = loginCached(user.username, user.password);
  const token = tokens.access_token;
  const conversationPollEvery = envNumber('HTTP_POLL_EVERY', 3);
  const groupPollEvery = envNumber('HTTP_GROUP_POLL_EVERY', 6);

  if (!everyNIterations(conversationPollEvery)) {
    sleep(envNumber('HTTP_SLEEP_SECONDS', 2));
    return;
  }

  expectOK(
    'conversation-list',
    apiGet('/api/v1/conversations', token, {
      tags: { scenario_group: 'mixed', endpoint: 'conversation_list' },
    }),
  );

  if (everyNIterations(groupPollEvery)) {
    expectOK(
      'group-list',
      apiGet('/api/v1/conversations/groups', token, {
        tags: { scenario_group: 'mixed', endpoint: 'group_list' },
      }),
    );
  }

  sleep(envNumber('HTTP_SLEEP_SECONDS', 2));
}

export function wsChatFlow() {
  const user = requirePair(pickUserByVU());
  const selfTokens = loginCached(user.username, user.password);
  const peerTokens = loginCached(user.peer.username, user.peer.password);
  const conversationId = user.peer.conversation_id || user.conversation_id;
  const burstSize = Math.max(1, envNumber('CHAT_MESSAGES_PER_SESSION', 5));
  const socketLifetimeMs = envNumber('SOCKET_LIFETIME_MS', 20000);

  if (!conversationId) {
    throw new Error('mixed-dev requires conversation_id in test user data');
  }

  ensureConversationOpen(selfTokens.access_token, conversationId);
  ensureConversationOpen(peerTokens.access_token, conversationId);

  const state = {
    selfOpen: false,
    peerOpen: false,
    sending: false,
    pendingMsgId: '',
    pendingText: '',
    pendingSeq: 0,
    sentAcked: false,
    createdSeen: false,
    readSeen: false,
    messagesCompleted: 0,
    done: false,
  };
  const startedAt = Date.now();

  const selfSocket = connect(selfTokens.access_token, {
    tags: { scenario_group: 'mixed', endpoint: 'chat_self_ws' },
  });
  const peerSocket = connect(peerTokens.access_token, {
    tags: { scenario_group: 'mixed', endpoint: 'chat_peer_ws' },
  });
  let timeoutId = null;

  function clearScenarioTimeout() {
    if (timeoutId !== null) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  }

  function complete() {
    if (state.done) {
      return;
    }
    state.done = true;
    clearScenarioTimeout();
    wsRoundTrip.add(Date.now() - startedAt);
    closeSocket(selfSocket);
    closeSocket(peerSocket);
  }

  function fail() {
    if (state.done) {
      return;
    }
    wsConnectFailure.add(1);
    state.done = true;
    clearScenarioTimeout();
    closeSocket(selfSocket, 4000, 'scenario_failed');
    closeSocket(peerSocket, 4000, 'scenario_failed');
  }

  function startNextMessage() {
    if (state.done || state.sending || !state.selfOpen || !state.peerOpen) {
      return;
    }
    if (state.messagesCompleted >= burstSize) {
      complete();
      return;
    }

    const msgId = randomUUID();
    state.pendingMsgId = msgId;
    state.pendingText = `mixed-${msgId}`;
    state.pendingSeq = 0;
    state.sentAcked = false;
    state.createdSeen = false;
    state.readSeen = false;
    state.sending = true;

    sendEnvelope(selfSocket, 'message.send', {
      msg_id: msgId,
      conversation_id: conversationId,
      content: state.pendingText,
    });
    wsMessageSent.add(1);
  }

  function advanceIfMessageDone() {
    if (!state.sending || !state.sentAcked || !state.createdSeen || !state.readSeen) {
      return;
    }

    state.messagesCompleted += 1;
    state.sending = false;
    state.pendingMsgId = '';
    state.pendingText = '';
    state.pendingSeq = 0;

    if (state.messagesCompleted >= burstSize) {
      complete();
      return;
    }

    startNextMessage();
  }

  timeoutId = setTimeout(fail, socketLifetimeMs);

  selfSocket.addEventListener('open', () => {
    state.selfOpen = true;
    wsConnectSuccess.add(1);
    startNextMessage();
  });
  selfSocket.addEventListener('message', (event) => {
    const env = safeParse(event.data);
    if (!env || !env.type) {
      return;
    }

    if (env.type === 'message.sent' && env.data?.msg_id === state.pendingMsgId) {
      if (!state.sentAcked) {
        state.sentAcked = true;
        wsMessageSentAck.add(1);
        advanceIfMessageDone();
      }
    } else if (env.type === 'message.read' && env.data?.read_seq >= state.pendingSeq && state.pendingSeq > 0) {
      if (!state.readSeen) {
        state.readSeen = true;
        wsMessageRead.add(1);
        advanceIfMessageDone();
      }
    } else if (env.type === 'sync.required') {
      wsSyncRequired.add(1);
      syncConversation(selfTokens.access_token, conversationId, state.pendingSeq);
    } else if (env.type === 'error') {
      wsErrors.add(1);
      fail();
    }
  });
  selfSocket.addEventListener('error', fail);
  selfSocket.addEventListener('close', () => {
    if (!state.done) {
      fail();
    }
  });

  peerSocket.addEventListener('open', () => {
    state.peerOpen = true;
    wsConnectSuccess.add(1);
    startNextMessage();
  });
  peerSocket.addEventListener('message', (event) => {
    const env = safeParse(event.data);
    if (!env || !env.type) {
      return;
    }
    if (env.type === 'message.created' && env.data?.msg_id === state.pendingMsgId) {
      if (!state.createdSeen) {
        state.createdSeen = true;
        state.pendingSeq = env.data.seq;
        wsMessageCreated.add(1);
        sendEnvelope(peerSocket, 'message.delivered', {
          conversation_id: conversationId,
          delivered_seq: state.pendingSeq,
        });
        sendEnvelope(peerSocket, 'message.read', {
          conversation_id: conversationId,
          read_seq: state.pendingSeq,
        });
        advanceIfMessageDone();
      }
    } else if (env.type === 'sync.required') {
      wsSyncRequired.add(1);
      syncConversation(peerTokens.access_token, conversationId, state.pendingSeq);
    } else if (env.type === 'error') {
      wsErrors.add(1);
      fail();
    }
  });
  peerSocket.addEventListener('error', fail);
  peerSocket.addEventListener('close', () => {
    if (!state.done) {
      fail();
    }
  });
}
