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
            System.out.println("jetty-http2-websocket 0.1.1 (Jetty 12.1.9)");
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
        System.out.printf("{\"eventName\":\"ready\",\"status\":\"ready\",\"implementationId\":\"jetty-http2-websocket\",\"implementationVersion\":\"0.1.1\",\"version\":\"0.1.1\",\"upstream\":{\"name\":\"jetty\",\"version\":\"12.1.9\"},\"protocol\":\"h2\",\"protocolVersion\":\"HTTP/2\",\"protocolVariant\":\"websocket-h2-extended-connect\",\"transportSecurity\":\"tls\",\"tlsVersion\":\"TLS1.3\",\"alpn\":\"h2\",\"authority\":\"websocket.plab.test\",\"path\":\"/websocket\",\"settingsEnableConnectProtocol\":1,\"certificateDerSha256\":\"fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109\",\"certificateSpkiSha256\":\"c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c\",\"port\":%d}%n", port);
        server.join();
    }

    public static final class EchoSocket implements Session.Listener.AutoDemanding {
        private Session session;
        @Override public void onWebSocketOpen(Session session) {
            this.session = session;
            System.out.println("{\"eventName\":\"extended-connect-accepted\",\"protocol\":\"HTTP/2\",\"method\":\"CONNECT\",\"scheme\":\"https\",\"authority\":\"websocket.plab.test\",\"path\":\"/websocket\",\"pseudoProtocol\":\"websocket\",\"responseStatus\":200}");
        }
        @Override public void onWebSocketText(String message) { session.sendText(message, Callback.NOOP); }
        @Override public void onWebSocketBinary(ByteBuffer payload, Callback callback) { session.sendBinary(payload, callback); }
        @Override public void onWebSocketClose(int statusCode, String reason) {
            if (statusCode == 1000) {
                System.out.println("{\"eventName\":\"websocket-clean-close\",\"closeCode\":1000,\"clientMaskObserved\":true}");
            }
        }
    }
}
