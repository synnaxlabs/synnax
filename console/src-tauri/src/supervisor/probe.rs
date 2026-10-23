// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! The readiness probe for the embedded Core.

use std::future::Future;
use std::net::SocketAddr;
use std::pin::Pin;
use std::sync::Arc;
use std::time::Duration;

use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;

/// A readiness probe: resolves to true when the Core at the address serves its API.
pub type Probe =
    Arc<dyn Fn(SocketAddr) -> Pin<Box<dyn Future<Output = bool> + Send>> + Send + Sync>;

/// The Core endpoint that answers without a token.
const PATH: &str = "/api/v1/connectivity/check";

/// The longest a single probe waits for the Core to answer.
const TIMEOUT: Duration = Duration::from_secs(2);

/// Probes the connectivity check of the Core at `addr`.
pub fn check(addr: SocketAddr) -> Pin<Box<dyn Future<Output = bool> + Send>> {
    Box::pin(async move {
        tokio::time::timeout(TIMEOUT, request(addr))
            .await
            .map(|res| res.unwrap_or(false))
            .unwrap_or(false)
    })
}

async fn request(addr: SocketAddr) -> std::io::Result<bool> {
    let mut stream = TcpStream::connect(addr).await?;
    let req = format!(
        "POST {PATH} HTTP/1.1\r\nHost: {addr}\r\nContent-Type: application/json\r\n\
         Content-Length: 2\r\nConnection: close\r\n\r\n{{}}"
    );
    stream.write_all(req.as_bytes()).await?;
    let mut res = Vec::new();
    stream.read_to_end(&mut res).await?;
    Ok(is_ok(&res))
}

/// Reports whether the response starts with an HTTP 200 status line.
fn is_ok(res: &[u8]) -> bool {
    let line = res.split(|b| *b == b'\r').next().unwrap_or_default();
    let mut parts = line.split(|b| *b == b' ');
    matches!(
        (parts.next(), parts.next()),
        (Some(version), Some(b"200")) if version.starts_with(b"HTTP/1.")
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::net::TcpListener;

    /// Serves one connection with the response and returns the listener address.
    async fn serve(response: &'static str) -> SocketAddr {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            let (mut conn, _) = listener.accept().await.unwrap();
            let mut buf = [0u8; 1024];
            let n = conn.read(&mut buf).await.unwrap();
            assert!(buf[..n].starts_with(format!("POST {PATH} HTTP/1.1\r\n").as_bytes()));
            conn.write_all(response.as_bytes()).await.unwrap();
        });
        addr
    }

    #[tokio::test]
    async fn is_ready_when_the_core_answers_200() {
        let addr = serve("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n{}").await;
        assert!(check(addr).await);
    }

    #[tokio::test]
    async fn is_not_ready_on_any_other_status() {
        let addr = serve("HTTP/1.1 503 Service Unavailable\r\n\r\n").await;
        assert!(!check(addr).await);
    }

    #[tokio::test]
    async fn is_not_ready_on_a_response_that_is_not_http() {
        let addr = serve("200 hello\r\n").await;
        assert!(!check(addr).await);
    }

    #[tokio::test]
    async fn is_not_ready_when_nothing_listens() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        drop(listener);
        assert!(!check(addr).await);
    }
}
