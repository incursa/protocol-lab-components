using System.Net;
using Microsoft.AspNetCore.Server.Kestrel.Core;

var builder = WebApplication.CreateBuilder(args);
var port = ResolvePort(builder.Configuration, 8443);

builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(port, listenOptions =>
    {
        listenOptions.Protocols = HttpProtocols.Http3;
        listenOptions.UseHttps();
    });
});

var app = builder.Build();
var bytes64Kb = new string('x', 65_536);
var bytes1Mb = new string('x', 1_048_576);

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
    supportedScenarios = new[] { "http3.core.status", "http3.payload.bytes.64kb", "http3.payload.bytes.1mb" },
    unsupportedKnownCases = new[] { "h1", "h2", "h2c", "raw-quic", "websocket", "server-sent-events" }
}));
app.MapGet("/plaintext", () => Results.Text("Hello, World!", "text/plain"));
app.MapGet("/json", () => Results.Text("""{"message":"Hello, World!"}""", "application/json"));
app.MapGet("/bytes/1kb", () => Results.Text(new string('x', 1024), "application/octet-stream"));
app.MapGet("/bytes/65536", () => Results.Text(bytes64Kb, "application/octet-stream"));
app.MapGet("/bytes/1048576", () => Results.Text(bytes1Mb, "application/octet-stream"));

app.Run();

static int ResolvePort(IConfiguration configuration, int fallback)
{
    var configuredPort = Environment.GetEnvironmentVariable("PLAB_HTTP_PORT") ?? Environment.GetEnvironmentVariable("PORT") ?? configuration["port"];
    return int.TryParse(configuredPort, out var port) && port is > IPEndPoint.MinPort and <= IPEndPoint.MaxPort ? port : fallback;
}
