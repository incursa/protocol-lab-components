using System.Buffers.Binary;
using System.Net;
using System.Security.Authentication;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using System.Text.Json;
using Microsoft.AspNetCore.Http.Features;
using Microsoft.AspNetCore.Server.Kestrel.Core;

const string ImplementationId = "kestrel-http2-websocket";
const string Version = "0.1.0";
const string Authority = "websocket.plab.test";
const string PathValue = "/websocket";
const string TextPayload = "protocol-lab";
const string ControlPayload = "protocol-lab-ping";
var supported = new HashSet<string>(StringComparer.Ordinal)
{
    "http2.websocket.rfc8441.extended-connect",
    "http2.websocket.rfc8441.control-frames",
    "http2.websocket.rfc8441.text-echo",
    "http2.websocket.rfc8441.binary-echo",
    "http2.websocket.rfc8441.close",
    "http2.websocket.rfc8441.multi-message-text-echo"
};

var requestedScenario = Environment.GetEnvironmentVariable("PLAB_SCENARIO_ID");
if (!string.IsNullOrWhiteSpace(requestedScenario) && !supported.Contains(requestedScenario))
    throw new InvalidOperationException($"Unsupported scenario identity {requestedScenario}");

var packageRoot = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "..", ".."));
var certPath = Environment.GetEnvironmentVariable("PLAB_TLS_CERTIFICATE_PATH") ?? Path.Combine(packageRoot, "certs", "leaf.pem");
var keyPath = Environment.GetEnvironmentVariable("PLAB_TLS_PRIVATE_KEY_PATH") ?? Path.Combine(packageRoot, "certs", "leaf-key.pem");
var certificate = X509Certificate2.CreateFromPemFile(certPath, keyPath);
certificate = X509CertificateLoader.LoadPkcs12(certificate.Export(X509ContentType.Pkcs12), null);
var listen = Environment.GetEnvironmentVariable("PLAB_LISTEN_ADDRESS") ?? "127.0.0.1:18451";
var endpoint = IPEndPoint.Parse(listen);

var builder = WebApplication.CreateSlimBuilder(args);
builder.WebHost.ConfigureKestrel(options =>
{
    options.Listen(endpoint, listenOptions =>
    {
        listenOptions.Protocols = HttpProtocols.Http2;
        listenOptions.UseHttps(https =>
        {
            https.ServerCertificate = certificate;
            https.SslProtocols = SslProtocols.Tls13;
        });
    });
});
var app = builder.Build();

app.Run(async context =>
{
    try
    {
        await HandleRequest(context);
    }
    catch (Exception exception)
    {
        Console.Error.WriteLine(JsonSerializer.Serialize(new { eventName = "request-rejected", message = exception.Message }));
        if (!context.Response.HasStarted) context.Response.StatusCode = StatusCodes.Status400BadRequest;
    }
});

app.Lifetime.ApplicationStarted.Register(() => Console.WriteLine(JsonSerializer.Serialize(new
{
    eventName = "ready", implementationId = ImplementationId, implementationVersion = Version,
    listenAddress = listen, protocol = "h2", protocolVersion = "HTTP/2", protocolVariant = "websocket-h2-extended-connect",
    transportSecurity = "tls", tlsVersion = "TLS1.3", alpn = "h2", authority = Authority, path = PathValue,
    settingsEnableConnectProtocol = 1, certificateDerSha256 = Convert.ToHexString(SHA256.HashData(certificate.RawData)).ToLowerInvariant(),
    certificateSpkiSha256 = Convert.ToHexString(SHA256.HashData(certificate.PublicKey.ExportSubjectPublicKeyInfo())).ToLowerInvariant(),
    supportedScenarios = supported.Order().ToArray()
})));
await app.RunAsync();

static async Task HandleRequest(HttpContext context)
{
    var feature = context.Features.Get<IHttpExtendedConnectFeature>() ?? throw new InvalidOperationException("extended CONNECT feature unavailable");
    if (!feature.IsExtendedConnect || feature.Protocol != "websocket") throw new InvalidOperationException("request is not RFC 8441 websocket extended CONNECT");
    if (context.Request.Protocol != "HTTP/2" || context.Request.Method != "CONNECT" || context.Request.Scheme != "https" || context.Request.Host.Value != Authority || context.Request.Path != PathValue)
        throw new InvalidOperationException("extended CONNECT pseudo-header projection mismatch");
    if (context.Request.Headers["sec-websocket-version"] != "13") throw new InvalidOperationException("sec-websocket-version mismatch");
    foreach (var prohibited in new[] { "connection", "upgrade", "sec-websocket-key", "sec-websocket-protocol", "sec-websocket-extensions", "origin" })
        if (context.Request.Headers.ContainsKey(prohibited)) throw new InvalidOperationException($"prohibited request header present: {prohibited}");
    context.Response.StatusCode = StatusCodes.Status200OK;
    await using var stream = await feature.AcceptAsync();
    Console.WriteLine(JsonSerializer.Serialize(new { eventName = "extended-connect-accepted", protocol = context.Request.Protocol, method = context.Request.Method, scheme = context.Request.Scheme, authority = context.Request.Host.Value, path = context.Request.Path.Value, pseudoProtocol = feature.Protocol, responseStatus = 200 }));
    while (true)
    {
        var frame = await ReadFrame(stream, context.RequestAborted);
        if (!frame.Masked) throw new InvalidOperationException("client WebSocket frame was not masked");
        if (!frame.Fin || frame.Rsv != 0) throw new InvalidOperationException("fragmented or RSV-bearing frame unsupported");
        switch (frame.Opcode)
        {
            case 0x1:
                if (Encoding.UTF8.GetString(frame.Payload) != TextPayload) throw new InvalidOperationException("text payload mismatch");
                await WriteFrame(stream, 0x1, frame.Payload, context.RequestAborted);
                break;
            case 0x2:
                if (frame.Payload.Length != 1024 || frame.Payload.Any(value => value != 66)) throw new InvalidOperationException("binary payload mismatch");
                await WriteFrame(stream, 0x2, frame.Payload, context.RequestAborted);
                break;
            case 0x9:
                if (Encoding.UTF8.GetString(frame.Payload) != ControlPayload) throw new InvalidOperationException("ping payload mismatch");
                await WriteFrame(stream, 0xA, frame.Payload, context.RequestAborted);
                break;
            case 0x8:
                if (frame.Payload.Length != 2 || BinaryPrimitives.ReadUInt16BigEndian(frame.Payload) != 1000) throw new InvalidOperationException("close payload mismatch");
                await WriteFrame(stream, 0x8, frame.Payload, context.RequestAborted);
                Console.WriteLine(JsonSerializer.Serialize(new { eventName = "websocket-clean-close", closeCode = 1000, clientMaskObserved = true }));
                return;
            default:
                throw new InvalidOperationException($"unsupported WebSocket opcode {frame.Opcode}");
        }
    }
}

static async Task<WsFrame> ReadFrame(Stream stream, CancellationToken cancellationToken)
{
    var header = new byte[2];
    await ReadExact(stream, header, cancellationToken);
    var fin = (header[0] & 0x80) != 0;
    var rsv = (byte)(header[0] & 0x70);
    var opcode = (byte)(header[0] & 0x0F);
    var masked = (header[1] & 0x80) != 0;
    ulong length = (uint)(header[1] & 0x7F);
    if (length == 126) { var ext = new byte[2]; await ReadExact(stream, ext, cancellationToken); length = BinaryPrimitives.ReadUInt16BigEndian(ext); }
    else if (length == 127) { var ext = new byte[8]; await ReadExact(stream, ext, cancellationToken); length = BinaryPrimitives.ReadUInt64BigEndian(ext); }
    if (length > 1_048_576) throw new InvalidOperationException("WebSocket frame too large");
    var mask = new byte[4];
    if (masked) await ReadExact(stream, mask, cancellationToken);
    var payload = new byte[(int)length];
    await ReadExact(stream, payload, cancellationToken);
    if (masked) for (var index = 0; index < payload.Length; index++) payload[index] ^= mask[index % 4];
    return new WsFrame(fin, rsv, opcode, masked, payload);
}

static async Task WriteFrame(Stream stream, byte opcode, byte[] payload, CancellationToken cancellationToken)
{
    using var buffer = new MemoryStream();
    buffer.WriteByte((byte)(0x80 | opcode));
    if (payload.Length <= 125) buffer.WriteByte((byte)payload.Length);
    else if (payload.Length <= ushort.MaxValue) { buffer.WriteByte(126); Span<byte> size = stackalloc byte[2]; BinaryPrimitives.WriteUInt16BigEndian(size, (ushort)payload.Length); buffer.Write(size); }
    else { buffer.WriteByte(127); Span<byte> size = stackalloc byte[8]; BinaryPrimitives.WriteUInt64BigEndian(size, (ulong)payload.Length); buffer.Write(size); }
    buffer.Write(payload);
    await stream.WriteAsync(buffer.ToArray(), cancellationToken);
    await stream.FlushAsync(cancellationToken);
}

static async Task ReadExact(Stream stream, byte[] buffer, CancellationToken cancellationToken)
{
    var offset = 0;
    while (offset < buffer.Length)
    {
        var read = await stream.ReadAsync(buffer.AsMemory(offset), cancellationToken);
        if (read == 0) throw new EndOfStreamException();
        offset += read;
    }
}

readonly record struct WsFrame(bool Fin, byte Rsv, byte Opcode, bool Masked, byte[] Payload);
