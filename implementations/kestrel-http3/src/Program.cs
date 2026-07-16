using System.Net;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using Microsoft.AspNetCore.Server.Kestrel.Core;

var builder = WebApplication.CreateBuilder(args);
var port = ResolvePort(builder.Configuration, 8443);
var certificate = CreateLoopbackCertificate();

builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(port, listenOptions =>
    {
        listenOptions.Protocols = HttpProtocols.Http3;
        listenOptions.UseHttps(certificate);
    });
});

var app = builder.Build();
app.Lifetime.ApplicationStopped.Register(certificate.Dispose);
var bytes1Kb = new string('x', 1024);
var bytes64Kb = new string('x', 65_536);
var bytes1Mb = new string('x', 1_048_576);
var repeatedHeaderValue = new string('h', 32);

app.MapGet("/health", () => Results.Json(new { status = "ok", implementationId = "kestrel-http3", protocol = "h3" }));
app.MapGet("/status", () => Results.Json(new
{
    protocol = "h3",
    server = "kestrel",
    implementation = "kestrel-http3",
    utc = DateTimeOffset.UtcNow,
    processId = Environment.ProcessId
}));
app.MapGet("/protocol-lab/metadata", () => Results.Json(new
{
    implementationId = "kestrel-http3",
    packageId = "org.protocol-lab.components.implementation.kestrel-http3",
    protocol = "h3",
    protocolVersion = "http/3",
    supportedScenarios = new[] { "http3.core.status", "http3.payload.bytes.1kb", "http3.payload.bytes.64kb", "http3.payload.bytes.1mb", "http3.headers.response-headers-50x32", "http3.protocol.qpack-repeated-headers" },
    unsupportedKnownCases = new[] { "h1", "h2", "h2c", "raw-quic", "websocket", "server-sent-events" }
}));
app.MapGet("/plaintext", () => Results.Text("Hello, World!", "text/plain"));
app.MapGet("/json", () => Results.Text("""{"message":"Hello, World!"}""", "application/json"));
app.MapGet("/bytes/1024", () => Results.Text(bytes1Kb, "application/octet-stream"));
app.MapGet("/bytes/1kb", () => Results.Text(bytes1Kb, "application/octet-stream"));
app.MapGet("/bytes/65536", () => Results.Text(bytes64Kb, "application/octet-stream"));
app.MapGet("/bytes/1048576", () => Results.Text(bytes1Mb, "application/octet-stream"));
app.MapGet("/headers/response", (HttpContext context, int? count, int? size) =>
{
    var headerCount = count ?? 50;
    var headerSize = size ?? 32;
    if (headerCount != 50 || headerSize != 32)
    {
        return Results.BadRequest(new { error = "expected count=50 and size=32" });
    }

    for (var index = 0; index < headerCount; index++)
    {
        context.Response.Headers.Append($"x-protocol-bench-header-{index:D2}", repeatedHeaderValue);
    }

    return Results.Text("headers", "text/plain");
});

app.Run();

static int ResolvePort(IConfiguration configuration, int fallback)
{
    var configuredPort = Environment.GetEnvironmentVariable("PLAB_HTTP_PORT") ?? Environment.GetEnvironmentVariable("PORT") ?? configuration["port"];
    return int.TryParse(configuredPort, out var port) && port is > IPEndPoint.MinPort and <= IPEndPoint.MaxPort ? port : fallback;
}

static X509Certificate2 CreateLoopbackCertificate()
{
    using var key = RSA.Create(2048);
    var request = new CertificateRequest(
        "CN=protocol-lab-kestrel-http3",
        key,
        HashAlgorithmName.SHA256,
        RSASignaturePadding.Pkcs1);
    request.CertificateExtensions.Add(new X509BasicConstraintsExtension(false, false, 0, true));
    request.CertificateExtensions.Add(new X509KeyUsageExtension(X509KeyUsageFlags.DigitalSignature, true));
    request.CertificateExtensions.Add(new X509SubjectKeyIdentifierExtension(request.PublicKey, false));
    var san = new SubjectAlternativeNameBuilder();
    san.AddDnsName("localhost");
    san.AddIpAddress(IPAddress.Loopback);
    san.AddIpAddress(IPAddress.IPv6Loopback);
    request.CertificateExtensions.Add(san.Build());
    var now = DateTimeOffset.UtcNow;
    return request.CreateSelfSigned(now.AddMinutes(-5), now.AddDays(7));
}
