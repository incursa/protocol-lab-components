package org.protocollab;

import java.nio.ByteBuffer;
import org.eclipse.jetty.alpn.server.ALPNServerConnectionFactory;
import org.eclipse.jetty.http2.server.HTTP2ServerConnectionFactory;
import org.eclipse.jetty.server.HttpConfiguration;
import org.eclipse.jetty.server.SecureRequestCustomizer;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.server.ServerConnector;
import org.eclipse.jetty.server.SslConnectionFactory;
import org.eclipse.jetty.server.handler.ContextHandler;
import org.eclipse.jetty.util.ssl.SslContextFactory;
import org.eclipse.jetty.websocket.api.Callback;
import org.eclipse.jetty.websocket.api.Session;
import org.eclipse.jetty.websocket.server.WebSocketUpgradeHandler;

public final class JettyHttp2WebSocketServer {
    private JettyHttp2WebSocketServer() {}

    public static void main(String[] args) throws Exception {
        if (args.length == 1 && "--version".equals(args[0])) {
            System.out.println("jetty-http2-websocket 0.1.0 (Jetty 12.1.9)");
            return;
        }

        int port = Integer.parseInt(System.getenv().getOrDefault("PLAB_TARGET_PORT", "18452"));
        Server server = new Server();
        HttpConfiguration http = new HttpConfiguration();
        http.addCustomizer(new SecureRequestCustomizer());

        HTTP2ServerConnectionFactory h2 = new HTTP2ServerConnectionFactory(http);
        h2.setConnectProtocolEnabled(true);
        ALPNServerConnectionFactory alpn = new ALPNServerConnectionFactory();
        alpn.setDefaultProtocol(h2.getProtocol());

        SslContextFactory.Server ssl = new SslContextFactory.Server();
        ssl.setKeyStorePath("/app/keystore.p12");
        ssl.setKeyStorePassword("protocol-lab");
        ssl.setIncludeProtocols("TLSv1.3");
        ssl.setIncludeCipherSuites("TLS_AES_128_GCM_SHA256");

        ServerConnector connector = new ServerConnector(server, new SslConnectionFactory(ssl, alpn.getProtocol()), alpn, h2);
        connector.setHost("0.0.0.0");
        connector.setPort(port);
        server.addConnector(connector);

        ContextHandler context = new ContextHandler("/");
        server.setHandler(context);
        WebSocketUpgradeHandler websocket = WebSocketUpgradeHandler.from(server, context, container -> {
            container.setMaxTextMessageSize(1024 * 1024);
            container.setMaxBinaryMessageSize(1024 * 1024);
            container.addMapping("/websocket", (request, response, callback) -> new EchoSocket());
        });
        context.setHandler(websocket);

        server.start();
        System.out.printf("{\"status\":\"ready\",\"implementationId\":\"jetty-http2-websocket\",\"version\":\"0.1.0\",\"upstream\":{\"name\":\"jetty\",\"version\":\"12.1.9\"},\"protocol\":\"h2\",\"protocolVersion\":\"HTTP/2\",\"protocolVariant\":\"rfc8441-extended-connect\",\"transportSecurity\":\"tls13\",\"port\":%d}%n", port);
        server.join();
    }

    public static final class EchoSocket implements Session.Listener.AutoDemanding {
        private Session session;
        @Override public void onWebSocketOpen(Session session) { this.session = session; }
        @Override public void onWebSocketText(String message) { session.sendText(message, Callback.NOOP); }
        @Override public void onWebSocketBinary(ByteBuffer payload, Callback callback) { session.sendBinary(payload, callback); }
    }
}
