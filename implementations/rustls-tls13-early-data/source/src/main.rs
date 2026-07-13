use rustls::pki_types::{CertificateDer, PrivateKeyDer};
use rustls::server::ServerSessionMemoryCache;
use rustls::{HandshakeKind, ProtocolVersion, ServerConfig, ServerConnection};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::env;
use std::fs::File;
use std::io::{self, BufReader, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

const IMPLEMENTATION_ID: &str = "rustls-tls13-early-data";
const IMPLEMENTATION_VERSION: &str = "0.1.0";
const ACCEPTED_ID: &str = "tls.early-data.accepted";
const REJECTED_ID: &str = "tls.early-data.rejected";
const ALPN: &[u8] = b"protocol-lab-tls";
const PAYLOAD_SIZE: usize = 1024;
const REJECTION_CIPHERTEXT_LIMIT: u32 = 4096;
const PAYLOAD_HASH: &str = "e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf";
const LEAF_DER_HASH: &str = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e";
const RESUMPTION_MARKER: &[u8] = b"protocol-lab-tls13-early-data-v1";

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Outcome {
    Accepted,
    Rejected,
}

impl Outcome {
    fn from_id(value: &str) -> Result<Self, String> {
        match value {
            ACCEPTED_ID => Ok(Self::Accepted),
            REJECTED_ID => Ok(Self::Rejected),
            "tls.handshake.full"
            | "tls.handshake.resumed"
            | "tls.handshake.full.tls12"
            | "tls.handshake.full.chacha20"
            | "tls.handshake.mutual-auth"
            | "tls.key-update.diagnostic"
            | "tls.record.coverage"
            | "tls.record.throughput" => Err(format!("unsupported:{value}")),
            _ => Err(format!("unknown:{value}")),
        }
    }

    fn scenario_id(self) -> &'static str {
        match self {
            Self::Accepted => ACCEPTED_ID,
            Self::Rejected => REJECTED_ID,
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct Ready<'a> {
    event_name: &'a str,
    implementation_id: &'a str,
    implementation_version: &'a str,
    scenario_id: &'a str,
    listen_address: &'a str,
    tls_version: &'a str,
    cipher_suite: &'a str,
    key_exchange_group: &'a str,
    alpn: &'a str,
    certificate_der_sha256: &'a str,
    crypto_provider: &'a str,
    max_early_data_bytes: usize,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct TargetProof<'a> {
    event_name: &'a str,
    implementation_id: &'a str,
    implementation_version: &'a str,
    scenario_id: &'a str,
    connection_role: &'a str,
    tls_version: &'a str,
    cipher_suite: &'a str,
    key_exchange_group: &'a str,
    alpn: &'a str,
    did_resume: bool,
    early_data_outcome: &'a str,
    early_data_bytes_delivered: usize,
    post_handshake_retry_bytes_delivered: usize,
    application_effect_count: usize,
    payload_sha256: &'a str,
    zero_duplicate_effects: bool,
}

fn main() {
    if let Err((code, message)) = run() {
        eprintln!("{message}");
        std::process::exit(code);
    }
}

fn run() -> Result<(), (i32, String)> {
    let requested = env::var("PLAB_SCENARIO_ID").unwrap_or_default();
    let outcome = Outcome::from_id(requested.trim()).map_err(|message| {
        let code = if message.starts_with("unsupported:") {
            3
        } else {
            2
        };
        (code, message)
    })?;
    let listen_address = configured_listen_address();
    let cert_path = material_path("PLAB_TLS_CERT_FILE", "certs/leaf.pem");
    let key_path = material_path("PLAB_TLS_KEY_FILE", "certs/leaf-key.pem");
    let config = server_config(&cert_path, &key_path).map_err(|e| (1, e.to_string()))?;
    let listener = TcpListener::bind(&listen_address).map_err(|e| (1, e.to_string()))?;
    println!(
        "{}",
        serde_json::to_string(&Ready {
            event_name: "ready",
            implementation_id: IMPLEMENTATION_ID,
            implementation_version: IMPLEMENTATION_VERSION,
            scenario_id: outcome.scenario_id(),
            listen_address: &listen_address,
            tls_version: "TLS1.3",
            cipher_suite: "TLS_AES_128_GCM_SHA256",
            key_exchange_group: "X25519",
            alpn: "protocol-lab-tls",
            certificate_der_sha256: LEAF_DER_HASH,
            crypto_provider: "rustls-rustcrypto@0.0.2-alpha",
            max_early_data_bytes: REJECTION_CIPHERTEXT_LIMIT as usize,
        })
        .map_err(|e| (1, e.to_string()))?
    );
    io::stdout().flush().map_err(|e| (1, e.to_string()))?;

    let mut next_is_source = true;
    for accepted in listener.incoming() {
        let mut tcp = accepted.map_err(|e| (1, e.to_string()))?;
        tcp.set_read_timeout(Some(Duration::from_secs(15)))
            .map_err(|e| (1, e.to_string()))?;
        tcp.set_write_timeout(Some(Duration::from_secs(15)))
            .map_err(|e| (1, e.to_string()))?;
        let role = if next_is_source { "source" } else { "measured" };
        let result = if next_is_source {
            handle_source(&mut tcp, config.clone(), outcome)
        } else {
            handle_measured(&mut tcp, config.clone(), outcome)
        };
        next_is_source = !next_is_source;
        if let Err(error) = result {
            eprintln!("{role} connection failed closed: {error}");
        }
    }
    Ok(())
}

fn server_config(
    cert_path: &Path,
    key_path: &Path,
) -> Result<Arc<ServerConfig>, Box<dyn std::error::Error>> {
    let certificates = load_certificates(cert_path)?;
    if certificates.len() != 1 || sha256(certificates[0].as_ref()) != LEAF_DER_HASH {
        return Err("server certificate substitution or chain expansion detected".into());
    }
    let key = load_private_key(key_path)?;
    let mut provider = rustls_rustcrypto::provider();
    provider
        .cipher_suites
        .retain(|suite| suite.suite() == rustls::CipherSuite::TLS13_AES_128_GCM_SHA256);
    provider
        .kx_groups
        .retain(|group| group.name() == rustls::NamedGroup::X25519);
    let mut config = ServerConfig::builder_with_provider(Arc::new(provider))
        .with_protocol_versions(&[&rustls::version::TLS13])?
        .with_no_client_auth()
        .with_single_cert(certificates, key)?;
    config.alpn_protocols = vec![ALPN.to_vec()];
    // rustls applies this bound to plaintext on acceptance and ciphertext on rejection.
    // The contract payload remains exactly 1024 bytes; the slop permits authenticated
    // discard of the record framing when the explicit rejection lane is selected.
    config.max_early_data_size = REJECTION_CIPHERTEXT_LIMIT;
    config.send_tls13_tickets = 1;
    config.session_storage = ServerSessionMemoryCache::new(64);
    Ok(Arc::new(config))
}

fn handle_source(
    tcp: &mut TcpStream,
    config: Arc<ServerConfig>,
    outcome: Outcome,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut tls = ServerConnection::new(config)?;
    tls.set_resumption_data(RESUMPTION_MARKER);
    drive_handshake(tcp, &mut tls, None, false)?;
    validate_negotiation(&tls, false)?;
    if tls.early_data().is_some() {
        return Err("source connection unexpectedly carried early data".into());
    }
    tls.writer().write_all(b"S")?;
    flush_tls(tcp, &mut tls)?;
    tls.send_close_notify();
    flush_tls(tcp, &mut tls)?;
    println!(
        "{}",
        serde_json::to_string(&TargetProof {
            event_name: "target-proof",
            implementation_id: IMPLEMENTATION_ID,
            implementation_version: IMPLEMENTATION_VERSION,
            scenario_id: outcome.scenario_id(),
            connection_role: "source",
            tls_version: "TLS1.3",
            cipher_suite: "TLS_AES_128_GCM_SHA256",
            key_exchange_group: "X25519",
            alpn: "protocol-lab-tls",
            did_resume: false,
            early_data_outcome: "not-attempted",
            early_data_bytes_delivered: 0,
            post_handshake_retry_bytes_delivered: 0,
            application_effect_count: 0,
            payload_sha256: PAYLOAD_HASH,
            zero_duplicate_effects: true,
        })?
    );
    io::stdout().flush()?;
    Ok(())
}

fn handle_measured(
    tcp: &mut TcpStream,
    config: Arc<ServerConfig>,
    outcome: Outcome,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut tls = ServerConnection::new(config)?;
    tls.set_resumption_data(RESUMPTION_MARKER);
    if outcome == Outcome::Rejected {
        tls.reject_early_data();
    }
    let mut early = Vec::new();
    drive_handshake(tcp, &mut tls, Some(&mut early), false)?;
    validate_negotiation(&tls, true)?;
    if tls.received_resumption_data() != Some(RESUMPTION_MARKER) {
        return Err("single-use source-session provenance marker was absent".into());
    }

    let mut retry = Vec::new();
    match outcome {
        Outcome::Accepted => {
            validate_payload(&early)?;
            ensure_clean_close_without_application_data(tcp, &mut tls)?;
        }
        Outcome::Rejected => {
            if !early.is_empty() {
                return Err("rejected early data reached the application".into());
            }
            retry.resize(PAYLOAD_SIZE, 0);
            read_exact_application_data(tcp, &mut tls, &mut retry)?;
            validate_payload(&retry)?;
            ensure_clean_close_without_application_data(tcp, &mut tls)?;
        }
    }
    tls.send_close_notify();
    flush_tls(tcp, &mut tls)?;
    let (early_count, retry_count, label) = match outcome {
        Outcome::Accepted => (early.len(), 0, "accepted"),
        Outcome::Rejected => (0, retry.len(), "rejected"),
    };
    println!(
        "{}",
        serde_json::to_string(&TargetProof {
            event_name: "target-proof",
            implementation_id: IMPLEMENTATION_ID,
            implementation_version: IMPLEMENTATION_VERSION,
            scenario_id: outcome.scenario_id(),
            connection_role: "measured",
            tls_version: "TLS1.3",
            cipher_suite: "TLS_AES_128_GCM_SHA256",
            key_exchange_group: "X25519",
            alpn: "protocol-lab-tls",
            did_resume: true,
            early_data_outcome: label,
            early_data_bytes_delivered: early_count,
            post_handshake_retry_bytes_delivered: retry_count,
            application_effect_count: 1,
            payload_sha256: PAYLOAD_HASH,
            zero_duplicate_effects: early_count + retry_count == PAYLOAD_SIZE,
        })?
    );
    io::stdout().flush()?;
    Ok(())
}

fn drive_handshake(
    tcp: &mut TcpStream,
    tls: &mut ServerConnection,
    mut early: Option<&mut Vec<u8>>,
    reject_after_client_hello: bool,
) -> Result<(), Box<dyn std::error::Error>> {
    let _ = reject_after_client_hello;
    while tls.is_handshaking() {
        if tls.wants_read() {
            let count = tls.read_tls(tcp)?;
            if count == 0 {
                return Err("peer closed before TLS handshake completion".into());
            }
            tls.process_new_packets()?;
            if let Some(buffer) = early.as_deref_mut()
                && let Some(mut reader) = tls.early_data()
            {
                reader.read_to_end(buffer)?;
            }
        }
        flush_tls(tcp, tls)?;
    }
    Ok(())
}

fn validate_negotiation(
    tls: &ServerConnection,
    resumed: bool,
) -> Result<(), Box<dyn std::error::Error>> {
    if tls.protocol_version() != Some(ProtocolVersion::TLSv1_3)
        || tls.negotiated_cipher_suite().map(|suite| suite.suite())
            != Some(rustls::CipherSuite::TLS13_AES_128_GCM_SHA256)
        || tls
            .negotiated_key_exchange_group()
            .map(|group| group.name())
            != Some(rustls::NamedGroup::X25519)
        || tls.alpn_protocol() != Some(ALPN)
        || (tls.handshake_kind() == Some(HandshakeKind::Resumed)) != resumed
        || tls.server_name() != Some("tls.plab.test")
    {
        return Err(
            "TLS version, cipher, key exchange, ALPN, SNI, or session-state mismatch".into(),
        );
    }
    Ok(())
}

fn read_exact_application_data(
    tcp: &mut TcpStream,
    tls: &mut ServerConnection,
    output: &mut [u8],
) -> Result<(), Box<dyn std::error::Error>> {
    let mut offset = 0;
    while offset < output.len() {
        match tls.reader().read(&mut output[offset..]) {
            Ok(0) => {}
            Ok(count) => {
                offset += count;
                continue;
            }
            Err(error) if error.kind() == io::ErrorKind::WouldBlock => {}
            Err(error) => return Err(error.into()),
        }
        let count = tls.read_tls(tcp)?;
        if count == 0 {
            return Err("peer closed before the post-handshake retry completed".into());
        }
        tls.process_new_packets()?;
        flush_tls(tcp, tls)?;
    }
    Ok(())
}

fn ensure_clean_close_without_application_data(
    tcp: &mut TcpStream,
    tls: &mut ServerConnection,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut extra = [0u8; 1];
    loop {
        match tls.reader().read(&mut extra) {
            Ok(0) => {}
            Ok(_) => {
                return Err(
                    "duplicate application effect or trailing application data detected".into(),
                );
            }
            Err(error) if error.kind() == io::ErrorKind::WouldBlock => {}
            Err(error) => return Err(error.into()),
        }
        let count = tls.read_tls(tcp)?;
        if count == 0 {
            return Ok(());
        }
        let state = tls.process_new_packets()?;
        flush_tls(tcp, tls)?;
        if state.peer_has_closed() {
            return Ok(());
        }
        if !tls.is_handshaking() && !tls.wants_read() {
            return Ok(());
        }
    }
}

fn flush_tls(tcp: &mut TcpStream, tls: &mut ServerConnection) -> io::Result<()> {
    while tls.wants_write() {
        tls.write_tls(tcp)?;
    }
    tcp.flush()
}

fn validate_payload(payload: &[u8]) -> Result<(), Box<dyn std::error::Error>> {
    if payload.len() != PAYLOAD_SIZE
        || payload.iter().any(|value| *value != 0x5a)
        || sha256(payload) != PAYLOAD_HASH
    {
        return Err("deterministic 1024-byte 0x5A payload identity mismatch".into());
    }
    Ok(())
}

fn load_certificates(
    path: &Path,
) -> Result<Vec<CertificateDer<'static>>, Box<dyn std::error::Error>> {
    Ok(
        rustls_pemfile::certs(&mut BufReader::new(File::open(path)?))
            .collect::<Result<Vec<_>, _>>()?,
    )
}

fn load_private_key(path: &Path) -> Result<PrivateKeyDer<'static>, Box<dyn std::error::Error>> {
    rustls_pemfile::private_key(&mut BufReader::new(File::open(path)?))?
        .ok_or_else(|| "private key PEM contained no supported key".into())
}

fn configured_listen_address() -> String {
    env::var("PLAB_LISTEN_ADDRESS")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .or_else(|| {
            env::var("PLAB_TARGET_PORT")
                .ok()
                .map(|port| format!("127.0.0.1:{port}"))
        })
        .unwrap_or_else(|| "127.0.0.1:18446".to_string())
}

fn material_path(variable: &str, relative: &str) -> PathBuf {
    if let Ok(explicit) = env::var(variable)
        && !explicit.trim().is_empty()
    {
        return PathBuf::from(explicit);
    }
    if let Ok(executable) = env::current_exe()
        && let Some(bin) = executable.parent()
    {
        let candidate = bin.join("..").join("..").join(relative);
        if candidate.exists() {
            return candidate;
        }
    }
    PathBuf::from(relative)
}

fn sha256(value: &[u8]) -> String {
    format!("{:x}", Sha256::digest(value))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn exact_scenario_routing_is_fail_closed() {
        assert_eq!(Outcome::from_id(ACCEPTED_ID), Ok(Outcome::Accepted));
        assert_eq!(Outcome::from_id(REJECTED_ID), Ok(Outcome::Rejected));
        assert!(
            Outcome::from_id("tls.handshake.full")
                .unwrap_err()
                .starts_with("unsupported:")
        );
        assert!(
            Outcome::from_id("tls.early-data.other")
                .unwrap_err()
                .starts_with("unknown:")
        );
    }

    #[test]
    fn canonical_payload_identity_is_exact() {
        let payload = vec![0x5a; PAYLOAD_SIZE];
        assert_eq!(sha256(&payload), PAYLOAD_HASH);
        assert!(validate_payload(&payload).is_ok());
        assert!(validate_payload(&payload[..1023]).is_err());
    }

    #[test]
    fn provider_is_narrowed_to_exact_profile() {
        let mut provider = rustls_rustcrypto::provider();
        provider
            .cipher_suites
            .retain(|suite| suite.suite() == rustls::CipherSuite::TLS13_AES_128_GCM_SHA256);
        provider
            .kx_groups
            .retain(|group| group.name() == rustls::NamedGroup::X25519);
        assert_eq!(provider.cipher_suites.len(), 1);
        assert_eq!(provider.kx_groups.len(), 1);
    }
}
