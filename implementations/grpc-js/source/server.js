'use strict';
const fs = require('fs');
const path = require('path');
const grpc = require('@grpc/grpc-js');
const loader = require('@grpc/proto-loader');
const root = '/app';
const def = loader.loadSync(path.join(root, 'contract/echo.proto'), {keepCase: false, longs: String, enums: String, defaults: true, oneofs: true});
const service = grpc.loadPackageDefinition(def).protocollab.performance.v1.EchoService;
const COUNT = 100;
const echo = request => ({payload: request.payload});
const handlers = {
  unaryEcho(call, callback) { callback(null, echo(call.request)); },
  unaryGzip(call, callback) { callback(null, echo(call.request)); },
  unaryFixedMetadata(call, callback) {
    const initial = new grpc.Metadata(); initial.set('x-plab-text', 'protocol-lab'); call.sendMetadata(initial);
    const trailers = new grpc.Metadata(); trailers.set('x-plab-bin-bin', Buffer.from([0,1,2,3]));
    callback(null, echo(call.request), trailers);
  },
  serverStreamingEcho(call) { for (let i=0;i<COUNT;i++) call.write(echo(call.request)); call.end(); },
  clientStreamingEcho(call, callback) {
    let last; let count=0; call.on('data', request => {last=request; count++;});
    call.on('end', () => count === COUNT ? callback(null, echo(last)) : callback({code: grpc.status.INVALID_ARGUMENT, details: 'expected 100 requests'}));
  },
  bidirectionalStreamingEcho(call) {
    let count=0; call.on('data', request => {count++; call.write(echo(request));});
    call.on('end', () => count === COUNT ? call.end() : call.destroy({code: grpc.status.INVALID_ARGUMENT, details: 'expected 100 requests'}));
  },
  trailersOnlyStatus(call, callback) { callback({code: grpc.status.INVALID_ARGUMENT, details: 'plab invalid fixture'}); },
  deadlineExceeded(call, callback) { setTimeout(() => callback(null, echo(call.request)), 250); },
  clientCancellation(call) { const initial = new grpc.Metadata(); initial.set('x-plab-ready', '1'); call.sendMetadata(initial); call.on('cancelled', () => call.end()); }
};
const server = new grpc.Server({'grpc.default_compression_algorithm': 2});
server.addService(service.service, handlers);
const credentials = grpc.ServerCredentials.createSsl(null, [{private_key: fs.readFileSync(path.join(root,'certs/leaf-key.pem')), cert_chain: fs.readFileSync(path.join(root,'certs/leaf.pem'))}], false);
server.bindAsync('0.0.0.0:18444', credentials, (error, port) => {
  if (error) throw error;
  console.log(JSON.stringify({implementationId:'grpc-js',implementationVersion:'0.1.0',listenPort:port,protocol:'grpc-over-h2'}));
});
