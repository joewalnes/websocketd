// Echo Throughput: single connection, fire-hose sending for 10 seconds.
// Measures maximum messages/sec on a single connection.

import ws from 'k6/ws';
import { Counter, Rate } from 'k6/metrics';
import { check } from 'k6';

const msgsSent = new Counter('ws_msgs_sent');
const msgsRecv = new Counter('ws_msgs_recv');

export const options = {
  scenarios: {
    throughput: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
    },
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;
  const duration = 10000; // 10 seconds

  const res = ws.connect(url, {}, function (socket) {
    let sending = true;

    socket.on('open', () => {
      // Send as fast as possible (1ms is k6's minimum interval)
      socket.setInterval(() => {
        if (sending) {
          for (let i = 0; i < 100; i++) {
            socket.send('x');
            msgsSent.add(1);
          }
        }
      }, 1);

      // Stop after duration
      socket.setTimeout(() => {
        sending = false;
        // Give time for remaining echoes
        socket.setTimeout(() => {
          socket.close();
        }, 2000);
      }, duration);
    });

    socket.on('message', () => {
      msgsRecv.add(1);
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
