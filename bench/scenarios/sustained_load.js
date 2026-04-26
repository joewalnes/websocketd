// Sustained Load: N connections sending continuously for 30 seconds.
// Measures steady-state throughput and latency under load.
// Set SUSTAINED_VUS env var to control concurrency (default 50).

import ws from 'k6/ws';
import { Trend, Counter } from 'k6/metrics';
import { check } from 'k6';

const rtt = new Trend('ws_sustained_rtt_ms', true);
const msgsRecv = new Counter('ws_sustained_msgs');

const vus = Number(__ENV.SUSTAINED_VUS || 50);

export const options = {
  scenarios: {
    sustained: {
      executor: 'constant-vus',
      vus: vus,
      duration: '30s',
    },
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      socket.setInterval(() => {
        socket.send(String(Date.now()));
      }, 10); // ~100 msgs/sec per VU
    });

    socket.on('message', (data) => {
      const sent = Number(data);
      if (!isNaN(sent) && sent > 0) {
        rtt.add(Date.now() - sent);
      }
      msgsRecv.add(1);
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });

    socket.setTimeout(() => {
      socket.close();
    }, 35000);
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
