'use strict';

const http = require('node:http');
const port = Number.parseInt(process.env.PLAB_HTTP_PORT || '8080', 10);

const server = http.createServer((request, response) => {
  if (request.url === '/plaintext') {
    response.writeHead(200, { 'content-type': 'text/plain' });
    response.end('Hello, World!');
    return;
  }
  if (request.url === '/json') {
    response.writeHead(200, { 'content-type': 'application/json' });
    response.end('{"message":"Hello, World!"}');
    return;
  }
  if (request.url === '/health') {
    response.writeHead(200, { 'content-type': 'application/json' });
    response.end('{"status":"ok","implementationId":"node-http1","protocol":"h1"}');
    return;
  }
  if (request.url === '/protocol-lab/metadata') {
    response.writeHead(200, { 'content-type': 'application/json' });
    response.end('{"implementationId":"node-http1","packageId":"org.protocol-lab.components.implementation.node-http1","protocol":"h1","protocolVersion":"http/1.1","supportedScenarios":["http1.core.plaintext","http1.core.json"]}');
    return;
  }
  response.writeHead(404, { 'content-type': 'text/plain' });
  response.end('not found');
});

server.listen(port, '0.0.0.0', () => console.log(`node-http1 listening on ${port}`));
