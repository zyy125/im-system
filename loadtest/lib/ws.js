import ws from 'k6/ws';
import { Counter, Trend } from 'k6/metrics';
import { config } from '../config.js';

export const wsConnectSuccess = new Counter('ws_connect_success_total');
export const wsConnectFailure = new Counter('ws_connect_failure_total');
export const wsMessageSent = new Counter('ws_message_sent_total');
export const wsMessageCreated = new Counter('ws_message_created_total');
export const wsMessageDelivered = new Counter('ws_message_delivered_total');
export const wsMessageRead = new Counter('ws_message_read_total');
export const wsSyncRequired = new Counter('ws_sync_required_total');
export const wsErrors = new Counter('ws_error_events_total');
export const wsRoundTrip = new Trend('ws_message_round_trip_ms');

export function buildWSURL(token) {
  return `${config.wsBaseUrl}/api/v1/ws/?token=${encodeURIComponent(token)}`;
}

export function connect(token, handlers, params = {}) {
  const url = buildWSURL(token);
  const response = ws.connect(
    url,
    {
      tags: params.tags,
      headers: params.headers || {},
      timeout: params.timeout || '15s',
    },
    handlers,
  );
  return response;
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
