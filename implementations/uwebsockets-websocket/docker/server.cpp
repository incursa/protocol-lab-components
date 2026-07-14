#include "App.h"
#include <cstdlib>
#include <iostream>
#include <string>

int main(int argc, char **argv) {
    if (argc == 2 && std::string(argv[1]) == "--version") {
        std::cout << "uwebsockets-websocket 0.1.0 (uWebSockets 20.79.0)" << std::endl;
        return 0;
    }
    struct PerSocketData {};
    const char *rawPort = std::getenv("PLAB_TARGET_PORT");
    const int port = rawPort ? std::stoi(rawPort) : 18081;

    uWS::App().ws<PerSocketData>("/websocket", {
        .compression = uWS::DISABLED,
        .maxPayloadLength = 1024 * 1024,
        .idleTimeout = 120,
        .maxBackpressure = 1024 * 1024,
        .closeOnBackpressureLimit = true,
        .resetIdleTimeoutOnSend = true,
        .sendPingsAutomatically = false,
        .message = [](auto *ws, std::string_view message, uWS::OpCode opcode) {
            ws->send(message, opcode, false);
        }
    }).listen("0.0.0.0", port, [port](auto *token) {
        if (!token) {
            std::cerr << "Failed to listen on port " << port << std::endl;
            std::exit(1);
        }
        std::cout << "{\"status\":\"ready\",\"implementationId\":\"uwebsockets-websocket\",\"version\":\"0.1.0\",\"upstream\":{\"name\":\"uWebSockets\",\"version\":\"20.79.0\"},\"protocol\":\"h1\",\"protocolVersion\":\"HTTP/1.1\",\"protocolVariant\":\"websocket-h1-cleartext-upgrade\",\"transportSecurity\":\"cleartext\",\"port\":" << port << "}" << std::endl;
    }).run();
}
