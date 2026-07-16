use std::{convert::Infallible, env, net::SocketAddr};

use bytes::Bytes;
use http_body_util::Full;
use hyper::service::service_fn;
use hyper::{Request, Response, StatusCode, body::Incoming, header};
use hyper_util::rt::{TokioExecutor, TokioIo};
use tokio::net::TcpListener;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let port = env::var("PLAB_HTTP_PORT")
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(8080);
    let protocol = env::var("PLAB_PROTOCOL").unwrap_or_else(|_| "h1".to_string());
    let listener = TcpListener::bind(SocketAddr::from(([0, 0, 0, 0], port))).await?;
    println!("rust-hyper-origin protocol={protocol} port={port}");
    loop {
        let (stream, _) = listener.accept().await?;
        let selected = protocol.clone();
        tokio::spawn(async move {
            let io = TokioIo::new(stream);
            let result = if selected == "h2" {
                hyper::server::conn::http2::Builder::new(TokioExecutor::new())
                    .serve_connection(io, service_fn(handle))
                    .await
            } else {
                hyper::server::conn::http1::Builder::new()
                    .serve_connection(io, service_fn(handle))
                    .await
            };
            if let Err(error) = result {
                eprintln!("connection error: {error}");
            }
        });
    }
}

async fn handle(request: Request<Incoming>) -> Result<Response<Full<Bytes>>, Infallible> {
    let (status, content_type, body) = response_for_path(request.uri().path());
    Ok(Response::builder()
        .status(status)
        .header(header::CONTENT_TYPE, content_type)
        .body(Full::new(Bytes::from(body)))
        .unwrap())
}

fn response_for_path(path: &str) -> (StatusCode, &'static str, &'static str) {
    match path {
        "/plaintext" => (StatusCode::OK, "text/plain", "Hello, World!"),
        "/json" => (
            StatusCode::OK,
            "application/json",
            "{\"message\":\"Hello, World!\"}",
        ),
        "/health" => (
            StatusCode::OK,
            "application/json",
            "{\"status\":\"ok\",\"implementationId\":\"rust-hyper-origin\"}",
        ),
        "/protocol-lab/metadata" => (
            StatusCode::OK,
            "application/json",
            "{\"implementationId\":\"rust-hyper-origin\",\"packageId\":\"org.protocol-lab.components.implementation.rust-hyper-origin\",\"supportedProtocols\":[\"h1\",\"h2\"]}",
        ),
        _ => (StatusCode::NOT_FOUND, "text/plain", "not found"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn common_rows_are_exact() {
        assert_eq!(
            response_for_path("/plaintext"),
            (StatusCode::OK, "text/plain", "Hello, World!")
        );
        assert_eq!(
            response_for_path("/json"),
            (
                StatusCode::OK,
                "application/json",
                "{\"message\":\"Hello, World!\"}"
            )
        );
    }
}
