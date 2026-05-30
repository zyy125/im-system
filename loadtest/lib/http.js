import http from 'k6/http';
import { checkOK } from './checks.js';
import { config } from '../config.js';

function buildHeaders(token, extra = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...extra,
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

export function apiGet(path, token, params = {}) {
  return http.get(`${config.baseUrl}${path}`, {
    headers: buildHeaders(token),
    tags: params.tags,
    timeout: params.timeout || '10s',
  });
}

export function apiPost(path, token, payload, params = {}) {
  return http.post(`${config.baseUrl}${path}`, JSON.stringify(payload), {
    headers: buildHeaders(token),
    tags: params.tags,
    timeout: params.timeout || '10s',
  });
}

export function parseJSON(res) {
  return res.json();
}

export function expectOK(label, res, extraChecks = {}) {
  checkOK(label, res, extraChecks);
  return parseJSON(res);
}
