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
var bytes1Kb = new string('x', 1024);

app.MapGet("/health", () => Results.Json(new { status = "ok", implementationId = "kestrel-http3", protocol = "h3" }));
app.MapGet("/protocol-lab/metadata", () => Results.Json(new
{
    implementationId = "kestrel-http3",
    packageId = "org.protocol-lab.components.implementation.kestrel-http3",
    protocol = "h3",
    protocolVersion = "http/3",
    supportedScenarios = new[] { "http.core.plaintext", "http.core.json", "http.payload.bytes.1kb" },
    unsupportedKnownCases = new[] { "h1", "h2", "h2c", "raw-quic", "websocket", "server-sent-events" }
}));
app.MapGet("/plaintext", () => Results.Text("Hello, World!", "text/plain"));
app.MapGet("/json", () => Results.Text("""{"message":"Hello, World!"}""", "application/json"));
app.MapGet("/bytes/1kb", () => Results.Text(bytes1Kb, "application/octet-stream"));

app.Run();

static int ResolvePort(IConfiguration configuration, int fallback)
{
    var configuredPort = Environment.GetEnvironmentVariable("PLAB_HTTP_PORT") ?? Environment.GetEnvironmentVariable("PORT") ?? configuration["port"];
    return int.TryParse(configuredPort, out var port) && port is > IPEndPoint.MinPort and <= IPEndPoint.MaxPort ? port : fallback;
}
