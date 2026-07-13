#define _POSIX_C_SOURCE 200809L

#include <openssl/err.h>
#include <openssl/evp.h>
#include <openssl/ssl.h>
#include <openssl/x509.h>

#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
typedef SOCKET plab_socket;
#define PLAB_INVALID_SOCKET INVALID_SOCKET
#define plab_close_socket closesocket
#else
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>
typedef int plab_socket;
#define PLAB_INVALID_SOCKET (-1)
#define plab_close_socket close
#endif

#define SCENARIO_ID "tls.key-update.diagnostic"
#define IMPLEMENTATION_ID "openssl-tls13-key-update"
#define IMPLEMENTATION_VERSION "0.1.0"
#define ALPN_VALUE "protocol-lab-tls"
#define SERVER_NAME "tls.plab.test"
#define PRE_BYTES 1024U
#define POST_BYTES 65536U
#define PRE_OCTET 0xA5U
#define POST_OCTET 0x5AU
#define PRE_HASH "e75809e0d15667ce44e6aa5c64689a4917b245eb0920094ff0b017dc0612a17a"
#define POST_HASH "944044fe482bc4e91085c15c5a923a1b9e02eac98d3bce04997d6dbecd2a5b8d"

typedef struct {
    unsigned sent;
    unsigned received;
    int sent_request;
    int received_request;
} key_update_observation;

typedef struct {
    uint8_t magic[4];
    uint8_t stage;
    uint8_t flags;
    uint8_t reserved[2];
    uint32_t length_be;
} plab_frame_header;

static const char *const unsupported_ids[] = {
    "tls.handshake.full",
    "tls.handshake.full.tls12",
    "tls.handshake.full.chacha20",
    "tls.handshake.mutual-auth",
    "tls.handshake.resumed",
    "tls.early-data.accepted",
    "tls.early-data.rejected",
    "tls.record.coverage",
    "tls.record.throughput"
};

static void fail_openssl(const char *message) {
    fprintf(stderr, "%s\n", message);
    ERR_print_errors_fp(stderr);
}

static int initialize_sockets(void) {
#ifdef _WIN32
    WSADATA data;
    return WSAStartup(MAKEWORD(2, 2), &data) == 0;
#else
    return 1;
#endif
}

static void cleanup_sockets(void) {
#ifdef _WIN32
    WSACleanup();
#endif
}

static int parse_port(const char *address) {
    const char *colon = strrchr(address, ':');
    char *end = NULL;
    long port;
    if (colon == NULL || colon[1] == '\0') {
        return -1;
    }
    port = strtol(colon + 1, &end, 10);
    if (*end != '\0' || port < 1 || port > 65535) {
        return -1;
    }
    return (int)port;
}

static plab_socket listen_loopback(int port) {
    plab_socket socket_fd = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    struct sockaddr_in address;
    int reuse = 1;
    if (socket_fd == PLAB_INVALID_SOCKET) {
        return PLAB_INVALID_SOCKET;
    }
    (void)setsockopt(socket_fd, SOL_SOCKET, SO_REUSEADDR, (const char *)&reuse, sizeof(reuse));
    memset(&address, 0, sizeof(address));
    address.sin_family = AF_INET;
    address.sin_port = htons((uint16_t)port);
    address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    if (bind(socket_fd, (struct sockaddr *)&address, sizeof(address)) != 0 || listen(socket_fd, 8) != 0) {
        plab_close_socket(socket_fd);
        return PLAB_INVALID_SOCKET;
    }
    return socket_fd;
}

static int alpn_select(SSL *ssl, const unsigned char **out, unsigned char *out_len,
                       const unsigned char *in, unsigned int in_len, void *arg) {
    static const unsigned char expected[] = {16U, 'p','r','o','t','o','c','o','l','-','l','a','b','-','t','l','s'};
    (void)ssl;
    (void)arg;
    if (SSL_select_next_proto((unsigned char **)out, out_len, expected, sizeof(expected),
                              in, in_len) != OPENSSL_NPN_NEGOTIATED) {
        return SSL_TLSEXT_ERR_ALERT_FATAL;
    }
    return SSL_TLSEXT_ERR_OK;
}

static void message_callback(int write_p, int version, int content_type, const void *buffer,
                             size_t length, SSL *ssl, void *arg) {
    key_update_observation *observation = (key_update_observation *)arg;
    const unsigned char *bytes = (const unsigned char *)buffer;
    (void)version;
    (void)ssl;
    if (content_type != SSL3_RT_HANDSHAKE || length < 5U || bytes[0] != SSL3_MT_KEY_UPDATE) {
        return;
    }
    if (write_p != 0) {
        observation->sent++;
        observation->sent_request = bytes[4];
    } else {
        observation->received++;
        observation->received_request = bytes[4];
    }
}

static int ssl_read_exact(SSL *ssl, void *buffer, size_t length) {
    unsigned char *cursor = (unsigned char *)buffer;
    size_t offset = 0;
    while (offset < length) {
        size_t read_count = 0;
        if (SSL_read_ex(ssl, cursor + offset, length - offset, &read_count) != 1 || read_count == 0U) {
            return 0;
        }
        offset += read_count;
    }
    return 1;
}

static int ssl_write_exact(SSL *ssl, const void *buffer, size_t length) {
    const unsigned char *cursor = (const unsigned char *)buffer;
    size_t offset = 0;
    while (offset < length) {
        size_t written = 0;
        if (SSL_write_ex(ssl, cursor + offset, length - offset, &written) != 1 || written == 0U) {
            return 0;
        }
        offset += written;
    }
    return 1;
}

static int hash_matches(const unsigned char *payload, size_t length, const char *expected) {
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int digest_length = 0;
    char actual[65];
    size_t index;
    if (EVP_Digest(payload, length, digest, &digest_length, EVP_sha256(), NULL) != 1 || digest_length != 32U) {
        return 0;
    }
    for (index = 0; index < digest_length; index++) {
        (void)snprintf(actual + (index * 2U), 3U, "%02x", digest[index]);
    }
    actual[64] = '\0';
    return strcmp(actual, expected) == 0;
}

static int read_frame(SSL *ssl, uint8_t expected_stage, uint8_t expected_octet, size_t expected_size,
                      const char *expected_hash, uint8_t *observed_flags) {
    plab_frame_header header;
    unsigned char *payload;
    uint32_t length;
    size_t index;
    int ok = 0;
    if (!ssl_read_exact(ssl, &header, sizeof(header)) || memcmp(header.magic, "PLAB", 4U) != 0 ||
        header.stage != expected_stage) {
        return 0;
    }
    length = ntohl(header.length_be);
    if ((size_t)length != expected_size) {
        return 0;
    }
    payload = (unsigned char *)malloc(expected_size);
    if (payload == NULL || !ssl_read_exact(ssl, payload, expected_size)) {
        free(payload);
        return 0;
    }
    for (index = 0; index < expected_size; index++) {
        if (payload[index] != expected_octet) {
            goto cleanup;
        }
    }
    if (!hash_matches(payload, expected_size, expected_hash)) {
        goto cleanup;
    }
    *observed_flags = header.flags;
    ok = 1;
cleanup:
    free(payload);
    return ok;
}

static int write_frame(SSL *ssl, uint8_t stage, uint8_t flags, uint8_t octet, size_t length) {
    plab_frame_header header;
    unsigned char *payload = (unsigned char *)malloc(length);
    int ok;
    if (payload == NULL) {
        return 0;
    }
    memcpy(header.magic, "PLAB", 4U);
    header.stage = stage;
    header.flags = flags;
    header.reserved[0] = 0;
    header.reserved[1] = 0;
    header.length_be = htonl((uint32_t)length);
    memset(payload, octet, length);
    ok = ssl_write_exact(ssl, &header, sizeof(header)) && ssl_write_exact(ssl, payload, length);
    free(payload);
    return ok;
}

static int configure_context(SSL_CTX *context, const char *certificate, const char *private_key) {
    if (SSL_CTX_set_min_proto_version(context, TLS1_3_VERSION) != 1 ||
        SSL_CTX_set_max_proto_version(context, TLS1_3_VERSION) != 1 ||
        SSL_CTX_set_ciphersuites(context, "TLS_AES_128_GCM_SHA256") != 1 ||
        SSL_CTX_set1_groups_list(context, "X25519") != 1 ||
        SSL_CTX_set1_sigalgs_list(context, "ecdsa_secp256r1_sha256") != 1 ||
        SSL_CTX_use_certificate_chain_file(context, certificate) != 1 ||
        SSL_CTX_use_PrivateKey_file(context, private_key, SSL_FILETYPE_PEM) != 1 ||
        SSL_CTX_check_private_key(context) != 1) {
        return 0;
    }
    SSL_CTX_set_alpn_select_cb(context, alpn_select, NULL);
    SSL_CTX_set_session_cache_mode(context, SSL_SESS_CACHE_OFF);
    SSL_CTX_set_num_tickets(context, 0);
    SSL_CTX_set_max_early_data(context, 0);
    return 1;
}

static int handle_connection(SSL_CTX *context, plab_socket client_socket) {
    SSL *ssl = SSL_new(context);
    key_update_observation observation = {0U, 0U, -1, -1};
    uint8_t flags = 0;
    const unsigned char *alpn = NULL;
    unsigned int alpn_length = 0;
    int ok = 0;
    if (ssl == NULL) {
        return 0;
    }
    SSL_set_msg_callback(ssl, message_callback);
    SSL_set_msg_callback_arg(ssl, &observation);
    if (SSL_set_fd(ssl, (int)client_socket) != 1 || SSL_accept(ssl) != 1) {
        fail_openssl("TLS accept failed.");
        goto cleanup;
    }
    SSL_get0_alpn_selected(ssl, &alpn, &alpn_length);
    if (SSL_version(ssl) != TLS1_3_VERSION || alpn_length != sizeof(ALPN_VALUE) - 1U ||
        memcmp(alpn, ALPN_VALUE, alpn_length) != 0 || SSL_session_reused(ssl) != 0 ||
        SSL_get_early_data_status(ssl) != SSL_EARLY_DATA_NOT_SENT) {
        fprintf(stderr, "TLS negotiation did not match the exact KeyUpdate profile.\n");
        goto cleanup;
    }
    if (!read_frame(ssl, 1U, PRE_OCTET, PRE_BYTES, PRE_HASH, &flags) || flags != 0U ||
        !write_frame(ssl, 1U, 0U, PRE_OCTET, PRE_BYTES)) {
        fprintf(stderr, "Pre-update deterministic exchange failed.\n");
        goto cleanup;
    }
    if (observation.received != 0U || observation.sent != 0U) {
        fprintf(stderr, "A KeyUpdate occurred before the measured update.\n");
        goto cleanup;
    }
    if (!read_frame(ssl, 2U, POST_OCTET, POST_BYTES, POST_HASH, &flags) || flags != 0U) {
        fprintf(stderr, "Post-update deterministic request failed.\n");
        goto cleanup;
    }
    if (observation.received != 1U || observation.received_request != SSL_KEY_UPDATE_NOT_REQUESTED ||
        observation.sent != 0U) {
        fprintf(stderr, "Exact client-initiated KeyUpdate observation failed.\n");
        goto cleanup;
    }
    if (!write_frame(ssl, 2U, 1U, POST_OCTET, POST_BYTES)) {
        fprintf(stderr, "Post-update deterministic response failed.\n");
        goto cleanup;
    }
    printf("{\"eventName\":\"target-proof\",\"scenarioId\":\"%s\",\"tlsVersion\":\"TLS1.3\",\"keyUpdateInitiator\":\"client\",\"keyUpdateMessagesReceived\":1,\"keyUpdateMessagesSent\":0,\"peerUpdateRequested\":false,\"clientWriteGenerationBefore\":0,\"clientWriteGenerationAfter\":1,\"serverReadGenerationBefore\":0,\"serverReadGenerationAfter\":1,\"serverWriteGenerationBefore\":0,\"serverWriteGenerationAfter\":0,\"postUpdateBytesReceived\":%u,\"postUpdateBytesSent\":%u,\"payloadSha256\":\"%s\",\"trafficSecretsPublished\":false}\n",
           SCENARIO_ID, POST_BYTES, POST_BYTES, POST_HASH);
    fflush(stdout);
    ok = 1;
cleanup:
    (void)SSL_shutdown(ssl);
    SSL_free(ssl);
    return ok;
}

static int self_test(void) {
    unsigned char *payload = (unsigned char *)malloc(POST_BYTES);
    int ok;
    if (payload == NULL) {
        return 1;
    }
    memset(payload, POST_OCTET, POST_BYTES);
    ok = hash_matches(payload, POST_BYTES, POST_HASH) && strcmp(SCENARIO_ID, "tls.key-update.diagnostic") == 0;
    free(payload);
    if (!ok) {
        fprintf(stderr, "self-test failed\n");
        return 1;
    }
    puts("self-test passed");
    return 0;
}

static int is_known_unsupported(const char *id) {
    size_t index;
    for (index = 0; index < sizeof(unsupported_ids) / sizeof(unsupported_ids[0]); index++) {
        if (strcmp(id, unsupported_ids[index]) == 0) {
            return 1;
        }
    }
    return 0;
}

int main(int argc, char **argv) {
    const char *scenario = getenv("PLAB_SCENARIO_ID");
    const char *listen_address = getenv("PLAB_LISTEN_ADDRESS");
    const char *certificate = getenv("PLAB_TLS_CERT_FILE");
    const char *private_key = getenv("PLAB_TLS_KEY_FILE");
    SSL_CTX *context = NULL;
    plab_socket listener = PLAB_INVALID_SOCKET;
    plab_socket client = PLAB_INVALID_SOCKET;
    int port;
    int exit_code = 1;
    if (argc == 2 && strcmp(argv[1], "--self-test") == 0) {
        return self_test();
    }
    if (scenario == NULL || strcmp(scenario, SCENARIO_ID) != 0) {
        if (scenario != NULL && is_known_unsupported(scenario)) {
            fprintf(stderr, "unsupported:%s\n", scenario);
            return 3;
        }
        fprintf(stderr, "unknown:%s\n", scenario == NULL ? "" : scenario);
        return 2;
    }
    if (listen_address == NULL || certificate == NULL || private_key == NULL ||
        (port = parse_port(listen_address)) < 1) {
        fprintf(stderr, "Required target environment is missing or invalid.\n");
        return 2;
    }
    if (!initialize_sockets()) {
        fprintf(stderr, "Socket initialization failed.\n");
        return 1;
    }
    SSL_load_error_strings();
    OPENSSL_init_ssl(0, NULL);
    context = SSL_CTX_new(TLS_server_method());
    if (context == NULL || !configure_context(context, certificate, private_key)) {
        fail_openssl("TLS context configuration failed.");
        goto cleanup;
    }
    listener = listen_loopback(port);
    if (listener == PLAB_INVALID_SOCKET) {
        fprintf(stderr, "Loopback listener failed.\n");
        goto cleanup;
    }
    printf("{\"eventName\":\"ready\",\"implementationId\":\"%s\",\"implementationVersion\":\"%s\",\"scenarioId\":\"%s\",\"listenAddress\":\"127.0.0.1:%d\",\"tlsVersion\":\"TLS1.3\",\"alpn\":\"%s\",\"keyUpdateApi\":\"SSL_key_update\"}\n",
           IMPLEMENTATION_ID, IMPLEMENTATION_VERSION, SCENARIO_ID, port, ALPN_VALUE);
    fflush(stdout);
    client = accept(listener, NULL, NULL);
    if (client == PLAB_INVALID_SOCKET || !handle_connection(context, client)) {
        goto cleanup;
    }
    exit_code = 0;
cleanup:
    if (client != PLAB_INVALID_SOCKET) {
        plab_close_socket(client);
    }
    if (listener != PLAB_INVALID_SOCKET) {
        plab_close_socket(listener);
    }
    SSL_CTX_free(context);
    cleanup_sockets();
    return exit_code;
}
