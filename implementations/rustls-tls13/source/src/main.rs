use rustls::pki_types::{CertificateDer, PrivateKeyDer};
use rustls::{ProtocolVersion, ServerConfig, ServerConnection};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::env;
use std::fs::File;
use std::io::{self, BufReader, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

const IMPLEMENTATION_ID: &str = "rustls-tls13";
const IMPLEMENTATION_VERSION: &str = "0.1.0";
const SCENARIO_ID: &str = "tls.handshake.full";
const ALPN: &[u8] = b"protocol-lab-tls";
const LEAF_DER_HASH: &str = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e";

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
    session_tickets_enabled: bool,
}

fn main() {
    if let Err((code, message)) = run() {
        eprintln!("{message}");
        std::process::exit(code);
    }
}

fn run() -> Result<(), (i32, String)> {
    let requested = env::var("PLAB_SCENARIO_ID").unwrap_or_else(|_| SCENARIO_ID.to_string());
    if requested.trim() != SCENARIO_ID {
        let kind = if requested.trim().starts_with("tls.") {
            "unsupported"
        } else {
            "unknown"
        };
        return Err((
            if kind == "unsupported" { 3 } else { 2 },
            format!("{kind}:{}", requested.trim()),
        ));
    }

    let listen_address = configured_listen_address();
    let config = server_config(
        &material_path("PLAB_TLS_CERT_FILE", "certs/leaf.pem"),
        &material_path("PLAB_TLS_KEY_FILE", "certs/leaf-key.pem"),
    )
    .map_err(|error| (1, error.to_string()))?;
    let listener = TcpListener::bind(&listen_address).map_err(|error| (1, error.to_string()))?;
    println!(
        "{}",
        serde_json::to_string(&Ready {
            event_name: "ready",
            implementation_id: IMPLEMENTATION_ID,
            implementation_version: IMPLEMENTATION_VERSION,
            scenario_id: SCENARIO_ID,
            listen_address: &listen_address,
            tls_version: "TLS1.3",
            cipher_suite: "TLS_AES_128_GCM_SHA256",
            key_exchange_group: "X25519",
            alpn: "protocol-lab-tls",
            certificate_der_sha256: LEAF_DER_HASH,
            session_tickets_enabled: false,
        })
        .map_err(|error| (1, error.to_string()))?
    );
    io::stdout()
        .flush()
        .map_err(|error| (1, error.to_string()))?;

    for accepted in listener.incoming() {
        match accepted {
            Ok(stream) => {
                let config = config.clone();
                std::thread::spawn(move || {
                    if let Err(error) = handle(stream, config) {
                        eprintln!("connection failed closed: {error}");
                    }
                });
            }
            Err(error) => return Err((1, error.to_string())),
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
    config.send_tls13_tickets = 0;
    config.max_early_data_size = 0;
    Ok(Arc::new(config))
}

fn handle(mut tcp: TcpStream, config: Arc<ServerConfig>) -> Result<(), Box<dyn std::error::Error>> {
    tcp.set_read_timeout(Some(Duration::from_secs(15)))?;
    tcp.set_write_timeout(Some(Duration::from_secs(15)))?;
    let mut tls = ServerConnection::new(config)?;
    while tls.is_handshaking() {
        if tls.wants_read() {
            if tls.read_tls(&mut tcp)? == 0 {
                return Err("peer closed before TLS handshake completion".into());
            }
            tls.process_new_packets()?;
        }
        flush_tls(&mut tcp, &mut tls)?;
    }
    if tls.protocol_version() != Some(ProtocolVersion::TLSv1_3)
        || tls.negotiated_cipher_suite().map(|suite| suite.suite())
            != Some(rustls::CipherSuite::TLS13_AES_128_GCM_SHA256)
        || tls
            .negotiated_key_exchange_group()
            .map(|group| group.name())
            != Some(rustls::NamedGroup::X25519)
        || tls.alpn_protocol() != Some(ALPN)
        || tls.server_name() != Some("tls.plab.test")
    {
        return Err("TLS version, cipher, key exchange, ALPN, or SNI mismatch".into());
    }
    let mut byte = [0u8; 1];
    match tls.reader().read(&mut byte) {
        Ok(0) => {}
        Ok(_) => {
            return Err("application data is not admitted by the full-handshake profile".into());
        }
        Err(error) if error.kind() == io::ErrorKind::WouldBlock => {}
        Err(error) => return Err(error.into()),
    }
    tls.send_close_notify();
    flush_tls(&mut tcp, &mut tls)?;
    Ok(())
}

fn flush_tls(tcp: &mut TcpStream, tls: &mut ServerConnection) -> io::Result<()> {
    while tls.wants_write() {
        tls.write_tls(tcp)?;
    }
    tcp.flush()
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
        .unwrap_or_else(|| "127.0.0.1:18447".to_string())
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
    #[test]
    fn exact_profile_provider_has_one_cipher_and_group() {
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
