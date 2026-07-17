package org.protocollab;

import com.google.protobuf.ByteString;
import io.grpc.*;
import io.grpc.netty.shaded.io.grpc.netty.GrpcSslContexts;
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder;
import io.grpc.stub.ServerCallStreamObserver;
import io.grpc.stub.StreamObserver;
import protocollab.performance.v1.Echo;
import protocollab.performance.v1.EchoServiceGrpc;
import java.io.File;
import java.util.concurrent.CountDownLatch;

public final class GrpcServer {
  static final int COUNT=100;
  static final Metadata.Key<String> TEXT=Metadata.Key.of("x-plab-text", Metadata.ASCII_STRING_MARSHALLER);
  static final Metadata.Key<byte[]> BIN=Metadata.Key.of("x-plab-bin-bin", Metadata.BINARY_BYTE_MARSHALLER);
  static final Metadata.Key<String> READY=Metadata.Key.of("x-plab-ready", Metadata.ASCII_STRING_MARSHALLER);
  static final Context.Key<Metadata> REQUEST_HEADERS=Context.key("plab-request-headers");
  @SuppressWarnings("rawtypes") static final Context.Key<ServerCall> ACTIVE_CALL=Context.key("plab-active-call");

  public static void main(String[] args) throws Exception {
    var ssl=GrpcSslContexts.forServer(new File("/app/certs/leaf.pem"),new File("/app/certs/leaf-key-pkcs8.pem")).protocols("TLSv1.3").build();
    var server=NettyServerBuilder.forPort(18444).sslContext(ssl).addService(ServerInterceptors.intercept(new Service(),new MetadataInterceptor())).build().start();
    System.out.println("{\"implementationId\":\"grpc-java-netty\",\"implementationVersion\":\"0.1.2\",\"protocol\":\"grpc-over-h2\"}");
    server.awaitTermination();
  }

  static final class MetadataInterceptor implements ServerInterceptor {
    public <ReqT,RespT> ServerCall.Listener<ReqT> interceptCall(ServerCall<ReqT,RespT> call,Metadata headers,ServerCallHandler<ReqT,RespT> next){
      String method=call.getMethodDescriptor().getBareMethodName();
      var wrapped=new ForwardingServerCall.SimpleForwardingServerCall<ReqT,RespT>(call){
        @Override public void sendHeaders(Metadata response){if("UnaryFixedMetadata".equals(method))response.put(TEXT,"protocol-lab");if("ClientCancellation".equals(method))response.put(READY,"1");super.sendHeaders(response);}
        @Override public void close(Status status,Metadata trailers){if("UnaryFixedMetadata".equals(method))trailers.put(BIN,new byte[]{0,1,2,3});super.close(status,trailers);}
      };
      return Contexts.interceptCall(Context.current().withValue(REQUEST_HEADERS,headers).withValue(ACTIVE_CALL,wrapped),wrapped,headers,next);
    }
  }

  static final class Service extends EchoServiceGrpc.EchoServiceImplBase {
    static Echo.EchoResponse echo(Echo.EchoRequest r){return Echo.EchoResponse.newBuilder().setPayload(r.getPayload()).build();}
    @Override public void unaryEcho(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){o.onNext(echo(r));o.onCompleted();}
    @Override public void unaryGzip(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){((ServerCallStreamObserver<?>)o).setCompression("gzip");o.onNext(echo(r));o.onCompleted();}
    @Override public void unaryFixedMetadata(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){var h=REQUEST_HEADERS.get();if(h==null||!"protocol-lab".equals(h.get(TEXT))||!java.util.Arrays.equals(h.get(BIN),new byte[]{0,1,2,3})){o.onError(Status.INVALID_ARGUMENT.withDescription("fixed request metadata mismatch").asRuntimeException());return;}o.onNext(echo(r));o.onCompleted();}
    @Override public void serverStreamingEcho(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){for(int i=0;i<COUNT;i++)o.onNext(echo(r));o.onCompleted();}
    @Override public StreamObserver<Echo.EchoRequest> clientStreamingEcho(StreamObserver<Echo.EchoResponse> o){return new StreamObserver<>(){Echo.EchoRequest last;int count;public void onNext(Echo.EchoRequest r){last=r;count++;}public void onError(Throwable t){}public void onCompleted(){if(count!=COUNT)o.onError(Status.INVALID_ARGUMENT.asRuntimeException());else{o.onNext(echo(last));o.onCompleted();}}};}
    @Override public StreamObserver<Echo.EchoRequest> bidirectionalStreamingEcho(StreamObserver<Echo.EchoResponse> o){return new StreamObserver<>(){int count;public void onNext(Echo.EchoRequest r){count++;o.onNext(echo(r));}public void onError(Throwable t){}public void onCompleted(){if(count!=COUNT)o.onError(Status.INVALID_ARGUMENT.asRuntimeException());else o.onCompleted();}};}
    @Override public void trailersOnlyStatus(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){o.onError(Status.INVALID_ARGUMENT.withDescription("plab invalid fixture").asRuntimeException());}
    @Override public void deadlineExceeded(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){try{Thread.sleep(250);}catch(InterruptedException ignored){}if(!((ServerCallStreamObserver<?>)o).isCancelled()){o.onNext(echo(r));o.onCompleted();}}
    @Override public void clientCancellation(Echo.EchoRequest r,StreamObserver<Echo.EchoResponse> o){var server=(ServerCallStreamObserver<Echo.EchoResponse>)o;server.setOnCancelHandler(()->{});var call=ACTIVE_CALL.get();if(call!=null)call.sendHeaders(new Metadata());}
  }
}
