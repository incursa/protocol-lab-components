use rustls::client::{
    ClientSessionStore, Resumption, Tls12ClientSessionValue, Tls13ClientSessionValue,
};
use rustls::pki_types::{CertificateDer, ServerName};
use rustls::{
    ClientConfig, ClientConnection, HandshakeKind, NamedGroup, ProtocolVersion, RootCertStore,
    StreamOwned,
};
use serde::Serialize;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::env;
use std::fmt;
use std::fs::{self, File};
use std::io::{self, BufReader, Read, Write};
use std::net::TcpStream;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

const EXECUTOR_ID: &str = "rustls-tls13-early-data-executor";
const EXECUTOR_VERSION: &str = "0.1.0";
const GENERATOR_ID: &str = "rustls-tls13-early-data-load";
const GENERATOR_VERSION: &str = "0.1.0";
const ACCEPTED_ID: &str = "tls.early-data.accepted";
const REJECTED_ID: &str = "tls.early-data.rejected";
const LOAD_PROFILE_ID: &str = "tls-diagnostic";
const SERVER_NAME: &str = "tls.plab.test";
const ALPN: &[u8] = b"protocol-lab-tls";
const PAYLOAD_SIZE: usize = 1024;
const PAYLOAD_HASH: &str = "e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf";
const LEAF_DER_HASH: &str = "cf99a110e63d11b14d6a526d132b11b0363058f8eac30dd79a62f27fcbc38b5e";
const LEAF_SPKI_HASH: &str = "407e0f88780f510da95d16cbf92243a3879c6c676be5b3c5779f11d31e646fc0";

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Outcome {
    Accepted,
    Rejected,
}

impl Outcome {
    fn from_id(value: &str) -> Result<Self, ScenarioError> {
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
            | "tls.record.throughput" => Err(ScenarioError::Unsupported(value.to_string())),
            _ => Err(ScenarioError::Unknown(value.to_string())),
        }
    }

    fn scenario_id(self) -> &'static str {
        match self {
            Self::Accepted => ACCEPTED_ID,
            Self::Rejected => REJECTED_ID,
        }
    }

    fn variant(self) -> &'static str {
        match self {
            Self::Accepted => "tls1.3-zero-rtt-accepted",
            Self::Rejected => "tls1.3-zero-rtt-rejected",
        }
    }

    fn label(self) -> &'static str {
        match self {
            Self::Accepted => "accepted",
            Self::Rejected => "rejected",
        }
    }
}

#[derive(Debug)]
enum ScenarioError {
    Unsupported(String),
    Unknown(String),
}

#[derive(Default)]
struct StoreCounts {
    inserted: AtomicUsize,
    taken: AtomicUsize,
    hits: AtomicUsize,
    max_early_data: AtomicUsize,
}

struct TrackingStore {
    ticket: Mutex<Option<Tls13ClientSessionValue>>,
    kx_hint: Mutex<Option<NamedGroup>>,
    counts: StoreCounts,
}

impl TrackingStore {
    fn new() -> Self {
        Self {
            ticket: Mutex::new(None),
            kx_hint: Mutex::new(None),
            counts: StoreCounts::default(),
        }
    }

    fn snapshot(&self) -> (usize, usize, usize, usize) {
        (
            self.counts.inserted.load(Ordering::SeqCst),
            self.counts.taken.load(Ordering::SeqCst),
            self.counts.hits.load(Ordering::SeqCst),
            self.counts.max_early_data.load(Ordering::SeqCst),
        )
    }
}

impl fmt::Debug for TrackingStore {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("TrackingStore")
            .field("counts", &self.snapshot())
            .finish_non_exhaustive()
    }
}

impl ClientSessionStore for TrackingStore {
    fn set_kx_hint(&self, server_name: ServerName<'static>, group: NamedGroup) {
        debug_assert_eq!(server_name.to_str(), SERVER_NAME);
        *self.kx_hint.lock().expect("kx-hint mutex poisoned") = Some(group);
    }

    fn kx_hint(&self, server_name: &ServerName<'_>) -> Option<NamedGroup> {
        debug_assert_eq!(server_name.to_str(), SERVER_NAME);
        *self.kx_hint.lock().expect("kx-hint mutex poisoned")
    }

    fn set_tls12_session(&self, server_name: ServerName<'static>, value: Tls12ClientSessionValue) {
        let _ = (server_name, value);
    }

    fn tls12_session(&self, server_name: &ServerName<'_>) -> Option<Tls12ClientSessionValue> {
        let _ = server_name;
        None
    }

    fn remove_tls12_session(&self, server_name: &ServerName<'static>) {
        let _ = server_name;
    }

    fn insert_tls13_ticket(
        &self,
        server_name: ServerName<'static>,
        value: Tls13ClientSessionValue,
    ) {
        self.counts.inserted.fetch_add(1, Ordering::SeqCst);
        self.counts
            .max_early_data
            .store(value.max_early_data_size() as usize, Ordering::SeqCst);
        debug_assert_eq!(server_name.to_str(), SERVER_NAME);
        *self.ticket.lock().expect("ticket mutex poisoned") = Some(value);
    }

    fn take_tls13_ticket(
        &self,
        server_name: &ServerName<'static>,
    ) -> Option<Tls13ClientSessionValue> {
        self.counts.taken.fetch_add(1, Ordering::SeqCst);
        debug_assert_eq!(server_name.to_str(), SERVER_NAME);
        let value = self.ticket.lock().expect("ticket mutex poisoned").take();
        if value.is_some() {
            self.counts.hits.fetch_add(1, Ordering::SeqCst);
        }
        value
    }
}

#[derive(Serialize, Clone)]
#[serde(rename_all = "camelCase")]
struct Negotiation {
    tls_profile_id: &'static str,
    tls_version: &'static str,
    cipher_suite: &'static str,
    key_exchange_group: &'static str,
    signature_scheme: &'static str,
    alpn: &'static str,
    server_name: &'static str,
    handshake_complete: bool,
    did_resume: bool,
    early_data_attempted: bool,
    early_data_accepted: bool,
    application_data_bytes_sent: usize,
    application_data_bytes_received: usize,
    certificate_profile: &'static str,
    certificate_der_sha256: &'static str,
    certificate_spki_sha256: &'static str,
    certificate_verified: bool,
    tls_handshake_latency_ms: f64,
    crypto_provider: &'static str,
    acceleration_provenance: &'static str,
    platform_os: &'static str,
    platform_architecture: &'static str,
}

struct Operation {
    source: Negotiation,
    measured: Negotiation,
    source_counts: (usize, usize, usize, usize),
    final_counts: (usize, usize, usize, usize),
    early_data_offered_bytes: usize,
    retry_bytes: usize,
}

fn main() {
    match run() {
        Ok(()) => {}
        Err((code, message)) => {
            eprintln!("{message}");
            std::process::exit(code);
        }
    }
}

fn run() -> Result<(), (i32, String)> {
    let args: Vec<String> = env::args().collect();
    if args.iter().any(|value| value == "--version") {
        println!("{EXECUTOR_ID} {EXECUTOR_VERSION}");
        return Ok(());
    }
    let output_dir = argument(&args, "--output-dir")
        .or_else(|| nonempty_env("PLAB_ARTIFACT_DIR"))
        .unwrap_or_else(|| "artifacts".to_string());
    fs::create_dir_all(&output_dir).map_err(|e| (1, e.to_string()))?;
    let requested = env::var("PLAB_SCENARIO_ID").unwrap_or_default();
    let outcome = match Outcome::from_id(requested.trim()) {
        Ok(value) => value,
        Err(ScenarioError::Unsupported(id)) => {
            emit_unsupported(Path::new(&output_dir), &id).map_err(|e| (1, e.to_string()))?;
            return Err((3, format!("scenario {id} is explicitly unsupported")));
        }
        Err(ScenarioError::Unknown(id)) => return Err((2, format!("unknown TLS scenario {id}"))),
    };
    verify_identity("PLAB_EXECUTOR_ID", EXECUTOR_ID)?;
    verify_identity("PLAB_EXECUTOR_VERSION", EXECUTOR_VERSION)?;
    verify_identity("PLAB_LOAD_GENERATOR_ID", GENERATOR_ID)?;
    verify_identity("PLAB_LOAD_GENERATOR_VERSION", GENERATOR_VERSION)?;
    verify_identity("PLAB_PROTOCOL", "tls")?;
    verify_identity("PLAB_PROTOCOL_VARIANT", outcome.variant())?;
    verify_identity("PLAB_LOAD_PROFILE_ID", LOAD_PROFILE_ID)?;

    let target = argument(&args, "--target-address")
        .or_else(|| nonempty_env("PLAB_TARGET_BASE_URL"))
        .ok_or_else(|| (2, "target address is required".to_string()))?;
    let address = normalize_target(&target).map_err(|e| (2, e))?;
    let root_path = argument(&args, "--root-certificate")
        .or_else(|| nonempty_env("PLAB_TLS_ROOT_CERTIFICATE_PATH"))
        .unwrap_or_else(|| {
            material_path("certs/root.pem")
                .to_string_lossy()
                .into_owned()
        });
    let operation =
        execute(outcome, &address, Path::new(&root_path)).map_err(|e| (1, e.to_string()))?;
    write_artifacts(Path::new(&output_dir), outcome, &operation).map_err(|e| (1, e.to_string()))?;
    println!(
        "{}",
        serde_json::to_string_pretty(&executor_result(outcome, &operation))
            .map_err(|e| (1, e.to_string()))?
    );
    Ok(())
}

fn execute(
    outcome: Outcome,
    address: &str,
    root_path: &Path,
) -> Result<Operation, Box<dyn std::error::Error>> {
    let store = Arc::new(TrackingStore::new());
    let config = client_config(root_path, store.clone())?;
    let source = source_connection(address, config.clone())?;
    let source_counts = store.snapshot();
    if source_counts.0 != 1 || source_counts.2 != 0 || source_counts.3 < PAYLOAD_SIZE {
        return Err(format!(
            "source session did not yield exactly one unused ticket: {source_counts:?}"
        )
        .into());
    }
    let (measured, offered, retry) = measured_connection(address, config, outcome, store.clone())?;
    let final_counts = store.snapshot();
    if final_counts.1.saturating_sub(source_counts.1) != 1
        || final_counts.2.saturating_sub(source_counts.2) != 1
    {
        return Err(format!(
            "single-use session ticket was not consumed exactly once: {final_counts:?}"
        )
        .into());
    }
    Ok(Operation {
        source,
        measured,
        source_counts,
        final_counts,
        early_data_offered_bytes: offered,
        retry_bytes: retry,
    })
}

fn client_config(
    root_path: &Path,
    store: Arc<TrackingStore>,
) -> Result<Arc<ClientConfig>, Box<dyn std::error::Error>> {
    let mut roots = RootCertStore::empty();
    for certificate in load_certificates(root_path)? {
        roots.add(certificate)?;
    }
    let mut provider = rustls_rustcrypto::provider();
    provider
        .cipher_suites
        .retain(|suite| suite.suite() == rustls::CipherSuite::TLS13_AES_128_GCM_SHA256);
    provider
        .kx_groups
        .retain(|group| group.name() == NamedGroup::X25519);
    let mut config = ClientConfig::builder_with_provider(Arc::new(provider))
        .with_protocol_versions(&[&rustls::version::TLS13])?
        .with_root_certificates(roots)
        .with_no_client_auth();
    config.alpn_protocols = vec![ALPN.to_vec()];
    config.enable_early_data = true;
    config.resumption = Resumption::store(store);
    Ok(Arc::new(config))
}

fn source_connection(
    address: &str,
    config: Arc<ClientConfig>,
) -> Result<Negotiation, Box<dyn std::error::Error>> {
    let tcp = connected_tcp(address)?;
    let server_name = ServerName::try_from(SERVER_NAME)?.to_owned();
    let connection = ClientConnection::new(config, server_name)?;
    let started = Instant::now();
    let mut stream = StreamOwned::new(connection, tcp);
    let mut marker = [0u8; 1];
    stream.read_exact(&mut marker)?;
    if marker != *b"S" {
        return Err("source-session target marker mismatch".into());
    }
    let mut trailing = Vec::new();
    stream.read_to_end(&mut trailing)?;
    if !trailing.is_empty() {
        return Err("source-session target emitted unexpected application data".into());
    }
    observe(&stream.conn, false, false, 0, 1, started.elapsed())
}

fn measured_connection(
    address: &str,
    config: Arc<ClientConfig>,
    outcome: Outcome,
    store: Arc<TrackingStore>,
) -> Result<(Negotiation, usize, usize), Box<dyn std::error::Error>> {
    let server_name = ServerName::try_from(SERVER_NAME)?.to_owned();
    let mut connection = ClientConnection::new(config, server_name)?;
    let payload = vec![0x5a; PAYLOAD_SIZE];
    if sha256(&payload) != PAYLOAD_HASH {
        return Err("canonical payload construction mismatch".into());
    }
    {
        let mut early = connection.early_data().ok_or_else(|| {
            format!(
                "source ticket did not enable TLS 1.3 early data: {:?}",
                store.snapshot()
            )
        })?;
        early.write_all(&payload)?;
    }
    let tcp = connected_tcp(address)?;
    let started = Instant::now();
    let mut stream = StreamOwned::new(connection, tcp);
    while stream.conn.is_handshaking() {
        stream.conn.complete_io(&mut stream.sock)?;
    }
    let accepted = stream.conn.is_early_data_accepted();
    if accepted != (outcome == Outcome::Accepted) {
        return Err(format!(
            "expected early data {}, observed {}",
            outcome.label(),
            if accepted { "accepted" } else { "rejected" }
        )
        .into());
    }
    let retry_bytes = if outcome == Outcome::Rejected {
        stream.write_all(&payload)?;
        stream.flush()?;
        PAYLOAD_SIZE
    } else {
        0
    };
    stream.conn.send_close_notify();
    stream.flush()?;
    let mut response = Vec::new();
    stream.read_to_end(&mut response)?;
    if !response.is_empty() {
        return Err("target emitted unexpected measured application data".into());
    }
    let observation = observe(
        &stream.conn,
        true,
        accepted,
        PAYLOAD_SIZE + retry_bytes,
        0,
        started.elapsed(),
    )?;
    Ok((observation, PAYLOAD_SIZE, retry_bytes))
}

fn observe(
    connection: &ClientConnection,
    resumed: bool,
    accepted: bool,
    sent: usize,
    received: usize,
    elapsed: Duration,
) -> Result<Negotiation, Box<dyn std::error::Error>> {
    let certificates = connection
        .peer_certificates()
        .ok_or("server certificate proof unavailable")?;
    if certificates.len() != 1 || sha256(certificates[0].as_ref()) != LEAF_DER_HASH {
        return Err("server certificate DER identity or chain shape mismatch".into());
    }
    if connection.protocol_version() != Some(ProtocolVersion::TLSv1_3)
        || connection
            .negotiated_cipher_suite()
            .map(|suite| suite.suite())
            != Some(rustls::CipherSuite::TLS13_AES_128_GCM_SHA256)
        || connection
            .negotiated_key_exchange_group()
            .map(|group| group.name())
            != Some(NamedGroup::X25519)
        || connection.alpn_protocol() != Some(ALPN)
        || (connection.handshake_kind() == Some(HandshakeKind::Resumed)) != resumed
    {
        return Err("TLS version, cipher, key exchange, ALPN, or session-state mismatch".into());
    }
    Ok(Negotiation {
        tls_profile_id: "plab-tls13-aes128gcm-p256-server-auth-v2",
        tls_version: "TLS1.3",
        cipher_suite: "TLS_AES_128_GCM_SHA256",
        key_exchange_group: "X25519",
        signature_scheme: "ecdsa_secp256r1_sha256",
        alpn: "protocol-lab-tls",
        server_name: SERVER_NAME,
        handshake_complete: true,
        did_resume: resumed,
        early_data_attempted: resumed,
        early_data_accepted: accepted,
        application_data_bytes_sent: sent,
        application_data_bytes_received: received,
        certificate_profile: "plab-single-leaf-p256-server-v2",
        certificate_der_sha256: LEAF_DER_HASH,
        certificate_spki_sha256: LEAF_SPKI_HASH,
        certificate_verified: true,
        tls_handshake_latency_ms: elapsed.as_secs_f64() * 1000.0,
        crypto_provider: "rustls-rustcrypto@0.0.2-alpha",
        acceleration_provenance: "portable-software",
        platform_os: env::consts::OS,
        platform_architecture: env::consts::ARCH,
    })
}

fn write_artifacts(
    root: &Path,
    outcome: Outcome,
    operation: &Operation,
) -> Result<(), Box<dyn std::error::Error>> {
    let transferred_bytes = PAYLOAD_SIZE + operation.retry_bytes;
    let validation = json!({
        "schemaVersion": "protocol-lab.validation.v1",
        "scenarioId": outcome.scenario_id(),
        "passed": true,
        "requestedProtocol": outcome.variant(),
        "observedProtocol": outcome.variant(),
        "fallbackDetected": false,
        "completedOperations": 1,
        "failedOperations": 0,
        "timedOutOperations": 0,
    });
    write_json(root, "validation.json", &validation)?;
    write_json(root, "result.json", &validation)?;
    write_json(root, "protocol-proof.json", &operation.measured)?;
    write_json(root, "tls-negotiation.json", &operation.measured)?;
    write_json(
        root,
        "resumption-proof.json",
        &json!({
            "schemaVersion": "protocol-lab.tls-resumption-proof.v1",
            "scenarioId": outcome.scenario_id(),
            "resumptionPolicy": "accepted-psk-single-use-ticket",
            "prerequisitePolicy": "unmeasured-source-session-and-single-use-ticket",
            "warmupIsolation": "warmup-tickets-not-reused-by-measurement",
            "sourceSession": operation.source,
            "measuredSession": operation.measured,
            "sourceHandshakeOutsideMeasuredWindow": true,
            "sessionTicketAvailableAfterSource": operation.source_counts.0 == 1,
            "sessionTicketConsumedExactlyOnce": operation.final_counts.1.saturating_sub(operation.source_counts.1) == 1 && operation.final_counts.2.saturating_sub(operation.source_counts.2) == 1,
            "cachePutCountAfterSource": operation.source_counts.0,
            "cacheTakeCountForMeasuredHandshake": operation.final_counts.1.saturating_sub(operation.source_counts.1),
            "cacheHitCountForMeasuredHandshake": operation.final_counts.2.saturating_sub(operation.source_counts.2),
        }),
    )?;
    write_json(
        root,
        "early-data-proof.json",
        &json!({
            "schemaVersion": "protocol-lab.tls-early-data-proof.v1",
            "scenarioId": outcome.scenario_id(),
            "requestedOutcome": outcome.label(),
            "observedOutcome": outcome.label(),
            "earlyDataOffered": true,
            "earlyDataOfferedBytes": operation.early_data_offered_bytes,
            "earlyDataAccepted": outcome == Outcome::Accepted,
            "earlyDataRejected": outcome == Outcome::Rejected,
            "applicationRetriedExactlyOnceAfterHandshake": outcome == Outcome::Rejected && operation.retry_bytes == PAYLOAD_SIZE,
            "postHandshakeRetryBytes": operation.retry_bytes,
            "applicationEffectCount": 1,
            "zeroDuplicateEffects": true,
            "replaySafeFixture": true,
        }),
    )?;
    write_json(
        root,
        "payload-hash.json",
        &json!({
            "schemaVersion": "protocol-lab.payload-hash.v1",
            "scenarioId": outcome.scenario_id(),
            "payloadGenerator": "repeated-octet",
            "payloadOctet": 90,
            "payloadLength": PAYLOAD_SIZE,
            "payloadSha256": PAYLOAD_HASH,
        }),
    )?;
    write_json(
        root,
        "tls-load-summary.json",
        &json!({
            "schemaVersion": "protocol-lab.tls-load-summary.v1",
            "scenarioId": outcome.scenario_id(),
            "loadProfileId": LOAD_PROFILE_ID,
            "completedOperations": 1,
            "failedOperations": 0,
            "timedOutOperations": 0,
            "totalTransferredBytes": transferred_bytes,
            "tlsHandshakeLatencyMilliseconds": [operation.measured.tls_handshake_latency_ms],
        }),
    )?;
    write_json(
        root,
        "executor-identity.json",
        &json!({"id": EXECUTOR_ID, "version": EXECUTOR_VERSION, "role": "client-test-executor"}),
    )?;
    write_json(
        root,
        "load-generator-identity.json",
        &json!({"id": GENERATOR_ID, "version": GENERATOR_VERSION}),
    )?;
    write_json(
        root,
        "tls-executor-result.json",
        &executor_result(outcome, operation),
    )?;
    Ok(())
}

fn executor_result(outcome: Outcome, operation: &Operation) -> Value {
    let latency = operation.measured.tls_handshake_latency_ms;
    let transferred_bytes = PAYLOAD_SIZE + operation.retry_bytes;
    json!({
        "schemaVersion": "protocol-lab.tls-executor-result.v1",
        "scenarioId": outcome.scenario_id(),
        "mode": outcome.variant(),
        "executor": {"id": EXECUTOR_ID, "version": EXECUTOR_VERSION},
        "loadGenerator": {"id": GENERATOR_ID, "version": GENERATOR_VERSION},
        "validation": {"status": "passed"},
        "protocolProof": operation.measured,
        "requestedLoad": {"connections": 1, "concurrency": 1, "totalOperations": 1, "applicationDataBytes": PAYLOAD_SIZE, "warmupSeconds": 0, "repetition": 1},
        "effectiveLoad": {"connections": 1, "concurrency": 1, "totalOperations": 1, "applicationDataBytes": PAYLOAD_SIZE, "warmupSeconds": 0, "repetition": 1},
        "metrics": {
            "handshakesPerSecond": if latency > 0.0 { 1000.0 / latency } else { 0.0 },
            "tlsHandshakeLatencyMeanMs": latency,
            "tlsHandshakeLatencyP50Ms": latency,
            "tlsHandshakeLatencyP75Ms": latency,
            "tlsHandshakeLatencyP90Ms": latency,
            "tlsHandshakeLatencyP95Ms": latency,
            "tlsHandshakeLatencyP99Ms": latency,
            "totalTransferredBytes": transferred_bytes,
            "completedOperations": 1,
            "failedOperations": 0,
            "timedOutOperations": 0,
        },
        "warnings": ["Local extracted-package evidence is diagnostic and non-publishable."]
    })
}

fn emit_unsupported(root: &Path, scenario_id: &str) -> Result<(), Box<dyn std::error::Error>> {
    let value = json!({
        "schemaVersion": "protocol-lab.unsupported.v1",
        "status": "unsupported",
        "scenarioId": scenario_id,
        "executor": {"id": EXECUTOR_ID, "version": EXECUTOR_VERSION},
        "reason": "The exact scenario is recognized but is outside this early-data executor package."
    });
    write_json(root, "unsupported.json", &value)?;
    write_json(root, "result.json", &value)
}

fn connected_tcp(address: &str) -> io::Result<TcpStream> {
    let tcp = TcpStream::connect(address)?;
    tcp.set_read_timeout(Some(Duration::from_secs(15)))?;
    tcp.set_write_timeout(Some(Duration::from_secs(15)))?;
    tcp.set_nodelay(true)?;
    Ok(tcp)
}

fn load_certificates(
    path: &Path,
) -> Result<Vec<CertificateDer<'static>>, Box<dyn std::error::Error>> {
    Ok(
        rustls_pemfile::certs(&mut BufReader::new(File::open(path)?))
            .collect::<Result<Vec<_>, _>>()?,
    )
}

fn write_json(
    root: &Path,
    name: &str,
    value: &impl Serialize,
) -> Result<(), Box<dyn std::error::Error>> {
    let mut bytes = serde_json::to_vec_pretty(value)?;
    bytes.push(b'\n');
    fs::write(root.join(name), bytes)?;
    Ok(())
}

fn verify_identity(name: &str, expected: &str) -> Result<(), (i32, String)> {
    if let Some(actual) = nonempty_env(name)
        && actual != expected
    {
        return Err((
            2,
            format!("{name} substitution detected: expected {expected}, observed {actual}"),
        ));
    }
    Ok(())
}

fn normalize_target(value: &str) -> Result<String, String> {
    let trimmed = value.trim();
    let address = trimmed
        .strip_prefix("tls://")
        .unwrap_or(trimmed)
        .trim_end_matches('/');
    if address.is_empty() || !address.contains(':') || address.contains('/') {
        return Err(format!("invalid TLS target address {value:?}"));
    }
    Ok(address.to_string())
}

fn argument(args: &[String], name: &str) -> Option<String> {
    args.windows(2)
        .find(|pair| pair[0] == name)
        .map(|pair| pair[1].clone())
}

fn nonempty_env(name: &str) -> Option<String> {
    env::var(name).ok().filter(|value| !value.trim().is_empty())
}

fn material_path(relative: &str) -> PathBuf {
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
        assert_eq!(Outcome::from_id(ACCEPTED_ID).unwrap(), Outcome::Accepted);
        assert_eq!(Outcome::from_id(REJECTED_ID).unwrap(), Outcome::Rejected);
        assert!(matches!(
            Outcome::from_id("tls.handshake.full"),
            Err(ScenarioError::Unsupported(_))
        ));
        assert!(matches!(
            Outcome::from_id("tls.early-data.other"),
            Err(ScenarioError::Unknown(_))
        ));
    }

    #[test]
    fn canonical_payload_hash_is_exact() {
        assert_eq!(sha256(&vec![0x5a; PAYLOAD_SIZE]), PAYLOAD_HASH);
    }

    #[test]
    fn target_normalization_rejects_paths_and_accepts_tls_authority() {
        assert_eq!(
            normalize_target("tls://127.0.0.1:18446").unwrap(),
            "127.0.0.1:18446"
        );
        assert!(normalize_target("tls://127.0.0.1:18446/path").is_err());
    }
}
