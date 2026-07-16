package org.protocollab.jetty;

import java.io.IOException;
import jakarta.servlet.http.HttpServlet;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

public final class OriginServlet extends HttpServlet {
    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response) throws IOException {
        switch (request.getRequestURI()) {
            case "/plaintext" -> write(response, "text/plain", "Hello, World!");
            case "/json" -> write(response, "application/json", "{\"message\":\"Hello, World!\"}");
            case "/health" -> write(response, "application/json", "{\"status\":\"ok\",\"implementationId\":\"jetty-http-origin\"}");
            case "/protocol-lab/metadata" -> write(response, "application/json", "{\"implementationId\":\"jetty-http-origin\",\"packageId\":\"org.protocol-lab.components.implementation.jetty-http-origin\",\"supportedProtocols\":[\"h1\",\"h2\"]}");
            default -> {
                response.setStatus(HttpServletResponse.SC_NOT_FOUND);
                write(response, "text/plain", "not found");
            }
        }
    }

    private static void write(HttpServletResponse response, String contentType, String body) throws IOException {
        response.setContentType(contentType);
        response.setCharacterEncoding("UTF-8");
        response.getWriter().write(body);
    }
}
