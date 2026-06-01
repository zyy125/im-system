import { WebSocket } from 'k6/websockets';
import { Counter, Trend } from 'k6/metrics';
import { config } from '../config.js';

export const wsConnectSuccess = new Counter('ws_connect_success_total');
export const wsConnectFailure = new Counter('ws_connect_failure_total');
export const wsMessageSent = new Counter('ws_message_sent_total');
export const wsMessageSentAck = new Counter('ws_message_sent_ack_total');
export const wsMessageCreated = new Counter('ws_message_created_total');
export const wsMessageDeliveredAck = new Counter('ws_message_delivered_ack_total');
export const wsMessageRead = new Counter('ws_message_read_total');
export const wsSyncRequired = new Counter('ws_sync_required_total');
export const wsErrors = new Counter('ws_error_events_total');
export const wsRoundTrip = new Trend('ws_message_round_trip_ms');
export const wsConnectDuration = new Trend('ws_connect_duration_ms');

export function buildWSURL(token) {
  return `${config.wsBaseUrl}/api/v1/ws/?token=${encodeURIComponent(token)}`;
}

export function connect(token, params = {}) {
  const url = buildWSURL(token);
  const headers = {
    ...(params.headers || {}),
  };
  if (config.wsOrigin && !headers.Origin) {
    headers.Origin = config.wsOrigin;
  }
  return new WebSocket(url, [], {
    headers,
    tags: params.tags,
  });
}

export function sendEnvelope(socket, type, data) {
  socket.send(JSON.stringify({ type, data }));
}

export function safeParse(message) {
  try {
    return JSON.parse(message);
  } catch (_) {
    return null;
  }
}

export function closeSocket(socket, code = 1000, reason = 'normal') {
  if (!socket || socket.readyState >= WebSocket.CLOSING) {
    return;
  }
  const safeCode = code === 1000 || (code >= 3000 && code <= 4999) ? code : 4000;
  socket.close(safeCode, reason);
}
