import exec from 'k6/execution';
import { Counter } from 'k6/metrics';
import { uuidv4 as randomUUID } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { config, ensureDedicatedChatPairs, envNumber, requirePair } from '../config.js';
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
  wsMessageRead,
  wsMessageSent,
  wsMessageSentAck,
  wsRoundTrip,
  wsSyncRequired,
} from '../lib/ws.js';

const configuredChatVUs = envNumber('CHAT_VUS', 8);
const configuredTotalConnections = envNumber('TOTAL_CONNECTIONS', configuredChatVUs * 2);
const configuredIdleWsVUs = Math.max(0, configuredTotalConnections - configuredChatVUs * 2);
const configuredRampUp = __ENV.RAMP_UP || '30s';
const configuredHold = __ENV.HOLD || '2m';
const configuredRampDown = __ENV.RAMP_DOWN || '15s';
const configuredDuration = __ENV.DURATION || '3m';
const configuredSocketLifetimeMs = envNumber('SOCKET_LIFETIME_MS', 20000);
const configuredIdleSocketLifetimeSeconds = envNumber('IDLE_SOCKET_LIFETIME_SECONDS', 60);
const configuredMessagesPerSession = Math.max(1, envNumber('CHAT_MESSAGES_PER_SESSION', 5));

const openedConversationCache = new Map();
const mixedChatFlowFailure = new Counter('mixed_chat_flow_failure_total');

function buildIdleUsers(pairs, reservedPairCount) {
  const reserved = new Set();
  for (let i = 0; i < reservedPairCount; i += 1) {
    const pair = requirePair(pairs[i]);
    reserved.add(pair.username);
    reserved.add(pair.peer.username);
  }

  const seen = new Set();
  const idleUsers = [];
  for (const pair of pairs) {
    const candidates = [
      { username: pair.username, password: pair.password },
      { username: pair.peer.username, password: pair.peer.password },
    ];
    for (const candidate of candidates) {
      if (!candidate.username || reserved.has(candidate.username) || seen.has(candidate.username)) {
        continue;
      }
      seen.add(candidate.username);
      idleUsers.push(candidate);
    }
  }

  return idleUsers;
}

const idleUsers = buildIdleUsers(config.users, configuredChatVUs);

function indexByUsername(users) {
  const result = new Map();
  for (const user of users) {
    result.set(user.username, user);
    if (user.peer?.username) {
      result.set(user.peer.username, {
        username: user.peer.username,
        password: user.peer.password,
      });
    }
  }
  return result;
}

const userByUsername = indexByUsername(config.users);

function chatUserByVU() {
  const index = (exec.vu.idInTest - 1) % configuredChatVUs;
  return requirePair(config.users[index]);
}

function idleUserByVU() {
  const index = (exec.vu.idInTest - 1) % idleUsers.length;
  return idleUsers[index];
}

function openConversation(token, conversationId) {
  return expectOK(
    'open-conversation',
    apiPost(`/api/v1/conversations/${conversationId}/open`, token, null, {
      tags: { scenario_group: 'mixed_connections', endpoint: 'conversation_open' },
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
      tags: { scenario_group: 'mixed_connections', endpoint: 'message_sync' },
    }),
  );
}

export const options = {
  setupTimeout: __ENV.SETUP_TIMEOUT || '10m',
  scenarios: {
    idle_ws: {
      executor: 'ramping-vus',
      exec: 'idleWs',
      startVUs: 0,
      stages: [
        { duration: configuredRampUp, target: configuredIdleWsVUs },
        { duration: configuredHold, target: configuredIdleWsVUs },
        { duration: configuredRampDown, target: 0 },
      ],
      gracefulRampDown: '5s',
    },
    ws_chat_flow: {
      executor: 'ramping-vus',
      exec: 'wsChatFlow',
      startVUs: 0,
      stages: [
        { duration: configuredRampUp, target: configuredChatVUs },
        { duration: configuredHold, target: configuredChatVUs },
        { duration: configuredRampDown, target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    ...config.thresholds,
    ws_error_events_total: ['count==0'],
    ws_sync_required_total: ['count<5'],
    mixed_chat_flow_failure_total: ['count==0'],
    ws_message_sent_ack_total: [`count>=${configuredChatVUs * 0.9}`],
    ws_message_created_total: [`count>=${configuredChatVUs * 0.9}`],
    ws_message_read_total: [`count>=${configuredChatVUs * 0.9}`],
    ws_message_round_trip_ms: ['p(95)<1500'],
  },
};

export function setup() {
  if (configuredTotalConnections < configuredChatVUs * 2) {
    throw new Error('TOTAL_CONNECTIONS must be at least CHAT_VUS * 2');
  }
  ensureDedicatedChatPairs(configuredChatVUs, 'mixed-dev');
  if (configuredIdleWsVUs > idleUsers.length) {
    throw new Error(
      `TOTAL_CONNECTIONS=${configuredTotalConnections} with CHAT_VUS=${configuredChatVUs} requires ${configuredIdleWsVUs} dedicated idle users, but only ${idleUsers.length} are available`,
    );
  }

  const tokenByUsername = {};
  const requiredUsernames = new Set();

  for (let i = 0; i < configuredChatVUs; i += 1) {
    const pair = requirePair(config.users[i]);
    requiredUsernames.add(pair.username);
    requiredUsernames.add(pair.peer.username);
  }

  for (let i = 0; i < configuredIdleWsVUs; i += 1) {
    requiredUsernames.add(idleUsers[i].username);
  }

  for (const username of requiredUsernames) {
    const user = userByUsername.get(username);
    if (!user) {
      throw new Error(`missing user config for username=${username}`);
    }
    const tokens = login(user.username, user.password);
    tokenByUsername[username] = tokens.access_token;
  }

  return {
    tokenByUsername,
  };
}

export function idleWs(data) {
  const user = idleUserByVU();
  const token = data.tokenByUsername[user.username];
  const socket = connect(token, {
    tags: { scenario_group: 'mixed_connections', endpoint: 'idle_ws' },
  });
  const timeoutId = setTimeout(() => {
    closeSocket(socket);
  }, configuredIdleSocketLifetimeSeconds * 1000);

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
  socket.addEventListener('error', recordFailure);
  socket.addEventListener('close', () => {
    clearTimeout(timeoutId);
    if (!opened) {
      recordFailure();
    }
  });
}

export function wsChatFlow(data) {
  const user = chatUserByVU();
  const selfToken = data.tokenByUsername[user.username];
  const peerToken = data.tokenByUsername[user.peer.username];
  const conversationId = user.peer.conversation_id || user.conversation_id;

  if (!conversationId) {
    throw new Error('mixed-dev requires conversation_id in test user data');
  }

  if (!selfToken || !peerToken) {
    throw new Error('missing preloaded access token for chat flow');
  }

  ensureConversationOpen(selfToken, conversationId);
  ensureConversationOpen(peerToken, conversationId);

  const state = {
    selfOpen: false,
    peerOpen: false,
    sending: false,
    pendingMsgId: '',
    pendingSeq: 0,
    sentAcked: false,
    createdSeen: false,
    readSeen: false,
    messagesCompleted: 0,
    done: false,
  };
  const startedAt = Date.now();

  const selfSocket = connect(selfToken, {
    tags: { scenario_group: 'mixed_connections', endpoint: 'chat_self_ws' },
  });
  const peerSocket = connect(peerToken, {
    tags: { scenario_group: 'mixed_connections', endpoint: 'chat_peer_ws' },
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
    mixedChatFlowFailure.add(1);
    state.done = true;
    clearScenarioTimeout();
    closeSocket(selfSocket, 4000, 'scenario_failed');
    closeSocket(peerSocket, 4000, 'scenario_failed');
  }

  function startNextMessage() {
    if (state.done || state.sending || !state.selfOpen || !state.peerOpen) {
      return;
    }
    if (state.messagesCompleted >= configuredMessagesPerSession) {
      complete();
      return;
    }

    const msgId = randomUUID();
    state.pendingMsgId = msgId;
    state.pendingSeq = 0;
    state.sentAcked = false;
    state.createdSeen = false;
    state.readSeen = false;
    state.sending = true;

    sendEnvelope(selfSocket, 'message.send', {
      msg_id: msgId,
      conversation_id: conversationId,
      content: `mixed-${msgId}`,
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
    state.pendingSeq = 0;

    if (state.messagesCompleted >= configuredMessagesPerSession) {
      complete();
      return;
    }

    startNextMessage();
  }

  timeoutId = setTimeout(fail, configuredSocketLifetimeMs);

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
      syncConversation(selfToken, conversationId, state.pendingSeq);
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
      syncConversation(peerToken, conversationId, state.pendingSeq);
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
