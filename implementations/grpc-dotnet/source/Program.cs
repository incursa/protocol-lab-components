using System.IO.Compression;
using System.Security.Authentication;
using Google.Protobuf;
using Grpc.Core;
using Grpc.Net.Compression;
using Microsoft.AspNetCore.Server.Kestrel.Core;
using ProtocolLab.Grpc;

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(18444, listen =>
    {
        listen.Protocols = HttpProtocols.Http2;
        listen.UseHttps(https =>
        {
            https.SslProtocols = SslProtocols.Tls13;
            https.ServerCertificate = System.Security.Cryptography.X509Certificates.X509Certificate2.CreateFromPemFile("/app/certs/leaf.pem", "/app/certs/leaf-key.pem");
        });
    });
});
builder.Services.AddGrpc(options =>
{
    options.ResponseCompressionAlgorithm = "gzip";
    options.ResponseCompressionLevel = CompressionLevel.Fastest;
}).AddServiceOptions<EchoServiceImpl>(options => options.ResponseCompressionAlgorithm = "gzip");
var app = builder.Build();
app.Use(async (context, next) =>
{
    context.Response.OnStarting(() =>
    {
        if (context.Response.ContentType?.StartsWith("application/grpc", StringComparison.OrdinalIgnoreCase) == true)
            context.Response.ContentType = "application/grpc+proto";
        return Task.CompletedTask;
    });
    await next();
});
app.MapGrpcService<EchoServiceImpl>();
Console.WriteLine("{\"implementationId\":\"grpc-dotnet\",\"implementationVersion\":\"0.1.0\",\"protocol\":\"grpc-over-h2\"}");
app.Run();

sealed class EchoServiceImpl : EchoService.EchoServiceBase
{
    private const int Count = 100;

    public override Task<EchoResponse> UnaryEcho(EchoRequest request, ServerCallContext context) => Task.FromResult(Echo(request));

    public override Task<EchoResponse> UnaryGzip(EchoRequest request, ServerCallContext context)
    {
        return Task.FromResult(Echo(request));
    }

    public override async Task<EchoResponse> UnaryFixedMetadata(EchoRequest request, ServerCallContext context)
    {
        var text = context.RequestHeaders.GetValue("x-plab-text");
        var binary = context.RequestHeaders.GetValueBytes("x-plab-bin-bin");
        if (text != "protocol-lab" || binary is null || !binary.SequenceEqual(new byte[] { 0, 1, 2, 3 }))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "fixed request metadata mismatch"));
        await context.WriteResponseHeadersAsync(new Metadata { { "x-plab-text", "protocol-lab" } });
        context.ResponseTrailers.Add("x-plab-bin-bin", new byte[] { 0, 1, 2, 3 });
        return Echo(request);
    }

    public override async Task ServerStreamingEcho(EchoRequest request, IServerStreamWriter<EchoResponse> responseStream, ServerCallContext context)
    {
        for (var i = 0; i < Count; i++) await responseStream.WriteAsync(Echo(request));
    }

    public override async Task<EchoResponse> ClientStreamingEcho(IAsyncStreamReader<EchoRequest> requestStream, ServerCallContext context)
    {
        EchoRequest? last = null;
        var count = 0;
        await foreach (var request in requestStream.ReadAllAsync()) { last = request; count++; }
        if (count != Count || last is null) throw new RpcException(new Status(StatusCode.InvalidArgument, "expected 100 requests"));
        return Echo(last);
    }

    public override async Task BidirectionalStreamingEcho(IAsyncStreamReader<EchoRequest> requestStream, IServerStreamWriter<EchoResponse> responseStream, ServerCallContext context)
    {
        var count = 0;
        await foreach (var request in requestStream.ReadAllAsync()) { await responseStream.WriteAsync(Echo(request)); count++; }
        if (count != Count) throw new RpcException(new Status(StatusCode.InvalidArgument, "expected 100 requests"));
    }

    public override Task<EchoResponse> TrailersOnlyStatus(EchoRequest request, ServerCallContext context) =>
        throw new RpcException(new Status(StatusCode.InvalidArgument, "plab invalid fixture"));

    public override async Task<EchoResponse> DeadlineExceeded(EchoRequest request, ServerCallContext context)
    {
        await Task.Delay(250);
        return Echo(request);
    }

    public override async Task ClientCancellation(EchoRequest request, IServerStreamWriter<EchoResponse> responseStream, ServerCallContext context)
    {
        await context.WriteResponseHeadersAsync(new Metadata { { "x-plab-ready", "1" } });
        await Task.Delay(Timeout.Infinite, context.CancellationToken);
    }

    private static EchoResponse Echo(EchoRequest request) => new() { Payload = ByteString.CopyFrom(request.Payload.Span) };
}
