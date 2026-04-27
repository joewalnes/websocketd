// Connection Storm: N concurrent connections, each sends 1 message and closes.
// Set STORM_VUS env var to control number of concurrent connections (default 100).

import ws from 'k6/ws';
import { Trend } from 'k6/metrics';
import { check } from 'k6';

const connectTime = new Trend('ws_connect_time_ms', true);
const cycleTime = new Trend('ws_cycle_time_ms', true);

const vus = Number(__ENV.STORM_VUS || 100);

export const options = {
  scenarios: {
    storm: {
      executor: 'shared-iterations',
      vus: vus,
      iterations: vus,
      maxDuration: '60s',
    },
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;
  const start = Date.now();

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      connectTime.add(Date.now() - start);
      socket.send('ping');
    });

    socket.on('message', () => {
      cycleTime.add(Date.now() - start);
      socket.close();
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });

    socket.setTimeout(() => {
      socket.close();
    }, 30000);
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
