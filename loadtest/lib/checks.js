import { check } from 'k6';

export function checkOK(label, res, extraChecks = {}) {
  return check(res, {
    [`${label}: http status < 400`]: (r) => r.status < 400,
    [`${label}: response code ok`]: (r) => {
      try {
        const body = r.json();
        return body && body.code === 'ok';
      } catch (_) {
        return false;
      }
    },
    ...extraChecks,
  });
}

export function checkWS(label, conditionMap) {
  return check(conditionMap, Object.fromEntries(Object.keys(conditionMap).map((key) => [`${label}: ${key}`, () => conditionMap[key]])));
}
