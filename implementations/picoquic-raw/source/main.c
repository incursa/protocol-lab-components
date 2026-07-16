#include <errno.h>
#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <picoquic.h>
#include <picoquic_packet_loop.h>

#define PLAB_ALPN "plab-raw-quic"
#define PLAB_DEFAULT_PORT 5450
#define PLAB_DEFAULT_ECHO_MAX (64U * 1024U)
#define PLAB_MAX_READ_BYTES (64U * 1024U * 1024U)
#define PLAB_DOWNLOAD_MAGIC "PLAB-DL1"
#define PLAB_DOWNLOAD_MAGIC_LENGTH 8U
#define PLAB_DOWNLOAD_REQUEST_LENGTH 16U
#define PLAB_INTERNAL_ERROR 0x504c4142U

typedef struct plab_stream_context_s {
    struct plab_stream_context_s* next;
    struct plab_stream_context_s* previous;
    uint64_t stream_id;
    uint8_t* input;
    size_t input_length;
    size_t input_capacity;
    int response_queued;
} plab_stream_context_t;

typedef struct plab_connection_context_s {
    size_t echo_max;
    int is_default;
    plab_stream_context_t* first_stream;
    plab_stream_context_t* last_stream;
} plab_connection_context_t;

static size_t plab_echo_max_for_scenario(const char* scenario)
{
    if (scenario == NULL || scenario[0] == 0) {
        return PLAB_DEFAULT_ECHO_MAX;
    }
    if (strcmp(scenario, "quic.transport.handshake-cold") == 0 ||
        strcmp(scenario, "quic.transport.stream-throughput.1mb") == 0) {
        return 0;
    }
    if (strcmp(scenario, "quic.transport.latency.echo-1kb") == 0) {
        return 1024;
    }
    return PLAB_DEFAULT_ECHO_MAX;
}

static int plab_parse_download_request(const uint8_t* bytes, size_t length, size_t* download_length)
{
    uint64_t value = 0;
    size_t i;

    if (bytes == NULL || download_length == NULL || length != PLAB_DOWNLOAD_REQUEST_LENGTH ||
        memcmp(bytes, PLAB_DOWNLOAD_MAGIC, PLAB_DOWNLOAD_MAGIC_LENGTH) != 0) {
        return 0;
    }
    for (i = PLAB_DOWNLOAD_MAGIC_LENGTH; i < PLAB_DOWNLOAD_REQUEST_LENGTH; i++) {
        value = (value << 8) | bytes[i];
    }
    if (value == 0 || value > PLAB_MAX_READ_BYTES) {
        return 0;
    }
    *download_length = (size_t)value;
    return 1;
}

static plab_stream_context_t* plab_create_stream(plab_connection_context_t* connection, uint64_t stream_id)
{
    plab_stream_context_t* stream = (plab_stream_context_t*)calloc(1, sizeof(plab_stream_context_t));
    if (stream == NULL) {
        return NULL;
    }
    stream->stream_id = stream_id;
    stream->previous = connection->last_stream;
    if (connection->last_stream == NULL) {
        connection->first_stream = stream;
    }
    else {
        connection->last_stream->next = stream;
    }
    connection->last_stream = stream;
    return stream;
}

static void plab_delete_stream(plab_connection_context_t* connection, plab_stream_context_t* stream)
{
    if (stream == NULL) {
        return;
    }
    if (stream->previous == NULL) {
        connection->first_stream = stream->next;
    }
    else {
        stream->previous->next = stream->next;
    }
    if (stream->next == NULL) {
        connection->last_stream = stream->previous;
    }
    else {
        stream->next->previous = stream->previous;
    }
    free(stream->input);
    free(stream);
}

static void plab_delete_connection(plab_connection_context_t* connection)
{
    while (connection->first_stream != NULL) {
        plab_delete_stream(connection, connection->first_stream);
    }
    free(connection);
}

static int plab_append_input(plab_stream_context_t* stream, const uint8_t* bytes, size_t length)
{
    size_t required;
    size_t capacity;
    uint8_t* resized;

    if (length == 0) {
        return 0;
    }
    if (stream->input_length > PLAB_MAX_READ_BYTES - length) {
        return -1;
    }
    required = stream->input_length + length;
    if (required > stream->input_capacity) {
        capacity = stream->input_capacity == 0 ? 4096 : stream->input_capacity;
        while (capacity < required) {
            size_t next = capacity * 2;
            capacity = next > PLAB_MAX_READ_BYTES ? PLAB_MAX_READ_BYTES : next;
            if (capacity < required && capacity == PLAB_MAX_READ_BYTES) {
                return -1;
            }
        }
        resized = (uint8_t*)realloc(stream->input, capacity);
        if (resized == NULL) {
            return -1;
        }
        stream->input = resized;
        stream->input_capacity = capacity;
    }
    memcpy(stream->input + stream->input_length, bytes, length);
    stream->input_length = required;
    return 0;
}

static int plab_queue_response(picoquic_cnx_t* cnx, plab_connection_context_t* connection,
    plab_stream_context_t* stream)
{
    size_t download_length = 0;
    const uint8_t* response = NULL;
    size_t response_length = 0;
    uint8_t* download = NULL;
    int ret;

    if (plab_parse_download_request(stream->input, stream->input_length, &download_length)) {
        size_t i;
        download = (uint8_t*)malloc(download_length);
        if (download == NULL) {
            return -1;
        }
        for (i = 0; i < download_length; i++) {
            download[i] = (uint8_t)(i % 251);
        }
        response = download;
        response_length = download_length;
    }
    else if (stream->input_length > 0 && stream->input_length <= connection->echo_max) {
        response = stream->input;
        response_length = stream->input_length;
    }

    ret = picoquic_add_to_stream_with_ctx(cnx, stream->stream_id, response, response_length, 1, stream);
    free(download);
    if (ret == 0) {
        stream->response_queued = 1;
        free(stream->input);
        stream->input = NULL;
        stream->input_length = 0;
        stream->input_capacity = 0;
    }
    return ret;
}

static int plab_callback(picoquic_cnx_t* cnx, uint64_t stream_id, uint8_t* bytes, size_t length,
    picoquic_call_back_event_t event, void* callback_context, void* stream_context)
{
    picoquic_quic_t* quic = picoquic_get_quic_ctx(cnx);
    plab_connection_context_t* connection = (plab_connection_context_t*)callback_context;
    plab_stream_context_t* stream = (plab_stream_context_t*)stream_context;

    if (connection == NULL || connection == picoquic_get_default_callback_context(quic) || connection->is_default) {
        plab_connection_context_t* defaults = (plab_connection_context_t*)picoquic_get_default_callback_context(quic);
        connection = (plab_connection_context_t*)calloc(1, sizeof(plab_connection_context_t));
        if (connection == NULL) {
            (void)picoquic_close(cnx, PLAB_INTERNAL_ERROR);
            return -1;
        }
        connection->echo_max = defaults == NULL ? PLAB_DEFAULT_ECHO_MAX : defaults->echo_max;
        picoquic_set_callback(cnx, plab_callback, connection);
    }

    switch (event) {
    case picoquic_callback_stream_data:
    case picoquic_callback_stream_fin:
        if (stream == NULL) {
            stream = plab_create_stream(connection, stream_id);
            if (stream == NULL || picoquic_set_app_stream_ctx(cnx, stream_id, stream) != 0) {
                (void)picoquic_reset_stream(cnx, stream_id, PLAB_INTERNAL_ERROR);
                return -1;
            }
        }
        if (stream->response_queued || plab_append_input(stream, bytes, length) != 0) {
            (void)picoquic_reset_stream(cnx, stream_id, PLAB_INTERNAL_ERROR);
            return -1;
        }
        if (event == picoquic_callback_stream_fin && plab_queue_response(cnx, connection, stream) != 0) {
            (void)picoquic_reset_stream(cnx, stream_id, PLAB_INTERNAL_ERROR);
            return -1;
        }
        break;
    case picoquic_callback_stream_reset:
    case picoquic_callback_stop_sending:
        if (stream != NULL) {
            picoquic_unlink_app_stream_ctx(cnx, stream_id);
            plab_delete_stream(connection, stream);
        }
        break;
    case picoquic_callback_stream_released:
        if (stream != NULL) {
            plab_delete_stream(connection, stream);
        }
        break;
    case picoquic_callback_stateless_reset:
    case picoquic_callback_close:
    case picoquic_callback_application_close:
        plab_delete_connection(connection);
        picoquic_set_callback(cnx, NULL, NULL);
        break;
    default:
        break;
    }
    return 0;
}

static int plab_self_test(void)
{
    uint8_t request[PLAB_DOWNLOAD_REQUEST_LENGTH] = PLAB_DOWNLOAD_MAGIC;
    size_t parsed = 0;
    request[12] = 0x00;
    request[13] = 0x10;
    request[14] = 0x00;
    request[15] = 0x00;
    if (!plab_parse_download_request(request, sizeof(request), &parsed) || parsed != 1024U * 1024U) {
        fprintf(stderr, "download prelude self-test failed\n");
        return 1;
    }
    if (plab_echo_max_for_scenario("quic.transport.handshake-cold") != 0 ||
        plab_echo_max_for_scenario("quic.transport.latency.echo-1kb") != 1024 ||
        plab_echo_max_for_scenario("quic.transport.multiplex.100x64kb") != PLAB_DEFAULT_ECHO_MAX) {
        fprintf(stderr, "scenario mapping self-test failed\n");
        return 1;
    }
    printf("picoquic raw adapter self-test passed\n");
    return 0;
}

int main(int argc, char** argv)
{
    const char* scenario;
    const char* port_text;
    const char* cert_file;
    const char* key_file;
    const char* advertise_host;
    char* port_end = NULL;
    long port_value;
    uint16_t port;
    uint64_t current_time;
    picoquic_quic_t* quic;
    plab_connection_context_t defaults = { 0 };
    int ret;

    if (argc == 2 && strcmp(argv[1], "--self-test") == 0) {
        return plab_self_test();
    }
    if (argc != 1) {
        fprintf(stderr, "usage: protocol-lab-picoquic-raw [--self-test]\n");
        return 2;
    }

    scenario = getenv("PLAB_SCENARIO_ID");
    port_text = getenv("PLAB_QUIC_PORT");
    cert_file = getenv("PLAB_CERT_FILE");
    key_file = getenv("PLAB_KEY_FILE");
    advertise_host = getenv("PROTOCOL_LAB_TARGET_ADVERTISE_HOST");
    if (port_text == NULL || port_text[0] == 0) {
        port_text = "5450";
    }
    errno = 0;
    port_value = strtol(port_text, &port_end, 10);
    if (errno != 0 || port_end == port_text || *port_end != 0 || port_value < 1 || port_value > 65535) {
        fprintf(stderr, "invalid PLAB_QUIC_PORT: %s\n", port_text);
        return 2;
    }
    port = (uint16_t)port_value;
    if (cert_file == NULL || cert_file[0] == 0) {
        cert_file = "certs/cert.pem";
    }
    if (key_file == NULL || key_file[0] == 0) {
        key_file = "certs/key.pem";
    }
    if (advertise_host == NULL) {
        advertise_host = "";
    }

    defaults.echo_max = plab_echo_max_for_scenario(scenario);
    defaults.is_default = 1;
    picoquic_register_all_congestion_control_algorithms();
    current_time = picoquic_current_time();
    quic = picoquic_create(128, cert_file, key_file, NULL, PLAB_ALPN, plab_callback,
        &defaults, NULL, NULL, NULL, current_time, NULL, NULL, NULL, 0);
    if (quic == NULL) {
        fprintf(stderr, "could not create picoquic server context\n");
        return 1;
    }

    ret = picoquic_set_default_tp_value(quic, picoquic_tp_initial_max_streams_bidi, 1024);
    if (ret == 0) {
        ret = picoquic_set_default_tp_value(quic, picoquic_tp_initial_max_stream_data_bidi_remote, PLAB_MAX_READ_BYTES);
    }
    if (ret == 0) {
        ret = picoquic_set_default_tp_value(quic, picoquic_tp_initial_max_data, 128U * 1024U * 1024U);
    }
    if (ret != 0) {
        fprintf(stderr, "could not configure picoquic transport parameters: %d\n", ret);
        picoquic_free(quic);
        return 1;
    }

    printf("{\"status\":\"ready\",\"implementationId\":\"picoquic-raw\","
           "\"packageId\":\"org.protocol-lab.components.implementation.picoquic-raw\","
           "\"protocol\":\"quic\",\"alpn\":\"%s\",\"port\":%u,"
           "\"advertiseHost\":\"%s\",\"picoquicCommit\":\"13671ce7bdf58c278a29da2d49a32f76c21d6c6d\","
           "\"processId\":%ld}\n",
        PLAB_ALPN, (unsigned int)port, advertise_host, (long)getpid());
    fflush(stdout);
    fprintf(stderr, "picoquic raw QUIC target listening on UDP port %u\n", (unsigned int)port);

    ret = picoquic_packet_loop(quic, port, 0, 0, 0, 0, NULL, NULL);
    fprintf(stderr, "picoquic packet loop exited: %d\n", ret);
    picoquic_free(quic);
    return ret == 0 ? 0 : 1;
}
