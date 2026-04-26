// Connection Churn: serial connect/send/recv/close cycles.
// Measures connections/sec (process startup + teardown overhead).

import ws from 'k6/ws';
import { Trend } from 'k6/metrics';
import { check } from 'k6';

const cycleTime = new Trend('ws_churn_cycle_ms', true);

export const options = {
  scenarios: {
    churn: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 200,
    },
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;
  const start = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      socket.send('ping');
    });

    socket.on('message', () => {
      socket.close();
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });

    socket.setTimeout(() => {
      socket.close();
    }, 10000);
  });

  cycleTime.add(Date.now() - start);
  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
