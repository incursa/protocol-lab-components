package org.protocollab;

import java.nio.ByteBuffer;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.server.ServerConnector;
import org.eclipse.jetty.server.handler.ContextHandler;
import org.eclipse.jetty.websocket.api.Callback;
import org.eclipse.jetty.websocket.api.Session;
import org.eclipse.jetty.websocket.server.WebSocketUpgradeHandler;

public final class JettyWebSocketServer {
    private JettyWebSocketServer() {}

    public static void main(String[] args) throws Exception {
        if (args.length == 1 && "--version".equals(args[0])) {
            System.out.println("jetty-websocket 0.1.0 (Jetty 12.1.9)");
            return;
        }
        int port = Integer.parseInt(System.getenv().getOrDefault("PLAB_TARGET_PORT", "18081"));
        Server server = new Server();
        ServerConnector connector = new ServerConnector(server);
        connector.setHost("0.0.0.0");
        connector.setPort(port);
        server.addConnector(connector);

        ContextHandler context = new ContextHandler("/");
        server.setHandler(context);
        WebSocketUpgradeHandler webSocketHandler = WebSocketUpgradeHandler.from(server, context, container -> {
            container.setMaxTextMessageSize(1024 * 1024);
            container.setMaxBinaryMessageSize(1024 * 1024);
            container.addMapping("/websocket", (request, response, callback) -> new EchoSocket());
        });
        context.setHandler(webSocketHandler);

        server.start();
        System.out.printf("{\"status\":\"ready\",\"implementationId\":\"jetty-websocket\",\"version\":\"0.1.0\",\"upstream\":{\"name\":\"jetty\",\"version\":\"12.1.9\"},\"protocol\":\"h1\",\"protocolVersion\":\"HTTP/1.1\",\"protocolVariant\":\"websocket-h1-cleartext-upgrade\",\"transportSecurity\":\"cleartext\",\"port\":%d}%n", port);
        server.join();
    }

    public static final class EchoSocket implements Session.Listener.AutoDemanding {
        private Session session;

        @Override
        public void onWebSocketOpen(Session session) {
            this.session = session;
        }

        @Override
        public void onWebSocketText(String message) {
            session.sendText(message, Callback.NOOP);
        }

        @Override
        public void onWebSocketBinary(ByteBuffer payload, Callback callback) {
            session.sendBinary(payload, callback);
        }
    }
}
