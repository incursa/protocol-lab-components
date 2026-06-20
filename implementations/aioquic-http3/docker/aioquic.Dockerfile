FROM python:3.12-slim

ARG AIOQUIC_VERSION=1.3.0
RUN python -m pip install --no-cache-dir "aioquic==${AIOQUIC_VERSION}"
WORKDIR /work
COPY docker/aioquic_http3_client.py /usr/local/bin/aioquic-http3-client
COPY docker/aioquic_http3_server.py /usr/local/bin/aioquic-http3-server
RUN chmod +x /usr/local/bin/aioquic-http3-client /usr/local/bin/aioquic-http3-server
RUN mkdir -p /www /certs \
    && printf 'aioquic HTTP/3 status\n' > /www/status \
    && printf 'aioquic HTTP/3 index\n' > /www/index.html \
    && python - <<'PY'
import datetime
import ipaddress
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import NameOID

key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
subject = issuer = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "localhost")])
now = datetime.datetime.utcnow()
certificate = (
    x509.CertificateBuilder()
    .subject_name(subject)
    .issuer_name(issuer)
    .public_key(key.public_key())
    .serial_number(x509.random_serial_number())
    .not_valid_before(now - datetime.timedelta(days=1))
    .not_valid_after(now + datetime.timedelta(days=3650))
    .add_extension(
        x509.SubjectAlternativeName(
            [x509.DNSName("localhost"), x509.IPAddress(ipaddress.ip_address("127.0.0.1"))]
        ),
        critical=False,
    )
    .sign(key, hashes.SHA256())
)

with open("/certs/priv.key", "wb") as handle:
    handle.write(
        key.private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.TraditionalOpenSSL,
            serialization.NoEncryption(),
        )
    )

with open("/certs/cert.pem", "wb") as handle:
    handle.write(certificate.public_bytes(serialization.Encoding.PEM))
PY
ENTRYPOINT ["python"]
