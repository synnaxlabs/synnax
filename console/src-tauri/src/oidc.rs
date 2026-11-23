// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! The loopback redirect of an OpenID Connect login: the Console listens on a port of
//! 127.0.0.1 that only this machine reaches, and the browser returns the
//! authorization code to it.

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

/// How long a login waits for the browser to return.
const TIMEOUT: Duration = Duration::from_secs(300);

/// How long one connection may take to send its request line and headers.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(5);

/// The most of a request read, its line and headers. A browser's redirect fits well
/// within it.
const MAX_REQUEST: usize = 64 * 1024;

/// The error of a wait that a later login replaced.
const REPLACED: &str = "a newer login replaced this one";

const PAGE: &str = "<!doctype html><title>Synnax</title>\
    <p>Login complete. You can close this tab and return to the Console.</p>";
const REFUSED_PAGE: &str = "<!doctype html><title>Synnax</title>\
    <p>The provider refused the login. Return to the Console to try again.</p>";

/// The login in progress, if any.
#[derive(Default)]
pub struct Loopback(Mutex<Pending>);

#[derive(Default)]
struct Pending {
    /// The listener, until a wait takes it.
    listener: Option<TcpListener>,
    /// The port of the listener, which names the login to its wait.
    port: u16,
    /// Set when a later login replaces this one, which ends its wait.
    replaced: Arc<AtomicBool>,
}

impl Loopback {
    fn listen(&self) -> Result<u16, String> {
        let listener = TcpListener::bind("127.0.0.1:0").map_err(|e| e.to_string())?;
        let port = listener.local_addr().map_err(|e| e.to_string())?.port();
        let mut pending = self.0.lock().map_err(|e| e.to_string())?;
        pending.replaced.store(true, Ordering::Relaxed);
        *pending = Pending {
            listener: Some(listener),
            port,
            replaced: Arc::default(),
        };
        Ok(port)
    }

    async fn wait(&self, port: u16, state: String, timeout: Duration) -> Result<String, String> {
        let (listener, replaced) = {
            let mut pending = self.0.lock().map_err(|e| e.to_string())?;
            if pending.port != port {
                return Err(REPLACED.into());
            }
            let listener = pending
                .listener
                .take()
                .ok_or("no login is waiting for a redirect")?;
            (listener, pending.replaced.clone())
        };
        tauri::async_runtime::spawn_blocking(move || accept(&listener, &state, timeout, &replaced))
            .await
            .map_err(|e| e.to_string())?
    }
}

/// Listens on a free port of 127.0.0.1 and returns it. A later call replaces the
/// listener of an earlier login and ends its wait.
#[tauri::command]
pub fn oidc_listen(loopback: tauri::State<'_, Loopback>) -> Result<u16, String> {
    loopback.listen()
}

/// Waits for the browser to return to the listener on the given port with the given
/// login state, and returns the query string of the redirect, which carries the code
/// and state. Fails after five minutes, when a later login replaces this one, or at
/// once when no login is listening.
#[tauri::command]
pub async fn oidc_wait(
    loopback: tauri::State<'_, Loopback>,
    port: u16,
    state: String,
) -> Result<String, String> {
    loopback.wait(port, state, TIMEOUT).await
}

/// Accepts connections until one requests the callback path with the given state,
/// answers it with a page that tells the person to return to the Console, and returns
/// its query string. Every request counts against the timeout. Fails once `replaced`
/// is set.
fn accept(
    listener: &TcpListener,
    state: &str,
    timeout: Duration,
    replaced: &AtomicBool,
) -> Result<String, String> {
    listener.set_nonblocking(true).map_err(|e| e.to_string())?;
    let deadline = Instant::now() + timeout;
    loop {
        if replaced.load(Ordering::Relaxed) {
            return Err(REPLACED.into());
        }
        if Instant::now() >= deadline {
            return Err("the login timed out".into());
        }
        match listener.accept() {
            Ok((stream, _)) => {
                if let Some(query) = respond(stream, state, deadline) {
                    return Ok(query);
                }
            }
            // a connection reset before its accept fails that client, not the login
            Err(e)
                if matches!(
                    e.kind(),
                    std::io::ErrorKind::WouldBlock
                        | std::io::ErrorKind::Interrupted
                        | std::io::ErrorKind::ConnectionAborted
                        | std::io::ErrorKind::ConnectionReset
                ) =>
            {
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(e) => return Err(e.to_string()),
        }
    }
}

/// Reads one request, within [`REQUEST_TIMEOUT`], the deadline, and [`MAX_REQUEST`]
/// bytes, so a local process cannot hold the listener or grow its memory. A request for
/// `/callback` with the given state yields its query, and gets a page that says whether
/// the provider refused the login. A callback with another state gets a 400, any other
/// request a 404, and both yield nothing. A failed connection yields what its request
/// asked for, so it cannot end the login.
fn respond(mut stream: TcpStream, state: &str, deadline: Instant) -> Option<String> {
    if stream.set_nonblocking(false).is_err() {
        return None;
    }
    let line = read_request(&mut stream, deadline.min(Instant::now() + REQUEST_TIMEOUT));
    let target = line.split_whitespace().nth(1).unwrap_or("");
    let (path, query) = target.split_once('?').unwrap_or((target, ""));
    let (status, body, found) = if path != "/callback" {
        ("404 Not Found", "", false)
    } else if query
        .split('&')
        .any(|p| p.strip_prefix("state=") == Some(state))
    {
        let refused = query.split('&').any(|p| p.starts_with("error="));
        ("200 OK", if refused { REFUSED_PAGE } else { PAGE }, true)
    } else {
        ("400 Bad Request", "", false)
    };
    let _ = write!(
        stream,
        "HTTP/1.1 {status}\r\nContent-Type: text/html; charset=utf-8\r\n\
         Content-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    );
    found.then(|| query.to_string())
}

/// Reads the request line and headers up to the blank line that ends them, and returns
/// the request line. Leaving headers unread would make the close reset the connection,
/// and some browsers then drop the page. It stops early, with what it has read, at the
/// deadline, at [`MAX_REQUEST`] bytes, or when the connection closes or fails.
fn read_request(stream: &mut TcpStream, deadline: Instant) -> String {
    let mut req = Vec::new();
    let mut buf = [0u8; 1024];
    while req.len() < MAX_REQUEST {
        let remaining = deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() || stream.set_read_timeout(Some(remaining)).is_err() {
            break;
        }
        match stream.read(&mut buf) {
            Ok(0) | Err(_) => break,
            Ok(n) => {
                req.extend_from_slice(&buf[..n]);
                if req.windows(4).any(|w| w == b"\r\n\r\n") || req.windows(2).any(|w| w == b"\n\n")
                {
                    break;
                }
            }
        }
    }
    let end = req.iter().position(|&b| b == b'\n').unwrap_or(req.len());
    req.truncate(end.min(MAX_REQUEST));
    String::from_utf8_lossy(&req)
        .trim_end_matches('\r')
        .to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use std::sync::atomic::{AtomicBool, Ordering};

    fn get(port: u16, target: &str) -> String {
        let mut stream = TcpStream::connect(("127.0.0.1", port)).unwrap();
        write!(stream, "GET {target} HTTP/1.1\r\nHost: localhost\r\n\r\n").unwrap();
        let mut res = String::new();
        stream.read_to_string(&mut res).unwrap();
        res
    }

    #[test]
    fn returns_the_callback_query() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let client = std::thread::spawn(move || {
            let missing = get(port, "/favicon.ico");
            let page = get(port, "/callback?code=abc&state=xyz");
            (missing, page)
        });
        let query = accept(
            &listener,
            "xyz",
            Duration::from_secs(5),
            &AtomicBool::new(false),
        )
        .unwrap();
        let (missing, page) = client.join().unwrap();
        assert_eq!(query, "code=abc&state=xyz");
        assert!(missing.starts_with("HTTP/1.1 404"));
        assert!(page.starts_with("HTTP/1.1 200") && page.contains("Login complete"));
    }

    #[test]
    fn says_the_provider_refused_the_login() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let client =
            std::thread::spawn(move || get(port, "/callback?error=access_denied&state=xyz"));
        let query = accept(
            &listener,
            "xyz",
            Duration::from_secs(5),
            &AtomicBool::new(false),
        )
        .unwrap();
        let page = client.join().unwrap();
        assert_eq!(query, "error=access_denied&state=xyz");
        assert!(page.contains("The provider refused the login"));
        assert!(!page.contains("Login complete"));
    }

    #[test]
    fn stops_reading_an_endless_request_line() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let flood = std::thread::spawn(move || {
            let mut stream = TcpStream::connect(("127.0.0.1", port)).unwrap();
            let chunk = [b'a'; 1024];
            while stream.write_all(&chunk).is_ok() {}
        });
        let (stream, _) = listener.accept().unwrap();
        let deadline = Instant::now() + Duration::from_secs(5);
        assert_eq!(respond(stream, "xyz", deadline), None);
        flood.join().unwrap();
    }

    fn respond_unwritable(target: &str) -> Option<String> {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let mut client = TcpStream::connect(("127.0.0.1", port)).unwrap();
        write!(client, "GET {target} HTTP/1.1\r\n\r\n").unwrap();
        let (stream, _) = listener.accept().unwrap();
        stream.shutdown(std::net::Shutdown::Write).unwrap();
        respond(stream, "xyz", Instant::now() + Duration::from_secs(5))
    }

    #[test]
    fn keeps_the_callback_when_the_page_cannot_be_written() {
        assert_eq!(
            respond_unwritable("/callback?code=abc&state=xyz").as_deref(),
            Some("code=abc&state=xyz")
        );
    }

    #[test]
    fn ignores_a_connection_that_cannot_be_written() {
        assert_eq!(respond_unwritable("/favicon.ico"), None);
    }

    #[test]
    fn times_out_without_a_redirect() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let err = accept(
            &listener,
            "xyz",
            Duration::from_millis(100),
            &AtomicBool::new(false),
        )
        .unwrap_err();
        assert_eq!(err, "the login timed out");
    }

    #[test]
    fn ignores_a_callback_for_another_login() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let client = std::thread::spawn(move || {
            let other = get(port, "/callback?code=bad&state=other");
            let page = get(port, "/callback?code=abc&state=xyz");
            (other, page)
        });
        let query = accept(
            &listener,
            "xyz",
            Duration::from_secs(5),
            &AtomicBool::new(false),
        )
        .unwrap();
        let (other, page) = client.join().unwrap();
        assert_eq!(query, "code=abc&state=xyz");
        assert!(other.starts_with("HTTP/1.1 400"));
        assert!(page.starts_with("HTTP/1.1 200"));
    }

    #[test]
    fn times_out_while_requests_keep_arriving() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let done = Arc::new(AtomicBool::new(false));
        let flood_done = done.clone();
        let flood = std::thread::spawn(move || {
            while !flood_done.load(Ordering::Relaxed) {
                if let Ok(mut stream) = TcpStream::connect(("127.0.0.1", port)) {
                    let _ = write!(stream, "GET /callback?state=other HTTP/1.1\r\n\r\n");
                    let _ = stream.read_to_end(&mut Vec::new());
                }
            }
        });
        let start = Instant::now();
        let err = accept(
            &listener,
            "xyz",
            Duration::from_millis(300),
            &AtomicBool::new(false),
        )
        .unwrap_err();
        done.store(true, Ordering::Relaxed);
        drop(listener);
        flood.join().unwrap();
        assert_eq!(err, "the login timed out");
        assert!(start.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn times_out_while_a_request_stalls() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let stalled = TcpStream::connect(("127.0.0.1", port)).unwrap();
        let start = Instant::now();
        let err = accept(
            &listener,
            "xyz",
            Duration::from_millis(300),
            &AtomicBool::new(false),
        )
        .unwrap_err();
        drop(stalled);
        assert_eq!(err, "the login timed out");
        assert!(start.elapsed() < REQUEST_TIMEOUT);
    }

    #[test]
    fn waits_for_the_redirect_to_the_listening_port() {
        let loopback = Loopback::default();
        let port = loopback.listen().unwrap();
        let target = "/callback?code=abc&state=xyz";
        let client = std::thread::spawn(move || get(port, target));
        let wait = loopback.wait(port, "xyz".into(), Duration::from_secs(5));
        let query = tauri::async_runtime::block_on(wait).unwrap();
        assert_eq!(query, "code=abc&state=xyz");
        assert!(client.join().unwrap().starts_with("HTTP/1.1 200"));
    }

    #[test]
    fn fails_to_wait_without_a_listening_login() {
        let loopback = Loopback::default();
        let wait = loopback.wait(0, "xyz".into(), Duration::from_secs(5));
        let err = tauri::async_runtime::block_on(wait).unwrap_err();
        assert_eq!(err, "no login is waiting for a redirect");
    }

    #[test]
    fn waits_only_once_for_one_listen() {
        let loopback = Loopback::default();
        let port = loopback.listen().unwrap();
        let first = loopback.wait(port, "xyz".into(), Duration::from_millis(100));
        assert_eq!(
            tauri::async_runtime::block_on(first).unwrap_err(),
            "the login timed out"
        );
        let second = loopback.wait(port, "xyz".into(), Duration::from_millis(100));
        assert_eq!(
            tauri::async_runtime::block_on(second).unwrap_err(),
            "no login is waiting for a redirect"
        );
    }

    #[test]
    fn reads_headers_that_arrive_after_the_request_line() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let port = listener.local_addr().unwrap().port();
        let target = format!("/callback?code={}&state=xyz", "c".repeat(4096));
        let client = std::thread::spawn(move || {
            let mut stream = TcpStream::connect(("127.0.0.1", port)).unwrap();
            write!(stream, "GET {target} HTTP/1.1\r\n").unwrap();
            std::thread::sleep(Duration::from_millis(100));
            write!(stream, "Host: localhost\r\nAccept: text/html\r\n\r\n").unwrap();
            let mut res = String::new();
            stream.read_to_string(&mut res).unwrap();
            res
        });
        let never = AtomicBool::new(false);
        let query = accept(&listener, "xyz", Duration::from_secs(5), &never).unwrap();
        assert!(query.ends_with("&state=xyz"));
        assert!(client.join().unwrap().contains("Login complete"));
    }

    #[test]
    fn ends_the_wait_of_a_replaced_login() {
        let loopback = Arc::new(Loopback::default());
        let port = loopback.listen().unwrap();
        let waiting = loopback.clone();
        let first = std::thread::spawn(move || {
            tauri::async_runtime::block_on(waiting.wait(port, "xyz".into(), Duration::from_secs(5)))
        });
        while loopback.0.lock().unwrap().listener.is_some() {
            std::thread::sleep(Duration::from_millis(10));
        }
        let start = Instant::now();
        loopback.listen().unwrap();
        assert_eq!(first.join().unwrap().unwrap_err(), REPLACED);
        assert!(start.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn ends_a_replaced_login_that_has_not_started_waiting() {
        let loopback = Loopback::default();
        let first = loopback.listen().unwrap();
        let second = loopback.listen().unwrap();
        let stale = loopback.wait(first, "a".into(), Duration::from_secs(5));
        assert_eq!(tauri::async_runtime::block_on(stale).unwrap_err(), REPLACED);
        let client = std::thread::spawn(move || get(second, "/callback?state=b"));
        let wait = loopback.wait(second, "b".into(), Duration::from_secs(5));
        assert_eq!(tauri::async_runtime::block_on(wait).unwrap(), "state=b");
        client.join().unwrap();
    }

    #[test]
    fn replaces_the_listener_of_an_earlier_login() {
        let loopback = Loopback::default();
        let first = loopback.listen().unwrap();
        let second = loopback.listen().unwrap();
        assert!(TcpStream::connect(("127.0.0.1", first)).is_err());
        let client = std::thread::spawn(move || get(second, "/callback?state=xyz"));
        let wait = loopback.wait(second, "xyz".into(), Duration::from_secs(5));
        assert_eq!(tauri::async_runtime::block_on(wait).unwrap(), "state=xyz");
        client.join().unwrap();
    }
}
