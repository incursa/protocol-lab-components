use std::{env, net::SocketAddr, sync::Arc};

use anyhow::{Context, Result};
use quinn::{crypto::rustls::QuicServerConfig, Endpoint, Incoming, TransportConfig, VarInt};
use quinn::rustls::pki_types::{CertificateDer, PrivateKeyDer, PrivatePkcs8KeyDer};
use rcgen::{generate_simple_self_signed, CertifiedKey};
use serde::Serialize;

const IMPLEMENTATION_ID: &str = "quinn-raw";
const PACKAGE_ID: &str = "org.protocol-lab.components.implementation.quinn-raw";
const ALPN: &[u8] = b"plab-raw-quic";
const DEFAULT_PORT: &str = "5448";
const DEFAULT_ECHO_MAX: usize = 64 * 1024;
const MAX_READ_BYTES: usize = 64 * 1024 * 1024;
const DOWNLOAD_MAGIC: &[u8] = b"PLAB-DL1";
const SUPPORTED_SCENARIOS: &[&str] = &[
    "quic.transport.handshake-cold",
    "quic.transport.latency.echo-1kb",
    "quic.transport.stream-throughput.1mb",
    "quic.transport.multiplex.100x64kb",
    "quic.transport.duplex-streams",
];

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct Metadata<'a> {
    status: &'static str,
    implementation_id: &'static str,
    package_id: &'static str,
    protocol: &'static str,
    alpn: &'static str,
    listen: String,
    advertise_host: &'a str,
    quinn_version: &'static str,
    process_id: u32,
    supported_scenarios: &'static [&'static str],
}

#[tokio::main]
async fn main() -> Result<()> {
    let listen = listen_address()?;
    let advertise_host = env::var("PROTOCOL_LAB_TARGET_ADVERTISE_HOST").unwrap_or_default();
    let scenario = env::var("PLAB_SCENARIO_ID").unwrap_or_default();
    let echo_max = echo_max_for_scenario(&scenario);
    let endpoint = Endpoint::server(server_config()?, listen).context("start Quinn endpoint")?;
    let actual = endpoint.local_addr().context("read Quinn listen address")?;

    println!(
        "{}",
        serde_json::to_string(&Metadata {
            status: "ready",
            implementation_id: IMPLEMENTATION_ID,
            package_id: PACKAGE_ID,
            protocol: "quic",
            alpn: "plab-raw-quic",
            listen: actual.to_string(),
            advertise_host: advertise_host.trim(),
            quinn_version: "0.11.11",
            process_id: std::process::id(),
            supported_scenarios: SUPPORTED_SCENARIOS,
        })?
    );
    eprintln!("Quinn raw QUIC target listening on {actual}");

    loop {
        tokio::select! {
            incoming = endpoint.accept() => {
                match incoming {
                    Some(incoming) => {
                        tokio::spawn(async move {
                            if let Err(error) = handle_connection(incoming, echo_max).await {
                                eprintln!("connection failed: {error:#}");
                            }
                        });
                    }
                    None => break,
                }
            }
            signal = tokio::signal::ctrl_c() => {
                signal.context("wait for shutdown signal")?;
                break;
            }
        }
    }

    endpoint.close(VarInt::from_u32(0), b"shutdown");
    endpoint.wait_idle().await;
    Ok(())
}

fn listen_address() -> Result<SocketAddr> {
    let port = env::var("PLAB_QUIC_PORT").unwrap_or_else(|_| DEFAULT_PORT.to_owned());
    let bind = env::var("PROTOCOL_LAB_TARGET_BIND_ADDRESS").unwrap_or_else(|_| "127.0.0.1".to_owned());
    if let Ok(address) = bind.parse::<SocketAddr>() {
        return Ok(address);
    }

    format!("{bind}:{port}")
        .parse()
        .with_context(|| format!("invalid QUIC listen address {bind}:{port}"))
}

fn server_config() -> Result<quinn::ServerConfig> {
    let CertifiedKey { cert, signing_key } =
        generate_simple_self_signed(vec!["localhost".to_owned(), "127.0.0.1".to_owned()])
            .context("generate self-signed certificate")?;
    let certificate = CertificateDer::from(cert.der().to_vec());
    let key = PrivateKeyDer::Pkcs8(PrivatePkcs8KeyDer::from(signing_key.serialize_der()));
    let mut crypto = quinn::rustls::ServerConfig::builder()
        .with_no_client_auth()
        .with_single_cert(vec![certificate], key)
        .context("build Quinn TLS configuration")?;
    crypto.alpn_protocols = vec![ALPN.to_vec()];

    let mut config = quinn::ServerConfig::with_crypto(Arc::new(
        QuicServerConfig::try_from(crypto).context("convert Quinn TLS configuration")?,
    ));
    let mut transport = TransportConfig::default();
    transport.max_concurrent_bidi_streams(VarInt::from_u32(1024));
    transport.max_concurrent_uni_streams(VarInt::from_u32(16));
    transport.stream_receive_window(VarInt::from_u32(64 * 1024 * 1024));
    transport.receive_window(VarInt::from_u32(128 * 1024 * 1024));
    config.transport = Arc::new(transport);
    Ok(config)
}

async fn handle_connection(incoming: Incoming, echo_max: usize) -> Result<()> {
    let connection = incoming.await.context("complete QUIC handshake")?;
    loop {
        match connection.accept_bi().await {
            Ok((send, receive)) => {
                tokio::spawn(async move {
                    if let Err(error) = handle_stream(send, receive, echo_max).await {
                        eprintln!("stream failed: {error:#}");
                    }
                });
            }
            Err(quinn::ConnectionError::ApplicationClosed(_))
            | Err(quinn::ConnectionError::LocallyClosed) => return Ok(()),
            Err(error) => return Err(error).context("accept bidirectional stream"),
        }
    }
}

async fn handle_stream(
    mut send: quinn::SendStream,
    mut receive: quinn::RecvStream,
    echo_max: usize,
) -> Result<()> {
    let payload = receive
        .read_to_end(MAX_READ_BYTES)
        .await
        .context("read stream payload")?;
    if let Some(length) = parse_download_request(&payload) {
        write_deterministic_payload(&mut send, length).await?;
    } else if payload.len() <= echo_max && !payload.is_empty() {
        send.write_all(&payload).await.context("write echo payload")?;
    }
    send.finish().context("finish response stream")?;
    Ok(())
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

async fn write_deterministic_payload(send: &mut quinn::SendStream, length: usize) -> Result<()> {
    let mut offset = 0usize;
    let mut buffer = vec![0u8; length.min(64 * 1024)];
    while offset < length {
        let chunk = buffer.len().min(length - offset);
        for (index, value) in buffer[..chunk].iter_mut().enumerate() {
            *value = ((offset + index) % 251) as u8;
        }
        send.write_all(&buffer[..chunk])
            .await
            .context("write deterministic download payload")?;
        offset += chunk;
    }
    Ok(())
}

fn echo_max_for_scenario(scenario: &str) -> usize {
    match scenario {
        "quic.transport.handshake-cold" | "quic.transport.stream-throughput.1mb" => 0,
        "quic.transport.latency.echo-1kb" => 1024,
        "quic.transport.multiplex.100x64kb" | "quic.transport.duplex-streams" => 64 * 1024,
        _ => DEFAULT_ECHO_MAX,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_exact_download_prelude() {
        let mut request = DOWNLOAD_MAGIC.to_vec();
        request.extend_from_slice(&(1024u64 * 1024).to_be_bytes());
        assert_eq!(parse_download_request(&request), Some(1024 * 1024));
        request.push(0);
        assert_eq!(parse_download_request(&request), None);
    }

    #[test]
    fn scenario_echo_limits_match_declared_wire_behavior() {
        assert_eq!(echo_max_for_scenario("quic.transport.handshake-cold"), 0);
        assert_eq!(echo_max_for_scenario("quic.transport.latency.echo-1kb"), 1024);
        assert_eq!(echo_max_for_scenario("quic.transport.stream-throughput.1mb"), 0);
        assert_eq!(echo_max_for_scenario("quic.transport.multiplex.100x64kb"), 64 * 1024);
        assert_eq!(echo_max_for_scenario("quic.transport.duplex-streams"), 64 * 1024);
    }
}
