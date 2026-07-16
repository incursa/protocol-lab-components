use std::{env, net::SocketAddr};

use anyhow::{bail, Context, Result};
use bytes::Bytes;
use rcgen::{generate_simple_self_signed, CertifiedKey};
use s2n_quic::{provider::limits::Limits, provider::tls::rustls, Server};
use serde::Serialize;

const IMPLEMENTATION_ID: &str = "s2n-quic-raw";
const PACKAGE_ID: &str = "org.protocol-lab.components.implementation.s2n-quic-raw";
const ALPN: &[u8] = b"plab-raw-quic";
const DEFAULT_PORT: &str = "5449";
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
    s2n_quic_version: &'static str,
    tls_provider: &'static str,
    process_id: u32,
    supported_scenarios: &'static [&'static str],
}

#[tokio::main]
async fn main() -> Result<()> {
    let listen = listen_address()?;
    let advertise_host = env::var("PROTOCOL_LAB_TARGET_ADVERTISE_HOST").unwrap_or_default();
    let scenario = env::var("PLAB_SCENARIO_ID").unwrap_or_default();
    let echo_max = echo_max_for_scenario(&scenario);
    let tls = tls_provider()?;
    let limits = Limits::new()
        .with_max_open_remote_bidirectional_streams(1024)?
        .with_max_open_local_bidirectional_streams(16)?
        .with_data_window(128 * 1024 * 1024)?
        .with_bidirectional_remote_data_window(64 * 1024 * 1024)?
        .with_bidirectional_local_data_window(64 * 1024 * 1024)?
        .with_max_send_buffer_size(64 * 1024 * 1024)?;
    let mut server = Server::builder()
        .with_tls(tls)
        .context("configure s2n-quic TLS")?
        .with_limits(limits)
        .context("configure s2n-quic limits")?
        .with_io(listen)
        .context("configure s2n-quic UDP endpoint")?
        .start()
        .context("start s2n-quic endpoint")?;
    let actual = server.local_addr().context("read s2n-quic listen address")?;

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
            s2n_quic_version: "1.83.0",
            tls_provider: "rustls",
            process_id: std::process::id(),
            supported_scenarios: SUPPORTED_SCENARIOS,
        })?
    );
    eprintln!("s2n-quic raw QUIC target listening on {actual}");

    loop {
        tokio::select! {
            connection = server.accept() => {
                match connection {
                    Some(mut connection) => {
                        tokio::spawn(async move {
                            while let Ok(Some(stream)) = connection.accept_bidirectional_stream().await {
                                tokio::spawn(async move {
                                    if let Err(error) = handle_stream(stream, echo_max).await {
                                        eprintln!("stream failed: {error:#}");
                                    }
                                });
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

fn tls_provider() -> Result<rustls::Server> {
    let CertifiedKey { cert, signing_key } =
        generate_simple_self_signed(vec!["localhost".to_owned(), "127.0.0.1".to_owned()])
            .context("generate self-signed certificate")?;
    rustls::Server::builder()
        .with_certificate(cert.der().to_vec(), signing_key.serialize_der())
        .map_err(|error| anyhow::anyhow!("configure s2n-quic certificate: {error}"))?
        .with_application_protocols(std::iter::once(ALPN))
        .map_err(|error| anyhow::anyhow!("configure s2n-quic ALPN: {error}"))?
        .build()
        .map_err(|error| anyhow::anyhow!("build s2n-quic rustls provider: {error}"))
}

async fn handle_stream(mut stream: s2n_quic::stream::BidirectionalStream, echo_max: usize) -> Result<()> {
    let mut payload = Vec::new();
    while let Some(chunk) = stream.receive().await.context("receive stream payload")? {
        if payload.len() + chunk.len() > MAX_READ_BYTES {
            bail!("stream payload exceeds {MAX_READ_BYTES} bytes");
        }
        payload.extend_from_slice(&chunk);
    }

    if let Some(length) = parse_download_request(&payload) {
        write_deterministic_payload(&mut stream, length).await?;
    } else if payload.len() <= echo_max && !payload.is_empty() {
        stream
            .send(Bytes::from(payload))
            .await
            .context("write echo payload")?;
    }
    stream.finish().context("finish response stream")?;
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

async fn write_deterministic_payload(
    stream: &mut s2n_quic::stream::BidirectionalStream,
    length: usize,
) -> Result<()> {
    let mut offset = 0usize;
    let mut buffer = vec![0u8; length.min(64 * 1024)];
    while offset < length {
        let chunk = buffer.len().min(length - offset);
        for (index, value) in buffer[..chunk].iter_mut().enumerate() {
            *value = ((offset + index) % 251) as u8;
        }
        stream
            .send(Bytes::copy_from_slice(&buffer[..chunk]))
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
