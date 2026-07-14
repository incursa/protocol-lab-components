import { WebSocketServer } from 'ws';

if (process.argv.includes('--version')) {
  process.stdout.write('node-ws-websocket 0.1.0 (ws 8.21.0)\n');
  process.exit(0);
}

const port = Number.parseInt(process.env.PLAB_TARGET_PORT ?? '18081', 10);
if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error(`Invalid PLAB_TARGET_PORT: ${process.env.PLAB_TARGET_PORT}`);
}

const server = new WebSocketServer({
  host: '0.0.0.0',
  port,
  path: '/websocket',
  perMessageDeflate: false,
  clientTracking: false,
  maxPayload: 1024 * 1024
});

server.on('connection', (socket) => {
  socket.on('message', (data, isBinary) => {
    socket.send(data, { binary: isBinary, compress: false });
  });
});

server.on('listening', () => {
  process.stdout.write(JSON.stringify({
    status: 'ready',
    implementationId: 'node-ws-websocket',
    version: '0.1.0',
    upstream: { name: 'ws', version: '8.21.0' },
    protocol: 'h1',
    protocolVersion: 'HTTP/1.1',
    protocolVariant: 'websocket-h1-cleartext-upgrade',
    transportSecurity: 'cleartext',
    port
  }) + '\n');
});

const stop = () => server.close(() => process.exit(0));
process.on('SIGTERM', stop);
process.on('SIGINT', stop);
