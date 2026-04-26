// Backpressure: fast sender, slow consumer backend.
// The backend (slow-consumer.sh) echoes with 100ms delay per line.
// Tests that websocketd handles backpressure gracefully.

import ws from 'k6/ws';
import { Counter, Trend } from 'k6/metrics';
import { check } from 'k6';

const msgsSent = new Counter('ws_bp_msgs_sent');
const msgsRecv = new Counter('ws_bp_msgs_recv');
const rtt = new Trend('ws_bp_rtt_ms', true);

export const options = {
  scenarios: {
    backpressure: {
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
      // Send rapidly — backend can only process ~10/sec
      socket.setInterval(() => {
        if (sending) {
          for (let i = 0; i < 10; i++) {
            socket.send(String(Date.now()));
            msgsSent.add(1);
          }
        }
      }, 10);

      socket.setTimeout(() => {
        sending = false;
        // Wait for remaining echoes
        socket.setTimeout(() => {
          socket.close();
        }, 5000);
      }, duration);
    });

    socket.on('message', (data) => {
      msgsRecv.add(1);
      const sent = Number(data);
      if (!isNaN(sent) && sent > 0) {
        rtt.add(Date.now() - sent);
      }
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
