// Echo Latency: single connection, sequential send/recv to measure round-trip time.
// Sends a timestamp, waits for echo, computes RTT. Repeats 1000 times.

import ws from 'k6/ws';
import { Trend } from 'k6/metrics';
import { check } from 'k6';

const rtt = new Trend('ws_rtt_ms', true);

export const options = {
  scenarios: {
    latency: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
    },
  },
  thresholds: {
    ws_rtt_ms: ['p(95)<50'],
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;
  const total = 1000;

  const res = ws.connect(url, {}, function (socket) {
    let count = 0;

    socket.on('open', () => {
      socket.send(String(Date.now()));
    });

    socket.on('message', (data) => {
      const sent = Number(data);
      rtt.add(Date.now() - sent);
      count++;
      if (count < total) {
        socket.send(String(Date.now()));
      } else {
        socket.close();
      }
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });

    socket.setTimeout(() => {
      console.error('Timeout after 60s');
      socket.close();
    }, 60000);
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
