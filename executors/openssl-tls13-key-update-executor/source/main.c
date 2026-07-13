#define _POSIX_C_SOURCE 200809L

#include <openssl/err.h>
#include <openssl/evp.h>
#include <openssl/obj_mac.h>
#include <openssl/pem.h>
#include <openssl/ssl.h>
#include <openssl/x509.h>
#include <openssl/x509_vfy.h>

#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
typedef SOCKET plab_socket;
#define PLAB_INVALID_SOCKET INVALID_SOCKET
#define plab_close_socket closesocket
#define PLATFORM_OS "windows"
#else
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
typedef int plab_socket;
#define PLAB_INVALID_SOCKET (-1)
#define plab_close_socket close
#define PLATFORM_OS "linux"
#endif

#define AUTHORITY_COMMIT "8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574"
#define SCENARIO_ID "tls.key-update.diagnostic"
#define SCENARIO_VERSION "2.0.0"
#define SCENARIO_HASH "10d9324acc0437fbefc6a3b29cb68937f45f52ff058d7a2111025180235ad9e7"
#define LOAD_PROFILE_ID "tls-diagnostic"
#define LOAD_PROFILE_VERSION "2.0.0"
#define LOAD_PROFILE_HASH "2bd9c53844fe77990a8b9888e5e260ea6979b193ef7537aba5b24b40f8253599"
#define EXECUTOR_ID "openssl-tls13-key-update-executor"
#define EXECUTOR_VERSION "0.1.0"
#define GENERATOR_ID "openssl-tls13-key-update-load"
#define GENERATOR_VERSION "0.1.0"
#define IMPLEMENTATION_ID "openssl-tls13-key-update"
#define IMPLEMENTATION_VERSION "0.1.0"
#define ALPN_VALUE "protocol-lab-tls"
#define SERVER_NAME "tls.plab.test"
#define TLS_PROFILE_ID "plab-tls13-aes128gcm-p256-server-auth-v2"
#define CERTIFICATE_PROFILE_ID "plab-single-leaf-p256-server-v2"
#define LEAF_DER_HASH "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e"
#define LEAF_SPKI_HASH "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0"
#define PRE_BYTES 1024U
#define POST_BYTES 65536U
#define PRE_OCTET 0xA5U
#define POST_OCTET 0x5AU
#define PRE_HASH "e75809e0d15667ce44e6aa5c64689a4917b245eb0920094ff0b017dc0612a17a"
#define POST_HASH "944044fe482bc4e91085c15c5a923a1b9e02eac98d3bce04997d6dbecd2a5b8d"

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

static int is_known_unsupported(const char *id) {
    size_t index;
    for (index = 0; index < sizeof(unsupported_ids) / sizeof(unsupported_ids[0]); index++) {
        if (strcmp(id, unsupported_ids[index]) == 0) {
            return 1;
        }
    }
    return 0;
}

static int ensure_exact_environment(const char *name, const char *expected) {
    const char *actual = getenv(name);
    if (actual == NULL || strcmp(actual, expected) != 0) {
        fprintf(stderr, "%s must equal %s.\n", name, expected);
        return 0;
    }
    return 1;
}

static int write_text_file(const char *root, const char *name, const char *format, ...) {
    char path[2048];
    FILE *file;
    va_list arguments;
    if (snprintf(path, sizeof(path), "%s/%s", root, name) < 0) {
        return 0;
    }
    file = fopen(path, "wb");
    if (file == NULL) {
        return 0;
    }
    va_start(arguments, format);
    if (vfprintf(file, format, arguments) < 0) {
        va_end(arguments);
        fclose(file);
        return 0;
    }
    va_end(arguments);
    {
        int write_ok = fputc('\n', file) != EOF;
        int close_ok = fclose(file) == 0;
        if (!write_ok || !close_ok) {
            return 0;
        }
    }
    return 1;
}

static int hash_buffer(const unsigned char *payload, size_t length, char output[65]) {
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int digest_length = 0;
    size_t index;
    if (EVP_Digest(payload, length, digest, &digest_length, EVP_sha256(), NULL) != 1 || digest_length != 32U) {
        return 0;
    }
    for (index = 0; index < digest_length; index++) {
        (void)snprintf(output + (index * 2U), 3U, "%02x", digest[index]);
    }
    output[64] = '\0';
    return 1;
}

static int hash_file(const char *root, const char *name, char output[65]) {
    char path[2048];
    FILE *file;
    EVP_MD_CTX *context;
    unsigned char buffer[8192];
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int digest_length = 0;
    size_t read_count;
    size_t index;
    int ok = 0;
    (void)snprintf(path, sizeof(path), "%s/%s", root, name);
    file = fopen(path, "rb");
    if (file == NULL) {
        return 0;
    }
    context = EVP_MD_CTX_new();
    if (context == NULL || EVP_DigestInit_ex(context, EVP_sha256(), NULL) != 1) {
        fclose(file);
        EVP_MD_CTX_free(context);
        return 0;
    }
    while ((read_count = fread(buffer, 1U, sizeof(buffer), file)) > 0U) {
        if (EVP_DigestUpdate(context, buffer, read_count) != 1) {
            goto cleanup;
        }
    }
    if (ferror(file) || EVP_DigestFinal_ex(context, digest, &digest_length) != 1 || digest_length != 32U) {
        goto cleanup;
    }
    for (index = 0; index < digest_length; index++) {
        (void)snprintf(output + (index * 2U), 3U, "%02x", digest[index]);
    }
    output[64] = '\0';
    ok = 1;
cleanup:
    fclose(file);
    EVP_MD_CTX_free(context);
    return ok;
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

static int parse_target(const char *target, char host[256], int *port) {
    const char *prefix = "tls://";
    const char *authority;
    const char *colon;
    size_t host_length;
    char *end = NULL;
    long parsed;
    if (strncmp(target, prefix, strlen(prefix)) != 0) {
        return 0;
    }
    authority = target + strlen(prefix);
    colon = strrchr(authority, ':');
    if (colon == NULL || colon == authority) {
        return 0;
    }
    host_length = (size_t)(colon - authority);
    if (host_length >= 256U) {
        return 0;
    }
    memcpy(host, authority, host_length);
    host[host_length] = '\0';
    parsed = strtol(colon + 1, &end, 10);
    if (*end != '\0' || parsed < 1 || parsed > 65535) {
        return 0;
    }
    *port = (int)parsed;
    return strcmp(host, "127.0.0.1") == 0;
}

static plab_socket connect_target(const char *host, int port) {
    plab_socket socket_fd = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    struct sockaddr_in address;
    if (socket_fd == PLAB_INVALID_SOCKET) {
        return PLAB_INVALID_SOCKET;
    }
    memset(&address, 0, sizeof(address));
    address.sin_family = AF_INET;
    address.sin_port = htons((uint16_t)port);
    if (inet_pton(AF_INET, host, &address.sin_addr) != 1 ||
        connect(socket_fd, (struct sockaddr *)&address, sizeof(address)) != 0) {
        plab_close_socket(socket_fd);
        return PLAB_INVALID_SOCKET;
    }
    return socket_fd;
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

static int write_frame(SSL *ssl, uint8_t stage, uint8_t octet, size_t length) {
    plab_frame_header header;
    unsigned char *payload = (unsigned char *)malloc(length);
    int ok;
    if (payload == NULL) {
        return 0;
    }
    memcpy(header.magic, "PLAB", 4U);
    header.stage = stage;
    header.flags = 0U;
    header.reserved[0] = 0U;
    header.reserved[1] = 0U;
    header.length_be = htonl((uint32_t)length);
    memset(payload, octet, length);
    ok = ssl_write_exact(ssl, &header, sizeof(header)) && ssl_write_exact(ssl, payload, length);
    free(payload);
    return ok;
}

static int read_frame(SSL *ssl, uint8_t stage, uint8_t octet, size_t length, const char *hash,
                      uint8_t *flags) {
    plab_frame_header header;
    unsigned char *payload = (unsigned char *)malloc(length);
    char actual_hash[65];
    size_t index;
    int ok = 0;
    if (payload == NULL || !ssl_read_exact(ssl, &header, sizeof(header)) ||
        memcmp(header.magic, "PLAB", 4U) != 0 || header.stage != stage ||
        ntohl(header.length_be) != length || !ssl_read_exact(ssl, payload, length)) {
        free(payload);
        return 0;
    }
    for (index = 0; index < length; index++) {
        if (payload[index] != octet) {
            goto cleanup;
        }
    }
    if (!hash_buffer(payload, length, actual_hash) || strcmp(actual_hash, hash) != 0) {
        goto cleanup;
    }
    *flags = header.flags;
    ok = 1;
cleanup:
    free(payload);
    return ok;
}

static int certificate_hashes(SSL *ssl, char der_hash[65], char spki_hash[65], int *chain_count) {
    X509 *certificate = SSL_get1_peer_certificate(ssl);
    STACK_OF(X509) *chain = SSL_get0_verified_chain(ssl);
    EVP_PKEY *public_key = NULL;
    unsigned char *der = NULL;
    unsigned char *spki = NULL;
    int der_length;
    int spki_length;
    int ok = 0;
    if (certificate == NULL || chain == NULL) {
        return 0;
    }
    *chain_count = sk_X509_num(chain) - 1;
    der_length = i2d_X509(certificate, &der);
    public_key = X509_get_pubkey(certificate);
    spki_length = public_key == NULL ? -1 : i2d_PUBKEY(public_key, &spki);
    if (der_length > 0 && spki_length > 0 && hash_buffer(der, (size_t)der_length, der_hash) &&
        hash_buffer(spki, (size_t)spki_length, spki_hash)) {
        ok = 1;
    }
    OPENSSL_free(der);
    OPENSSL_free(spki);
    EVP_PKEY_free(public_key);
    X509_free(certificate);
    return ok;
}

static void utc_now(char output[32]) {
    time_t now = time(NULL);
    struct tm value;
#ifdef _WIN32
    gmtime_s(&value, &now);
#else
    gmtime_r(&now, &value);
#endif
    (void)strftime(output, 32U, "%Y-%m-%dT%H:%M:%SZ", &value);
}

static int write_unsupported(const char *root, const char *scenario) {
    const char *format = "{\"schemaVersion\":\"protocol-lab.unsupported.v1\",\"status\":\"unsupported\",\"scenarioId\":\"%s\",\"authorityCommit\":\"%s\",\"executor\":{\"id\":\"%s\",\"version\":\"%s\"},\"reason\":\"The exact committed TLS identity is recognized but is outside this KeyUpdate-only executor.\"}";
    return write_text_file(root, "unsupported.json", format, scenario, AUTHORITY_COMMIT, EXECUTOR_ID, EXECUTOR_VERSION) &&
           write_text_file(root, "result.json", format, scenario, AUTHORITY_COMMIT, EXECUTOR_ID, EXECUTOR_VERSION);
}

static int write_artifacts(const char *root, const char *host, int port, double elapsed_ms,
                           const char *openssl_version) {
    char timestamp[32];
    char validation_hash[65];
    char protocol_hash[65];
    char negotiation_hash[65];
    char key_update_hash[65];
    char payload_hash[65];
    const char *run_id = getenv("PLAB_RUN_ID");
    const char *cell_id = getenv("PLAB_CELL_ID");
    const char *run_plan_hash = getenv("PLAB_RUN_PLAN_SHA256");
    const char *scenario_package_hash = getenv("PLAB_SCENARIO_PACKAGE_SHA256");
    const char *executor_package_hash = getenv("PLAB_EXECUTOR_PACKAGE_SHA256");
    const char *implementation_package_hash = getenv("PLAB_IMPLEMENTATION_PACKAGE_SHA256");
    if (run_id == NULL || cell_id == NULL || run_plan_hash == NULL || scenario_package_hash == NULL ||
        executor_package_hash == NULL || implementation_package_hash == NULL) {
        fprintf(stderr, "Protocol Execution Result v2 snapshot environment is missing.\n");
        return 0;
    }
    utc_now(timestamp);
    if (!write_text_file(root, "validation.json",
                         "{\"schemaVersion\":\"protocol-lab.validation.v1\",\"scenarioId\":\"%s\",\"passed\":true,\"requestedProtocol\":\"tls1.3-key-update\",\"observedProtocol\":\"tls1.3-key-update\",\"fallbackDetected\":false,\"completedOperations\":1,\"failedOperations\":0,\"timedOutOperations\":0}", SCENARIO_ID) ||
        !write_text_file(root, "result.json",
                         "{\"schemaVersion\":\"protocol-lab.validation.v1\",\"scenarioId\":\"%s\",\"passed\":true,\"completedOperations\":1,\"failedOperations\":0,\"timedOutOperations\":0}", SCENARIO_ID) ||
        !write_text_file(root, "protocol-proof.json",
                         "{\"schemaVersion\":\"protocol-lab.tls-key-update-protocol-proof.v1\",\"scenarioId\":\"%s\",\"requested\":\"tls1.3-key-update\",\"observed\":\"tls1.3-key-update\",\"fallbackAllowed\":false,\"fallbackOccurred\":false,\"tlsVersion\":\"TLS1.3\",\"cipherSuite\":\"TLS_AES_128_GCM_SHA256\",\"keyExchangeGroup\":\"X25519\",\"signatureScheme\":\"ecdsa_secp256r1_sha256\",\"alpn\":\"%s\",\"serverName\":\"%s\",\"certificateProfile\":\"%s\",\"certificateDerSha256\":\"%s\",\"certificateSpkiSha256\":\"%s\",\"certificateVerified\":true,\"sentCertificateCount\":1,\"trustAnchorSent\":false,\"resumption\":\"not-offered\",\"earlyData\":\"not-attempted\"}", SCENARIO_ID, ALPN_VALUE, SERVER_NAME, CERTIFICATE_PROFILE_ID, LEAF_DER_HASH, LEAF_SPKI_HASH) ||
        !write_text_file(root, "tls-negotiation.json",
                         "{\"schemaVersion\":\"protocol-lab.tls-negotiation.v1\",\"scenarioId\":\"%s\",\"tlsProfileId\":\"%s\",\"tlsVersion\":\"TLS1.3\",\"cipherSuite\":\"TLS_AES_128_GCM_SHA256\",\"keyExchangeGroup\":\"X25519\",\"signatureScheme\":\"ecdsa_secp256r1_sha256\",\"alpn\":\"%s\",\"serverName\":\"%s\",\"didResume\":false,\"sessionStateOffered\":false,\"earlyDataAttempted\":false,\"handshakeCompleteBeforeMeasuredWindow\":true,\"handshakeLatencyMilliseconds\":%.3f,\"opensslVersion\":\"%s\"}", SCENARIO_ID, TLS_PROFILE_ID, ALPN_VALUE, SERVER_NAME, elapsed_ms, openssl_version) ||
        !write_text_file(root, "key-update-proof.json",
                         "{\"schemaVersion\":\"protocol-lab.tls-key-update-proof.v1\",\"scenarioId\":\"%s\",\"initiator\":\"client\",\"requestedUpdates\":1,\"peerUpdateRequested\":false,\"opensslApi\":\"SSL_key_update(SSL_KEY_UPDATE_NOT_REQUESTED)\",\"apiCallSucceeded\":true,\"clientMessageCallbackSentCount\":1,\"clientMessageCallbackReceivedCount\":0,\"targetAcknowledgedReceivedCount\":1,\"keyUpdateRequestByte\":0,\"clientWriteGenerationBefore\":0,\"clientWriteGenerationAfter\":1,\"serverReadGenerationBefore\":0,\"serverReadGenerationAfter\":1,\"serverWriteGenerationBefore\":0,\"serverWriteGenerationAfter\":0,\"postUpdateBytesClientToServer\":%u,\"postUpdateBytesServerToClient\":%u,\"postUpdateDataComplete\":true,\"trafficSecretsPublished\":false,\"proofMethod\":\"openssl-api-plus-bilateral-message-callback-and-post-update-decrypt\"}", SCENARIO_ID, POST_BYTES, POST_BYTES) ||
        !write_text_file(root, "payload-hash.json",
                         "{\"schemaVersion\":\"protocol-lab.payload-hash.v1\",\"scenarioId\":\"%s\",\"generator\":\"repeated-octet\",\"preUpdate\":{\"payloadLength\":%u,\"payloadOctet\":%u,\"payloadSha256\":\"%s\"},\"postUpdate\":{\"payloadLengthPerDirection\":%u,\"payloadOctet\":%u,\"payloadSha256\":\"%s\"}}", SCENARIO_ID, PRE_BYTES, PRE_OCTET, PRE_HASH, POST_BYTES, POST_OCTET, POST_HASH) ||
        !write_text_file(root, "executor-identity.json", "{\"id\":\"%s\",\"version\":\"%s\",\"role\":\"client-test-executor\"}", EXECUTOR_ID, EXECUTOR_VERSION) ||
        !write_text_file(root, "load-generator-identity.json", "{\"id\":\"%s\",\"version\":\"%s\"}", GENERATOR_ID, GENERATOR_VERSION) ||
        !write_text_file(root, "tls-executor-result.json",
                         "{\"schemaVersion\":\"protocol-lab.tls-executor-result.v1\",\"scenarioId\":\"%s\",\"mode\":\"tls1.3-key-update\",\"executor\":{\"id\":\"%s\",\"version\":\"%s\"},\"loadGenerator\":{\"id\":\"%s\",\"version\":\"%s\"},\"validation\":{\"status\":\"passed\"},\"protocolProof\":{\"tlsVersion\":\"TLS1.3\",\"tlsProfileId\":\"%s\",\"cipherSuite\":\"TLS_AES_128_GCM_SHA256\",\"keyExchangeGroup\":\"X25519\",\"signatureScheme\":\"ecdsa_secp256r1_sha256\",\"alpn\":\"%s\",\"certificateDerSha256\":\"%s\",\"didResume\":false,\"earlyDataAttempted\":false,\"keyUpdateCount\":1,\"peerUpdateRequested\":false,\"postUpdateDataComplete\":true,\"trafficSecretsPublished\":false},\"metrics\":{\"completedOperations\":1,\"failedOperations\":0,\"timedOutOperations\":0,\"totalTransferredBytes\":%u},\"warnings\":[\"Local diagnostic evidence is non-publishable.\"]}", SCENARIO_ID, EXECUTOR_ID, EXECUTOR_VERSION, GENERATOR_ID, GENERATOR_VERSION, TLS_PROFILE_ID, ALPN_VALUE, LEAF_DER_HASH, POST_BYTES * 2U)) {
        return 0;
    }
    if (!hash_file(root, "validation.json", validation_hash) ||
        !hash_file(root, "protocol-proof.json", protocol_hash) ||
        !hash_file(root, "tls-negotiation.json", negotiation_hash) ||
        !hash_file(root, "key-update-proof.json", key_update_hash) ||
        !hash_file(root, "payload-hash.json", payload_hash)) {
        return 0;
    }
    return write_text_file(root, "protocol-execution-result-v2.json",
        "{\"schemaVersion\":\"protocol-lab.protocol-execution-result.v2\",\"resultId\":\"%s-key-update-result\",\"runBinding\":{\"runId\":\"%s\",\"cellId\":\"%s\",\"repetition\":1,\"runPlanSnapshot\":{\"id\":\"local-key-update-run-plan\",\"version\":\"1.0.0\",\"sha256\":\"%s\",\"canonicalization\":\"package-bound-sha256\"},\"scenarioSnapshot\":{\"id\":\"%s\",\"version\":\"%s\",\"sha256\":\"%s\",\"canonicalization\":\"authority-file-sha256\"},\"loadProfileSnapshot\":{\"id\":\"%s\",\"version\":\"%s\",\"sha256\":\"%s\",\"canonicalization\":\"authority-file-sha256\"},\"packageSnapshots\":[{\"id\":\"org.protocol-lab.components.scenario.tls13-handshake-performance\",\"version\":\"0.2.0\",\"sha256\":\"%s\",\"canonicalization\":\"plabpkg-sha256\"},{\"id\":\"org.protocol-lab.components.executor.openssl-tls13-key-update-executor\",\"version\":\"0.1.0\",\"sha256\":\"%s\",\"canonicalization\":\"plabpkg-sha256\"},{\"id\":\"org.protocol-lab.components.implementation.openssl-tls13-key-update\",\"version\":\"0.1.0\",\"sha256\":\"%s\",\"canonicalization\":\"plabpkg-sha256\"}]},\"selectedComponent\":{\"implementationId\":\"%s\",\"testExecutorId\":\"%s\",\"role\":\"library-backed-target\",\"executionMode\":\"process\",\"selectionProof\":\"exact package identities and extracted entrypoints\"},\"endpointProof\":{\"bindingId\":\"local-loopback-tls-key-update\",\"scheme\":\"tls\",\"authority\":\"%s:%d\",\"host\":\"%s\",\"port\":%d},\"protocolProof\":{\"requested\":\"tls1.3-key-update\",\"observed\":\"tls1.3-key-update\",\"binding\":\"tls-over-tcp\",\"protocolStack\":[\"tls1.3\",\"tcp\"],\"fallbackAllowed\":false,\"fallbackOccurred\":false},\"validation\":{\"overallOutcome\":\"pass\",\"allRequiredChecksPassed\":true,\"checks\":[{\"checkId\":\"protocol:tls1.3\",\"required\":true,\"outcome\":\"pass\"},{\"checkId\":\"key-update-sent\",\"required\":true,\"outcome\":\"pass\"},{\"checkId\":\"traffic-secret-changed\",\"required\":true,\"outcome\":\"pass\",\"detail\":\"Generation transition proven by OpenSSL API, bilateral message callbacks, and post-update decrypt; no secrets published.\"},{\"checkId\":\"post-update-data-complete\",\"required\":true,\"outcome\":\"pass\"},{\"checkId\":\"payload-sha256\",\"required\":true,\"outcome\":\"pass\"},{\"checkId\":\"zero-unexpected-failures\",\"required\":true,\"outcome\":\"pass\"},{\"checkId\":\"zero-timeouts\",\"required\":true,\"outcome\":\"pass\"}]},\"metrics\":[{\"metricId\":\"completedOperations\",\"value\":1,\"unit\":\"operations\",\"aggregation\":\"count\",\"windowStartUtc\":\"%s\",\"windowEndUtc\":\"%s\",\"source\":\"test-executor\",\"sampleCount\":1},{\"metricId\":\"failedOperations\",\"value\":0,\"unit\":\"operations\",\"aggregation\":\"count\",\"windowStartUtc\":\"%s\",\"windowEndUtc\":\"%s\",\"source\":\"test-executor\",\"sampleCount\":1},{\"metricId\":\"timedOutOperations\",\"value\":0,\"unit\":\"operations\",\"aggregation\":\"count\",\"windowStartUtc\":\"%s\",\"windowEndUtc\":\"%s\",\"source\":\"test-executor\",\"sampleCount\":1},{\"metricId\":\"totalTransferredBytes\",\"value\":%u,\"unit\":\"bytes\",\"aggregation\":\"sum\",\"windowStartUtc\":\"%s\",\"windowEndUtc\":\"%s\",\"source\":\"test-executor\",\"sampleCount\":1}],\"artifacts\":[{\"path\":\"validation.json\",\"mediaType\":\"application/json\",\"sha256\":\"%s\"},{\"path\":\"protocol-proof.json\",\"mediaType\":\"application/json\",\"sha256\":\"%s\"},{\"path\":\"tls-negotiation.json\",\"mediaType\":\"application/json\",\"sha256\":\"%s\"},{\"path\":\"key-update-proof.json\",\"mediaType\":\"application/json\",\"sha256\":\"%s\"},{\"path\":\"payload-hash.json\",\"mediaType\":\"application/json\",\"sha256\":\"%s\"}],\"familyEvidence\":{\"family\":\"tls\",\"mode\":\"key-update-diagnostic\",\"version\":\"tls1.3\",\"serverName\":\"%s\",\"alpn\":\"%s\",\"cipherSuite\":\"TLS_AES_128_GCM_SHA256\",\"keyExchangeGroup\":\"x25519\",\"signatureScheme\":\"ecdsa_secp256r1_sha256\",\"certificateDerSha256\":\"%s\",\"peerAuthentication\":\"server-certificate-required\",\"clientCertificateDerSha256\":null,\"resumption\":\"not-offered\",\"earlyData\":\"not-attempted\",\"earlyDataRetryCount\":0,\"keyUpdateCount\":1,\"recordCoverageCases\":0,\"payloadSha256\":\"%s\",\"measuredWindow\":\"key-update-and-post-update-transfer\",\"platformProvenance\":{\"os\":\"%s\",\"architecture\":\"x64\"},\"accelerationProvenance\":{\"cryptoLibrary\":\"OpenSSL\",\"cryptoLibraryVersion\":\"%s\",\"provider\":\"default\",\"cpuFeatureTelemetry\":\"unavailable\",\"hardwareAccelerationStatus\":\"not-asserted\",\"comparisonEligibility\":\"diagnostic-only\"}},\"outcome\":\"completed\"}",
        cell_id, run_id, cell_id, run_plan_hash, SCENARIO_ID, SCENARIO_VERSION, SCENARIO_HASH,
        LOAD_PROFILE_ID, LOAD_PROFILE_VERSION, LOAD_PROFILE_HASH, scenario_package_hash,
        executor_package_hash, implementation_package_hash, IMPLEMENTATION_ID, EXECUTOR_ID,
        host, port, host, port, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp,
        POST_BYTES * 2U, timestamp, timestamp, validation_hash, protocol_hash, negotiation_hash,
        key_update_hash, payload_hash, SERVER_NAME, ALPN_VALUE, LEAF_DER_HASH, POST_HASH,
        PLATFORM_OS, openssl_version);
}

static int self_test(void) {
    unsigned char *payload = (unsigned char *)malloc(POST_BYTES);
    char hash[65];
    int ok;
    if (payload == NULL) {
        return 1;
    }
    memset(payload, POST_OCTET, POST_BYTES);
    ok = hash_buffer(payload, POST_BYTES, hash) && strcmp(hash, POST_HASH) == 0 &&
         is_known_unsupported("tls.handshake.full") &&
         !is_known_unsupported("tls.key-update.unknown") &&
         strcmp(SCENARIO_ID, "tls.key-update.diagnostic") == 0;
    free(payload);
    if (!ok) {
        fprintf(stderr, "self-test failed\n");
        return 1;
    }
    puts("self-test passed");
    return 0;
}

int main(int argc, char **argv) {
    const char *scenario = getenv("PLAB_SCENARIO_ID");
    const char *artifact_root = getenv("PLAB_ARTIFACT_DIR");
    const char *target = getenv("PLAB_TARGET_BASE_URL");
    const char *root_certificate = getenv("PLAB_TLS_ROOT_CERTIFICATE_PATH");
    char host[256];
    int port = 0;
    plab_socket socket_fd = PLAB_INVALID_SOCKET;
    SSL_CTX *context = NULL;
    SSL *ssl = NULL;
    key_update_observation observation = {0U, 0U, -1, -1};
    unsigned char alpn_wire[] = {16U, 'p','r','o','t','o','c','o','l','-','l','a','b','-','t','l','s'};
    const unsigned char *selected_alpn = NULL;
    unsigned int selected_alpn_length = 0;
    uint8_t flags = 0U;
    char der_hash[65];
    char spki_hash[65];
    int chain_count = 0;
    int signature_nid = NID_undef;
    int signature_type_nid = NID_undef;
    const char *group_name;
    const char *cipher_name;
    clock_t handshake_start;
    clock_t handshake_end;
    double handshake_ms;
    int exit_code = 1;
    if (argc == 2 && strcmp(argv[1], "--self-test") == 0) {
        return self_test();
    }
    if (scenario == NULL || artifact_root == NULL) {
        fprintf(stderr, "Scenario and artifact directory are required.\n");
        return 2;
    }
    if (strcmp(scenario, SCENARIO_ID) != 0) {
        if (is_known_unsupported(scenario)) {
            return write_unsupported(artifact_root, scenario) ? 3 : 1;
        }
        fprintf(stderr, "unknown TLS scenario %s\n", scenario);
        return 2;
    }
    if (target == NULL || root_certificate == NULL || !parse_target(target, host, &port) ||
        !ensure_exact_environment("PLAB_EXECUTOR_ID", EXECUTOR_ID) ||
        !ensure_exact_environment("PLAB_EXECUTOR_VERSION", EXECUTOR_VERSION) ||
        !ensure_exact_environment("PLAB_LOAD_GENERATOR_ID", GENERATOR_ID) ||
        !ensure_exact_environment("PLAB_LOAD_GENERATOR_VERSION", GENERATOR_VERSION) ||
        !ensure_exact_environment("PLAB_PROTOCOL", "tls") ||
        !ensure_exact_environment("PLAB_PROTOCOL_VARIANT", "tls1.3-key-update") ||
        !ensure_exact_environment("PLAB_LOAD_PROFILE_ID", LOAD_PROFILE_ID)) {
        return 2;
    }
    if (!initialize_sockets()) {
        fprintf(stderr, "Socket initialization failed.\n");
        return 1;
    }
    SSL_load_error_strings();
    OPENSSL_init_ssl(0, NULL);
    context = SSL_CTX_new(TLS_client_method());
    if (context == NULL || SSL_CTX_set_min_proto_version(context, TLS1_3_VERSION) != 1 ||
        SSL_CTX_set_max_proto_version(context, TLS1_3_VERSION) != 1 ||
        SSL_CTX_set_ciphersuites(context, "TLS_AES_128_GCM_SHA256") != 1 ||
        SSL_CTX_set1_groups_list(context, "X25519") != 1 ||
        SSL_CTX_set1_sigalgs_list(context, "ecdsa_secp256r1_sha256") != 1 ||
        SSL_CTX_load_verify_locations(context, root_certificate, NULL) != 1) {
        ERR_print_errors_fp(stderr);
        goto cleanup;
    }
    SSL_CTX_set_verify(context, SSL_VERIFY_PEER, NULL);
    SSL_CTX_set_session_cache_mode(context, SSL_SESS_CACHE_OFF);
    SSL_CTX_set_max_early_data(context, 0);
    socket_fd = connect_target(host, port);
    ssl = SSL_new(context);
    if (socket_fd == PLAB_INVALID_SOCKET || ssl == NULL || SSL_set_fd(ssl, (int)socket_fd) != 1 ||
        SSL_set_tlsext_host_name(ssl, SERVER_NAME) != 1 || SSL_set1_host(ssl, SERVER_NAME) != 1 ||
        SSL_set_alpn_protos(ssl, alpn_wire, sizeof(alpn_wire)) != 0) {
        ERR_print_errors_fp(stderr);
        goto cleanup;
    }
    SSL_set_msg_callback(ssl, message_callback);
    SSL_set_msg_callback_arg(ssl, &observation);
    handshake_start = clock();
    if (SSL_connect(ssl) != 1) {
        ERR_print_errors_fp(stderr);
        goto cleanup;
    }
    handshake_end = clock();
    handshake_ms = ((double)(handshake_end - handshake_start) * 1000.0) / (double)CLOCKS_PER_SEC;
    SSL_get0_alpn_selected(ssl, &selected_alpn, &selected_alpn_length);
    group_name = SSL_get0_group_name(ssl);
    cipher_name = SSL_CIPHER_get_name(SSL_get_current_cipher(ssl));
    if (!certificate_hashes(ssl, der_hash, spki_hash, &chain_count) ||
        SSL_get_peer_signature_nid(ssl, &signature_nid) != 1 ||
        SSL_get_peer_signature_type_nid(ssl, &signature_type_nid) != 1 ||
        SSL_version(ssl) != TLS1_3_VERSION || strcmp(cipher_name, "TLS_AES_128_GCM_SHA256") != 0 ||
        group_name == NULL || strcmp(group_name, "x25519") != 0 ||
        signature_nid != NID_sha256 || signature_type_nid != EVP_PKEY_EC ||
        selected_alpn_length != sizeof(ALPN_VALUE) - 1U ||
        memcmp(selected_alpn, ALPN_VALUE, selected_alpn_length) != 0 ||
        SSL_get_verify_result(ssl) != X509_V_OK || strcmp(der_hash, LEAF_DER_HASH) != 0 ||
        strcmp(spki_hash, LEAF_SPKI_HASH) != 0 || chain_count != 1 || SSL_session_reused(ssl) != 0 ||
        SSL_get_early_data_status(ssl) != SSL_EARLY_DATA_NOT_SENT) {
        fprintf(stderr, "Exact TLS negotiation proof failed. group=%s signature=%d type=%d chain=%d\n",
                group_name == NULL ? "" : group_name, signature_nid, signature_type_nid, chain_count);
        goto cleanup;
    }
    if (!write_frame(ssl, 1U, PRE_OCTET, PRE_BYTES) ||
        !read_frame(ssl, 1U, PRE_OCTET, PRE_BYTES, PRE_HASH, &flags) || flags != 0U ||
        observation.sent != 0U || observation.received != 0U) {
        fprintf(stderr, "Pre-update deterministic exchange failed.\n");
        goto cleanup;
    }
    if (SSL_key_update(ssl, SSL_KEY_UPDATE_NOT_REQUESTED) != 1 ||
        !write_frame(ssl, 2U, POST_OCTET, POST_BYTES) ||
        !read_frame(ssl, 2U, POST_OCTET, POST_BYTES, POST_HASH, &flags)) {
        ERR_print_errors_fp(stderr);
        fprintf(stderr, "KeyUpdate or post-update exchange failed.\n");
        goto cleanup;
    }
    if (observation.sent != 1U || observation.sent_request != SSL_KEY_UPDATE_NOT_REQUESTED ||
        observation.received != 0U || flags != 1U) {
        fprintf(stderr, "Bilateral KeyUpdate observation failed: sent=%u request=%d received=%u target=%u\n",
                observation.sent, observation.sent_request, observation.received, flags);
        goto cleanup;
    }
    if (!write_artifacts(artifact_root, host, port, handshake_ms, OpenSSL_version(OPENSSL_VERSION))) {
        fprintf(stderr, "Required artifact generation failed.\n");
        goto cleanup;
    }
    exit_code = 0;
cleanup:
    if (ssl != NULL) {
        (void)SSL_shutdown(ssl);
    }
    SSL_free(ssl);
    SSL_CTX_free(context);
    if (socket_fd != PLAB_INVALID_SOCKET) {
        plab_close_socket(socket_fd);
    }
    cleanup_sockets();
    return exit_code;
}
