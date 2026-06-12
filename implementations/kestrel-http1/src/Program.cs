using System.Net;
using System.Text.Json;
using Microsoft.AspNetCore.Server.Kestrel.Core;

const string message = "Hello, World!";

var builder = WebApplication.CreateBuilder(args);

var port = ResolvePort(builder.Configuration);
builder.WebHost.ConfigureKestrel(options =>
{
    options.ListenAnyIP(port, listenOptions =>
    {
        listenOptions.Protocols = HttpProtocols.Http1;
    });
});

builder.Logging.ClearProviders();
builder.Logging.AddSimpleConsole(options =>
{
    options.SingleLine = true;
    options.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffZ ";
    options.UseUtcTimestamp = true;
});

var app = builder.Build();

app.MapGet("/health", () => Results.Json(new
{
    status = "ok",
    implementationId = "kestrel-http1",
    protocol = "h1"
}));

app.MapGet("/protocol-lab/metadata", () => Results.Json(new
{
    implementationId = "kestrel-http1",
    packageId = "org.protocol-lab.components.implementation.kestrel-http1",
    protocol = "h1",
    protocolVersion = "http/1.1",
    supportedScenarios = new[] { "http.core.plaintext", "http.core.echo", "http.core.json" },
    supportedTestCaseIds = new[] { "http.core.plaintext", "http.core.echo", "http.core.json" },
    supportedCapabilities = new[] { "http.server", "httpPlaintext", "httpEcho", "httpJson" },
    unsupportedKnownCases = new[] { "h2", "h3", "https", "websocket", "server-sent-events" },
    endpoints = new
    {
        plaintext = "/plaintext",
        echo = "/echo",
        json = "/json"
    }
}));

app.MapGet("/plaintext", static () =>
    Results.Text(message, "text/plain", statusCode: StatusCodes.Status200OK));

app.MapGet("/json", static () =>
    Results.Text("""{"message":"Hello, World!"}""", "application/json", statusCode: StatusCodes.Status200OK));

app.MapPost("/echo", static async (HttpContext context) =>
{
    context.Response.StatusCode = StatusCodes.Status200OK;
    context.Response.ContentType = "application/octet-stream";
    await context.Request.Body.CopyToAsync(context.Response.Body, context.RequestAborted);
});

app.Run();

static int ResolvePort(IConfiguration configuration)
{
    var configuredPort =
        Environment.GetEnvironmentVariable("PLAB_HTTP_PORT") ??
        Environment.GetEnvironmentVariable("PORT") ??
        configuration["port"];

    if (int.TryParse(configuredPort, out var port) &&
        port is > IPEndPoint.MinPort and <= IPEndPoint.MaxPort)
    {
        return port;
    }

    return 8080;
}
