'use strict';

const http2 = require('node:http2');
const port = Number.parseInt(process.env.PLAB_HTTP_PORT || '8082', 10);
const server = http2.createServer();

server.on('stream', (stream, headers) => {
  const path = headers[':path'];
  if (path === '/plaintext') {
    stream.respond({ ':status': 200, 'content-type': 'text/plain' });
    stream.end('Hello, World!');
    return;
  }
  if (path === '/json') {
    stream.respond({ ':status': 200, 'content-type': 'application/json' });
    stream.end('{"message":"Hello, World!"}');
    return;
  }
  if (path === '/health') {
    stream.respond({ ':status': 200, 'content-type': 'application/json' });
    stream.end('{"status":"ok","implementationId":"node-http2","protocol":"h2"}');
    return;
  }
  if (path === '/protocol-lab/metadata') {
    stream.respond({ ':status': 200, 'content-type': 'application/json' });
    stream.end('{"implementationId":"node-http2","packageId":"org.protocol-lab.components.implementation.node-http2","protocol":"h2","protocolVersion":"h2c","supportedScenarios":["http2.core.plaintext","http2.core.json"]}');
    return;
  }
  stream.respond({ ':status': 404, 'content-type': 'text/plain' });
  stream.end('not found');
});

server.listen(port, '0.0.0.0', () => console.log(`node-http2 listening on ${port}`));
