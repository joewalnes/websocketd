// Binary Throughput: binary mode, various payload sizes.
// Measures bytes/sec for 1KB, 10KB, 64KB, 256KB payloads.

import ws from 'k6/ws';
import { Trend, Counter } from 'k6/metrics';
import { check } from 'k6';

const bytesRecv = new Counter('ws_binary_bytes_recv');
const rtt = new Trend('ws_binary_rtt_ms', true);

const payloadSize = Number(__ENV.PAYLOAD_SIZE || 10240);

export const options = {
  scenarios: {
    binary: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
    },
  },
};

export default function () {
  const url = `ws://127.0.0.1:${__ENV.WS_PORT}/`;
  const count = 100;

  const res = ws.connect(url, {}, function (socket) {
    let sent = 0;
    let recv = 0;
    const payload = new ArrayBuffer(payloadSize);

    socket.on('open', () => {
      sendNext();
    });

    function sendNext() {
      if (sent < count) {
        socket.sendBinary(payload);
        sent++;
      }
    }

    socket.on('binaryMessage', (data) => {
      bytesRecv.add(data.byteLength || data.length);
      recv++;
      if (recv >= count) {
        socket.close();
      } else {
        sendNext();
      }
    });

    // Also handle text messages in case server echoes as text
    socket.on('message', (data) => {
      bytesRecv.add(data.length);
      recv++;
      if (recv >= count) {
        socket.close();
      } else {
        sendNext();
      }
    });

    socket.on('error', (e) => {
      console.error('WebSocket error:', e.error());
    });

    socket.setTimeout(() => {
      console.error(`Timeout: sent=${sent} recv=${recv}`);
      socket.close();
    }, 60000);
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
