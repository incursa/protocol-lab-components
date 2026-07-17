#include <grpcpp/grpcpp.h>
#include "echo.grpc.pb.h"
#include <chrono>
#include <fstream>
#include <iostream>
#include <sstream>
#include <thread>

using grpc::ServerContext;
using grpc::Status;
using grpc::StatusCode;
using protocollab::performance::v1::EchoRequest;
using protocollab::performance::v1::EchoResponse;
using protocollab::performance::v1::EchoService;

static std::string ReadFile(const char* path){std::ifstream f(path);std::ostringstream s;s<<f.rdbuf();return s.str();}
static void Echo(const EchoRequest& request,EchoResponse* response){response->set_payload(request.payload());}

class Service final: public EchoService::Service {
 public:
  Status UnaryEcho(ServerContext*,const EchoRequest* request,EchoResponse* response) override {Echo(*request,response);return Status::OK;}
  Status UnaryGzip(ServerContext* context,const EchoRequest* request,EchoResponse* response) override {context->set_compression_algorithm(GRPC_COMPRESS_GZIP);Echo(*request,response);return Status::OK;}
  Status UnaryFixedMetadata(ServerContext* context,const EchoRequest* request,EchoResponse* response) override {
    auto text=context->client_metadata().find("x-plab-text");auto bin=context->client_metadata().find("x-plab-bin-bin");
    if(text==context->client_metadata().end()||std::string(text->second.data(),text->second.size())!="protocol-lab"||bin==context->client_metadata().end()||std::string(bin->second.data(),bin->second.size())!=std::string("\0\1\2\3",4))return Status(StatusCode::INVALID_ARGUMENT,"fixed request metadata mismatch");
    context->AddInitialMetadata("x-plab-text","protocol-lab");context->AddTrailingMetadata("x-plab-bin-bin",std::string("\0\1\2\3",4));Echo(*request,response);return Status::OK;
  }
  Status ServerStreamingEcho(ServerContext*,const EchoRequest* request,grpc::ServerWriter<EchoResponse>* writer) override {EchoResponse response;Echo(*request,&response);for(int i=0;i<100;i++)if(!writer->Write(response))break;return Status::OK;}
  Status ClientStreamingEcho(ServerContext*,grpc::ServerReader<EchoRequest>* reader,EchoResponse* response) override {EchoRequest request;int count=0;while(reader->Read(&request)){Echo(request,response);count++;}return count==100?Status::OK:Status(StatusCode::INVALID_ARGUMENT,"expected 100 requests");}
  Status BidirectionalStreamingEcho(ServerContext*,grpc::ServerReaderWriter<EchoResponse,EchoRequest>* stream) override {EchoRequest request;EchoResponse response;int count=0;while(stream->Read(&request)){Echo(request,&response);if(!stream->Write(response))break;count++;}return count==100?Status::OK:Status(StatusCode::INVALID_ARGUMENT,"expected 100 requests");}
  Status TrailersOnlyStatus(ServerContext*,const EchoRequest*,EchoResponse*) override {return Status(StatusCode::INVALID_ARGUMENT,"plab invalid fixture");}
  Status DeadlineExceeded(ServerContext* context,const EchoRequest* request,EchoResponse* response) override {std::this_thread::sleep_for(std::chrono::milliseconds(250));if(context->IsCancelled())return Status(StatusCode::CANCELLED,"");Echo(*request,response);return Status::OK;}
  Status ClientCancellation(ServerContext* context,const EchoRequest*,grpc::ServerWriter<EchoResponse>* writer) override {context->AddInitialMetadata("x-plab-ready","1");writer->SendInitialMetadata();while(!context->IsCancelled())std::this_thread::sleep_for(std::chrono::milliseconds(2));return Status(StatusCode::CANCELLED,"");}
};

int main(){grpc::SslServerCredentialsOptions ssl;ssl.pem_key_cert_pairs.push_back({ReadFile("/app/certs/leaf-key.pem"),ReadFile("/app/certs/leaf.pem")});Service service;grpc::ServerBuilder builder;builder.AddListeningPort("0.0.0.0:18444",grpc::SslServerCredentials(ssl));builder.RegisterService(&service);auto server=builder.BuildAndStart();if(!server){std::cerr<<"server start failed\n";return 1;}std::cout<<"{\"implementationId\":\"grpc-cpp\",\"implementationVersion\":\"0.1.2\",\"protocol\":\"grpc-over-h2\"}"<<std::endl;server->Wait();}
