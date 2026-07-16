// The UDP connection loop follows Cloudflare quiche's BSD-2-Clause server
// example. ProtocolLab-specific code is limited to configuration, stream
// accumulation, and the canonical raw application wire behavior.

#[macro_use]
extern crate log;

use ring::rand::*;
use std::collections::HashMap;
use std::env;
use std::net;

const IMPLEMENTATION_ID: &str = "quiche-raw";
const PACKAGE_ID: &str = "org.protocol-lab.components.implementation.quiche-raw";
const ALPN: &[u8] = b"plab-raw-quic";
const DEFAULT_PORT: u16 = 5452;
const MAX_DATAGRAM_SIZE: usize = 1350;
const MAX_READ_BYTES: usize = 64 * 1024 * 1024;
const DEFAULT_ECHO_MAX: usize = 64 * 1024;
const DOWNLOAD_MAGIC: &[u8] = b"PLAB-DL1";

struct PartialResponse {
    body: Vec<u8>,
    written: usize,
}

struct Client {
    conn: quiche::Connection,
    requests: HashMap<u64, Vec<u8>>,
    partial_responses: HashMap<u64, PartialResponse>,
}

type ClientMap = HashMap<quiche::ConnectionId<'static>, Client>;

fn echo_max_for_scenario(scenario: &str) -> usize {
    match scenario {
        "quic.transport.handshake-cold" | "quic.transport.stream-throughput.1mb" => 0,
        "quic.transport.latency.echo-1kb" => 1024,
        _ => DEFAULT_ECHO_MAX,
    }
}

fn parse_download_request(payload: &[u8]) -> Option<usize> {
    if payload.len() != DOWNLOAD_MAGIC.len() + 8 || !payload.starts_with(DOWNLOAD_MAGIC) {
        return None;
    }

    let length = u64::from_be_bytes(payload[DOWNLOAD_MAGIC.len()..].try_into().ok()?);
    if length == 0 || length > MAX_READ_BYTES as u64 {
        return None;
    }

    Some(length as usize)
}

fn deterministic_payload(length: usize) -> Vec<u8> {
    (0..length).map(|index| (index % 251) as u8).collect()
}

fn self_test() {
    let mut request = DOWNLOAD_MAGIC.to_vec();
    request.extend_from_slice(&(1024_u64 * 1024).to_be_bytes());
    assert_eq!(parse_download_request(&request), Some(1024 * 1024));
    request.push(b'x');
    assert_eq!(parse_download_request(&request), None);
    assert_eq!(
        deterministic_payload(502),
        [
            (0_u8..=250).collect::<Vec<_>>(),
            (0_u8..=250).collect::<Vec<_>>(),
        ]
        .concat()
    );
    assert_eq!(echo_max_for_scenario("quic.transport.handshake-cold"), 0);
    assert_eq!(
        echo_max_for_scenario("quic.transport.latency.echo-1kb"),
        1024
    );
    assert_eq!(
        echo_max_for_scenario("quic.transport.multiplex.100x64kb"),
        64 * 1024
    );
    println!("quiche raw adapter self-test passed");
}

fn main() {
    if env::args().any(|arg| arg == "--self-test") {
        self_test();
        return;
    }

    env_logger::init();

    let host =
        env::var("PROTOCOL_LAB_TARGET_BIND_ADDRESS").unwrap_or_else(|_| "0.0.0.0".to_string());
    let port = env::var("PLAB_QUIC_PORT")
        .ok()
        .and_then(|value| value.parse::<u16>().ok())
        .unwrap_or(DEFAULT_PORT);
    let bind_addr: net::SocketAddr = format!("{host}:{port}").parse().unwrap();
    let cert = env::var("PLAB_CERT_FILE").unwrap_or_else(|_| "certs/leaf.pem".to_string());
    let key = env::var("PLAB_KEY_FILE").unwrap_or_else(|_| "certs/leaf-key.pem".to_string());
    let scenario = env::var("PLAB_SCENARIO_ID").unwrap_or_default();
    let echo_max = echo_max_for_scenario(&scenario);

    let mut buf = [0; 65535];
    let mut out = [0; MAX_DATAGRAM_SIZE];

    let mut poll = mio::Poll::new().unwrap();
    let mut events = mio::Events::with_capacity(1024);
    let mut socket = mio::net::UdpSocket::bind(bind_addr).unwrap();
    poll.registry()
        .register(&mut socket, mio::Token(0), mio::Interest::READABLE)
        .unwrap();

    let mut config = quiche::Config::new(quiche::PROTOCOL_VERSION).unwrap();
    config.load_cert_chain_from_pem_file(&cert).unwrap();
    config.load_priv_key_from_pem_file(&key).unwrap();
    config.set_application_protos(&[ALPN]).unwrap();
    config.set_max_idle_timeout(30_000);
    config.set_max_recv_udp_payload_size(MAX_DATAGRAM_SIZE);
    config.set_max_send_udp_payload_size(MAX_DATAGRAM_SIZE);
    config.set_initial_max_data(128 * 1024 * 1024);
    config.set_initial_max_stream_data_bidi_local(64 * 1024 * 1024);
    config.set_initial_max_stream_data_bidi_remote(64 * 1024 * 1024);
    config.set_initial_max_stream_data_uni(64 * 1024 * 1024);
    config.set_initial_max_streams_bidi(256);
    config.set_initial_max_streams_uni(16);
    config.set_disable_active_migration(true);

    let rng = SystemRandom::new();
    let conn_id_seed = ring::hmac::Key::generate(ring::hmac::HMAC_SHA256, &rng).unwrap();
    let mut clients = ClientMap::new();
    let local_addr = socket.local_addr().unwrap();

    println!(
        "{{\"status\":\"ready\",\"implementationId\":\"{}\",\"packageId\":\"{}\",\"protocol\":\"quic\",\"alpn\":\"{}\",\"listen\":\"{}\",\"quicheVersion\":\"0.29.3\",\"processId\":{},\"supportedScenarios\":[\"quic.transport.handshake-cold\",\"quic.transport.latency.echo-1kb\",\"quic.transport.stream-throughput.1mb\",\"quic.transport.multiplex.100x64kb\",\"quic.transport.duplex-streams\"]}}",
        IMPLEMENTATION_ID,
        PACKAGE_ID,
        String::from_utf8_lossy(ALPN),
        local_addr,
        std::process::id()
    );

    loop {
        let timeout = clients.values().filter_map(|c| c.conn.timeout()).min();
        poll.poll(&mut events, timeout).unwrap();

        'read: loop {
            if events.is_empty() {
                clients.values_mut().for_each(|c| c.conn.on_timeout());
                break 'read;
            }

            let (len, from) = match socket.recv_from(&mut buf) {
                Ok(value) => value,
                Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {
                    break 'read;
                }
                Err(error) => panic!("recv() failed: {error:?}"),
            };

            let pkt_buf = &mut buf[..len];
            let hdr = match quiche::Header::from_slice(pkt_buf, quiche::MAX_CONN_ID_LEN) {
                Ok(value) => value,
                Err(error) => {
                    error!("Parsing packet header failed: {error:?}");
                    continue 'read;
                }
            };

            let derived_conn_id = ring::hmac::sign(&conn_id_seed, &hdr.dcid);
            let derived_conn_id = &derived_conn_id.as_ref()[..quiche::MAX_CONN_ID_LEN];
            let derived_conn_id: quiche::ConnectionId<'static> = derived_conn_id.to_vec().into();

            let client = if !clients.contains_key(&hdr.dcid)
                && !clients.contains_key(&derived_conn_id)
            {
                if hdr.ty != quiche::Type::Initial {
                    continue 'read;
                }

                if !quiche::version_is_supported(hdr.version) {
                    let write = quiche::negotiate_version(&hdr.scid, &hdr.dcid, &mut out).unwrap();
                    if let Err(error) = socket.send_to(&out[..write], from) {
                        if error.kind() == std::io::ErrorKind::WouldBlock {
                            break 'read;
                        }
                        panic!("version negotiation send failed: {error:?}");
                    }
                    continue 'read;
                }

                let mut scid_bytes = [0; quiche::MAX_CONN_ID_LEN];
                scid_bytes.copy_from_slice(&derived_conn_id);
                let scid = quiche::ConnectionId::from_ref(&scid_bytes);
                let token = hdr.token.as_ref().unwrap();

                if token.is_empty() {
                    let new_token = mint_token(&hdr, &from);
                    let write = quiche::retry(
                        &hdr.scid,
                        &hdr.dcid,
                        &scid,
                        &new_token,
                        hdr.version,
                        &mut out,
                    )
                    .unwrap();
                    if let Err(error) = socket.send_to(&out[..write], from) {
                        if error.kind() == std::io::ErrorKind::WouldBlock {
                            break 'read;
                        }
                        panic!("retry send failed: {error:?}");
                    }
                    continue 'read;
                }

                let Some(odcid) = validate_token(&from, token) else {
                    continue 'read;
                };
                if scid.len() != hdr.dcid.len() {
                    continue 'read;
                }

                let scid = hdr.dcid.clone();
                let conn =
                    quiche::accept(&scid, Some(&odcid), local_addr, from, &mut config).unwrap();
                clients.insert(
                    scid.clone(),
                    Client {
                        conn,
                        requests: HashMap::new(),
                        partial_responses: HashMap::new(),
                    },
                );
                clients.get_mut(&scid).unwrap()
            } else if clients.contains_key(&hdr.dcid) {
                clients.get_mut(&hdr.dcid).unwrap()
            } else {
                clients.get_mut(&derived_conn_id).unwrap()
            };

            let recv_info = quiche::RecvInfo {
                to: socket.local_addr().unwrap(),
                from,
            };
            if let Err(error) = client.conn.recv(pkt_buf, recv_info) {
                error!("{} recv failed: {error:?}", client.conn.trace_id());
                continue 'read;
            }

            if client.conn.is_established() {
                let writable: Vec<u64> = client.conn.writable().collect();
                for stream_id in writable {
                    handle_writable(client, stream_id);
                }

                let readable: Vec<u64> = client.conn.readable().collect();
                for stream_id in readable {
                    loop {
                        match client.conn.stream_recv(stream_id, &mut buf) {
                            Ok((read, fin)) => {
                                handle_stream_chunk(client, stream_id, &buf[..read], fin, echo_max);
                            }
                            Err(quiche::Error::Done) => break,
                            Err(error) => {
                                error!(
                                    "{} stream {stream_id} recv failed: {error:?}",
                                    client.conn.trace_id()
                                );
                                break;
                            }
                        }
                    }
                }
            }
        }

        for client in clients.values_mut() {
            loop {
                let (write, send_info) = match client.conn.send(&mut out) {
                    Ok(value) => value,
                    Err(quiche::Error::Done) => break,
                    Err(error) => {
                        error!("{} send failed: {error:?}", client.conn.trace_id());
                        client.conn.close(false, 0x1, b"fail").ok();
                        break;
                    }
                };

                if let Err(error) = socket.send_to(&out[..write], send_info.to) {
                    if error.kind() == std::io::ErrorKind::WouldBlock {
                        break;
                    }
                    panic!("send() failed: {error:?}");
                }
            }
        }

        clients.retain(|_, client| !client.conn.is_closed());
    }
}

fn mint_token(hdr: &quiche::Header, src: &net::SocketAddr) -> Vec<u8> {
    let mut token = Vec::new();
    token.extend_from_slice(b"quiche");
    match src.ip() {
        net::IpAddr::V4(address) => token.extend_from_slice(&address.octets()),
        net::IpAddr::V6(address) => token.extend_from_slice(&address.octets()),
    }
    token.extend_from_slice(&hdr.dcid);
    token
}

fn validate_token<'a>(src: &net::SocketAddr, token: &'a [u8]) -> Option<quiche::ConnectionId<'a>> {
    if token.len() < 6 || &token[..6] != b"quiche" {
        return None;
    }
    let token = &token[6..];
    let address = match src.ip() {
        net::IpAddr::V4(value) => value.octets().to_vec(),
        net::IpAddr::V6(value) => value.octets().to_vec(),
    };
    if token.len() < address.len() || &token[..address.len()] != address {
        return None;
    }
    Some(quiche::ConnectionId::from_ref(&token[address.len()..]))
}

fn handle_stream_chunk(
    client: &mut Client,
    stream_id: u64,
    chunk: &[u8],
    fin: bool,
    echo_max: usize,
) {
    let request = client.requests.entry(stream_id).or_default();
    if request.len() + chunk.len() > MAX_READ_BYTES {
        client.requests.remove(&stream_id);
        client
            .conn
            .stream_shutdown(stream_id, quiche::Shutdown::Read, 0x504c4142)
            .ok();
        client
            .conn
            .stream_shutdown(stream_id, quiche::Shutdown::Write, 0x504c4142)
            .ok();
        return;
    }
    request.extend_from_slice(chunk);
    if !fin {
        return;
    }

    let request = client.requests.remove(&stream_id).unwrap_or_default();
    let response = if let Some(length) = parse_download_request(&request) {
        deterministic_payload(length)
    } else if !request.is_empty() && request.len() <= echo_max {
        request
    } else {
        Vec::new()
    };
    send_response(client, stream_id, response);
}

fn send_response(client: &mut Client, stream_id: u64, body: Vec<u8>) {
    let written = match client.conn.stream_send(stream_id, &body, true) {
        Ok(value) => value,
        Err(quiche::Error::Done) => 0,
        Err(error) => {
            error!(
                "{} stream {stream_id} send failed: {error:?}",
                client.conn.trace_id()
            );
            return;
        }
    };

    if written < body.len() {
        client
            .partial_responses
            .insert(stream_id, PartialResponse { body, written });
    }
}

fn handle_writable(client: &mut Client, stream_id: u64) {
    let Some(response) = client.partial_responses.get_mut(&stream_id) else {
        return;
    };
    let body = &response.body[response.written..];
    let written = match client.conn.stream_send(stream_id, body, true) {
        Ok(value) => value,
        Err(quiche::Error::Done) => 0,
        Err(error) => {
            error!(
                "{} stream {stream_id} continuation failed: {error:?}",
                client.conn.trace_id()
            );
            client.partial_responses.remove(&stream_id);
            return;
        }
    };
    response.written += written;
    if response.written == response.body.len() {
        client.partial_responses.remove(&stream_id);
    }
}
