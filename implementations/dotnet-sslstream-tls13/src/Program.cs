using System.Net;
using System.Net.Security;
using System.Net.Sockets;
using System.Security.Authentication;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text.Json;

var port = ReadPort();
var componentRoot = Path.GetFullPath(Path.Combine(AppContext.BaseDirectory, "..", "..", "..", ".."));
var packagedRoot = Directory.Exists(Path.Combine(AppContext.BaseDirectory, "certs"))
    ? AppContext.BaseDirectory
    : componentRoot;
var certificatePath = ResolvePath("PLAB_TLS_CERTIFICATE_PATH", Path.Combine(packagedRoot, "certs", "leaf.pem"));
var privateKeyPath = ResolvePath("PLAB_TLS_PRIVATE_KEY_PATH", Path.Combine(packagedRoot, "certs", "leaf-key.pem"));

if (OperatingSystem.IsWindows())
{
    throw new PlatformNotSupportedException(
        "The committed plab-tls13-p256-v1 contract requires TLS_AES_128_GCM_SHA256. " +
        "Windows Schannel does not support per-process CipherSuitesPolicy, so this target must run in its Linux container mode.");
}

using var sourceCertificate = X509Certificate2.CreateFromPemFile(certificatePath, privateKeyPath);
using var certificate = X509CertificateLoader.LoadPkcs12(
    sourceCertificate.Export(X509ContentType.Pkcs12),
    password: null,
    X509KeyStorageFlags.UserKeySet | X509KeyStorageFlags.Exportable);

var expectedLeafHash = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e";
var observedLeafHash = Convert.ToHexStringLower(SHA256.HashData(certificate.RawData));
if (!string.Equals(expectedLeafHash, observedLeafHash, StringComparison.Ordinal))
{
    throw new InvalidOperationException($"TLS certificate substitution detected: expected {expectedLeafHash}, observed {observedLeafHash}.");
}

var options = new SslServerAuthenticationOptions
{
    ServerCertificate = certificate,
    ClientCertificateRequired = false,
    EnabledSslProtocols = SslProtocols.Tls13,
    CipherSuitesPolicy = new CipherSuitesPolicy([TlsCipherSuite.TLS_AES_128_GCM_SHA256]),
    ApplicationProtocols = [new SslApplicationProtocol("protocol-lab-tls")],
    CertificateRevocationCheckMode = X509RevocationMode.NoCheck,
    AllowTlsResume = false,
};

using var shutdown = new CancellationTokenSource();
Console.CancelKeyPress += (_, eventArgs) =>
{
    eventArgs.Cancel = true;
    shutdown.Cancel();
};

var listener = new TcpListener(IPAddress.Any, port);
listener.Start();
Console.WriteLine(JsonSerializer.Serialize(new
{
    eventName = "ready",
    implementationId = "dotnet-sslstream-tls13",
    protocol = "tls",
    version = "TLS1.3",
    endpoint = $"tls://0.0.0.0:{port}",
    alpn = "protocol-lab-tls",
    cipherSuite = "TLS_AES_128_GCM_SHA256",
    certificateProfileId = "plab-single-leaf-p256-v1",
    certificateDerSha256 = observedLeafHash,
    allowTlsResume = false,
}));

try
{
    while (!shutdown.IsCancellationRequested)
    {
        TcpClient client;
        try
        {
            client = await listener.AcceptTcpClientAsync(shutdown.Token);
        }
        catch (OperationCanceledException) when (shutdown.IsCancellationRequested)
        {
            break;
        }

        _ = HandleAsync(client, options, shutdown.Token);
    }
}
finally
{
    listener.Stop();
}

static async Task HandleAsync(
    TcpClient client,
    SslServerAuthenticationOptions options,
    CancellationToken shutdown)
{
    try
    {
        using (client)
        await using (var stream = new SslStream(client.GetStream(), leaveInnerStreamOpen: false))
        {
            using var timeout = CancellationTokenSource.CreateLinkedTokenSource(shutdown);
            timeout.CancelAfter(TimeSpan.FromSeconds(5));
            await stream.AuthenticateAsServerAsync(options, timeout.Token);
            if (stream.SslProtocol != SslProtocols.Tls13 ||
                stream.NegotiatedApplicationProtocol != new SslApplicationProtocol("protocol-lab-tls") ||
                stream.NegotiatedCipherSuite != TlsCipherSuite.TLS_AES_128_GCM_SHA256)
            {
                throw new AuthenticationException(
                    $"Unexpected negotiation: protocol={stream.SslProtocol}, alpn={stream.NegotiatedApplicationProtocol}.");
            }

            var buffer = new byte[1];
            var applicationBytes = await stream.ReadAsync(buffer, timeout.Token);
            if (applicationBytes != 0)
            {
                throw new InvalidOperationException("TLS handshake workload received unexpected application data.");
            }
        }
    }
    catch (OperationCanceledException) when (shutdown.IsCancellationRequested)
    {
    }
    catch (Exception exception)
    {
        Console.Error.WriteLine(JsonSerializer.Serialize(new
        {
            eventName = "session-failed",
            exception = exception.GetType().Name,
            message = exception.Message,
        }));
    }
}

static int ReadPort()
{
    var value = Environment.GetEnvironmentVariable("PLAB_TLS_PORT");
    return int.TryParse(value, out var port) && port is > 0 and <= 65535 ? port : 8443;
}

static string ResolvePath(string variable, string fallback)
{
    var value = Environment.GetEnvironmentVariable(variable);
    return Path.GetFullPath(string.IsNullOrWhiteSpace(value) ? fallback : value);
}
