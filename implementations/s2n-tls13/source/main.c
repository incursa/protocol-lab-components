#define _POSIX_C_SOURCE 200809L

#include <arpa/inet.h>
#include <errno.h>
#include <netinet/in.h>
#include <signal.h>
#include <s2n.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

#define IMPLEMENTATION_ID "s2n-tls13"
#define IMPLEMENTATION_VERSION "0.1.0"
#define S2N_VERSION "1.7.5"
#define SCENARIO_ID "tls.handshake.full"
#define REQUIRED_CIPHER "TLS_AES_128_GCM_SHA256"
#define REQUIRED_CURVE "x25519"
#define REQUIRED_ALPN "protocol-lab-tls"
#define REQUIRED_SNI "tls.plab.test"

static void fail_s2n(const char *operation)
{
    fprintf(stderr, "%s: %s (%s)\n", operation, s2n_strerror(s2n_errno, "EN"), s2n_strerror_debug(s2n_errno, "EN"));
    exit(1);
}

static char *read_file(const char *path)
{
    FILE *file = fopen(path, "rb");
    if (file == NULL) {
        perror(path);
        exit(1);
    }
    if (fseek(file, 0, SEEK_END) != 0) {
        perror("fseek");
        exit(1);
    }
    long length = ftell(file);
    if (length <= 0 || fseek(file, 0, SEEK_SET) != 0) {
        fprintf(stderr, "invalid credential file: %s\n", path);
        exit(1);
    }
    char *buffer = calloc((size_t) length + 1, 1);
    if (buffer == NULL || fread(buffer, 1, (size_t) length, file) != (size_t) length) {
        fprintf(stderr, "failed to read credential file: %s\n", path);
        exit(1);
    }
    fclose(file);
    return buffer;
}

static struct s2n_config *build_config(void)
{
    char *certificate = read_file("/opt/protocol-lab/certs/leaf.pem");
    char *private_key = read_file("/opt/protocol-lab/certs/leaf-key.pem");
    struct s2n_cert_chain_and_key *chain = s2n_cert_chain_and_key_new();
    struct s2n_config *config = s2n_config_new();
    if (chain == NULL || config == NULL) {
        fail_s2n("allocate s2n configuration");
    }
    if (s2n_cert_chain_and_key_load_pem(chain, certificate, private_key) != S2N_SUCCESS
        || s2n_config_add_cert_chain_and_key_to_store(config, chain) != S2N_SUCCESS
        || s2n_config_set_cipher_preferences(config, "default_tls13") != S2N_SUCCESS
        || s2n_config_set_session_tickets_onoff(config, 0) != S2N_SUCCESS) {
        fail_s2n("configure canonical TLS profile");
    }
    const char *protocols[] = { REQUIRED_ALPN };
    if (s2n_config_set_protocol_preferences(config, protocols, 1) != S2N_SUCCESS) {
        fail_s2n("configure ALPN");
    }
    free(certificate);
    free(private_key);
    return config;
}

static int listen_socket(void)
{
    const char *port_text = getenv("PLAB_TLS_PORT");
    int port = port_text == NULL ? 8443 : atoi(port_text);
    if (port <= 0 || port > 65535) {
        fprintf(stderr, "invalid PLAB_TLS_PORT\n");
        exit(1);
    }
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    int enabled = 1;
    if (fd < 0 || setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &enabled, sizeof(enabled)) < 0) {
        perror("socket");
        exit(1);
    }
    struct sockaddr_in address = { 0 };
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = htonl(INADDR_ANY);
    address.sin_port = htons((uint16_t) port);
    if (bind(fd, (struct sockaddr *) &address, sizeof(address)) < 0 || listen(fd, 128) < 0) {
        perror("bind/listen");
        exit(1);
    }
    return fd;
}

static bool exact(const char *actual, const char *expected)
{
    return actual != NULL && strcmp(actual, expected) == 0;
}

static void handle_client(int fd, struct s2n_config *config)
{
    struct s2n_connection *connection = s2n_connection_new(S2N_SERVER);
    if (connection == NULL || s2n_connection_set_config(connection, config) != S2N_SUCCESS
        || s2n_connection_set_fd(connection, fd) != S2N_SUCCESS) {
        fprintf(stderr, "connection setup failed\n");
        if (connection != NULL) {
            s2n_connection_free(connection);
        }
        return;
    }

    s2n_blocked_status blocked = S2N_NOT_BLOCKED;
    if (s2n_negotiate(connection, &blocked) != S2N_SUCCESS) {
        if (s2n_error_get_type(s2n_errno) != S2N_ERR_T_CLOSED) {
            fprintf(stderr, "handshake rejected: %s\n", s2n_strerror(s2n_errno, "EN"));
        }
        s2n_connection_free(connection);
        return;
    }

    const int version = s2n_connection_get_actual_protocol_version(connection);
    const char *cipher = s2n_connection_get_cipher(connection);
    const char *curve = s2n_connection_get_curve(connection);
    const char *alpn = s2n_get_application_protocol(connection);
    const char *server_name = s2n_get_server_name(connection);
    if (version != S2N_TLS13 || !exact(cipher, REQUIRED_CIPHER) || !exact(curve, REQUIRED_CURVE)
        || !exact(alpn, REQUIRED_ALPN) || !exact(server_name, REQUIRED_SNI)) {
        fprintf(stderr, "negotiation mismatch: version=%d cipher=%s curve=%s alpn=%s sni=%s\n",
                version, cipher ? cipher : "<missing>", curve ? curve : "<missing>",
                alpn ? alpn : "<missing>", server_name ? server_name : "<missing>");
        s2n_connection_free(connection);
        return;
    }

    (void) s2n_shutdown(connection, &blocked);
    s2n_connection_free(connection);
}

int main(int argc, char **argv)
{
    if (argc == 2 && strcmp(argv[1], "--version") == 0) {
        puts("s2n-tls " S2N_VERSION);
        return 0;
    }
    const char *scenario = getenv("PLAB_SCENARIO_ID");
    if (scenario != NULL && strcmp(scenario, SCENARIO_ID) != 0) {
        fprintf(stderr, "unsupported:%s\n", scenario);
        return 3;
    }
    signal(SIGPIPE, SIG_IGN);
    if (s2n_init() != S2N_SUCCESS) {
        fail_s2n("s2n_init");
    }
    struct s2n_config *config = build_config();
    int listener = listen_socket();
    printf("{\"eventName\":\"ready\",\"implementationId\":\"%s\",\"implementationVersion\":\"%s\",\"scenarioId\":\"%s\",\"tlsVersion\":\"TLS1.3\",\"cipherSuite\":\"%s\",\"keyExchangeGroup\":\"X25519\",\"alpn\":\"%s\",\"serverName\":\"%s\",\"sessionTicketsEnabled\":false}\n",
           IMPLEMENTATION_ID, IMPLEMENTATION_VERSION, SCENARIO_ID, REQUIRED_CIPHER, REQUIRED_ALPN, REQUIRED_SNI);
    fflush(stdout);

    while (true) {
        int client = accept(listener, NULL, NULL);
        if (client < 0) {
            if (errno == EINTR) {
                continue;
            }
            perror("accept");
            break;
        }
        handle_client(client, config);
        close(client);
    }
    close(listener);
    s2n_config_free(config);
    s2n_cleanup();
    return 1;
}
