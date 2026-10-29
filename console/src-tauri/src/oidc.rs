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
use std::sync::Mutex;
use std::time::{Duration, Instant};

/// How long a login waits for the browser to return.
const TIMEOUT: Duration = Duration::from_secs(300);

/// How long one connection may take to send its request line.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(5);

/// The longest request line read. A browser's redirect fits well within it.
const MAX_REQUEST_LINE: usize = 16 * 1024;

const PAGE: &str = "<!doctype html><title>Synnax</title>\
    <p>Login complete. You can close this tab and return to the Console.</p>";

/// The listener of the login in progress, if any.
#[derive(Default)]
pub struct Loopback(Mutex<Option<TcpListener>>);

/// Listens on a free port of 127.0.0.1 and returns it. A later call replaces the
/// listener of an earlier login.
#[tauri::command]
pub fn oidc_listen(state: tauri::State<'_, Loopback>) -> Result<u16, String> {
    let listener = TcpListener::bind("127.0.0.1:0").map_err(|e| e.to_string())?;
    let port = listener.local_addr().map_err(|e| e.to_string())?.port();
    *state.0.lock().map_err(|e| e.to_string())? = Some(listener);
    Ok(port)
}

/// Waits for the browser to return to the listener with the given login state, and
/// returns the query string of the redirect, which carries the code and state. Fails
/// after five minutes.
#[tauri::command]
pub async fn oidc_wait(
    loopback: tauri::State<'_, Loopback>,
    state: String,
) -> Result<String, String> {
    let listener = loopback
        .0
        .lock()
        .map_err(|e| e.to_string())?
        .take()
        .ok_or("no login is waiting for a redirect")?;
    tauri::async_runtime::spawn_blocking(move || accept(&listener, &state, TIMEOUT))
        .await
        .map_err(|e| e.to_string())?
}

/// Accepts connections until one requests the callback path with the given state,
/// answers it with a page that tells the person to return to the Console, and returns
/// its query string. Every request counts against the timeout.
fn accept(listener: &TcpListener, state: &str, timeout: Duration) -> Result<String, String> {
    listener.set_nonblocking(true).map_err(|e| e.to_string())?;
    let deadline = Instant::now() + timeout;
    loop {
        if Instant::now() >= deadline {
            return Err("the login timed out".into());
        }
        match listener.accept() {
            Ok((stream, _)) => {
                if let Some(query) = respond(stream, state, deadline)? {
                    return Ok(query);
                }
            }
            Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(e) => return Err(e.to_string()),
        }
    }
}

/// Reads one request line, within [`REQUEST_TIMEOUT`], the deadline, and
/// [`MAX_REQUEST_LINE`] bytes, so a local process cannot hold the listener or grow its
/// memory. A request for `/callback` with the given state gets the page and yields its
/// query. A callback with another state gets a 400, any other request a 404, and both
/// yield nothing.
fn respond(
    mut stream: TcpStream,
    state: &str,
    deadline: Instant,
) -> Result<Option<String>, String> {
    stream.set_nonblocking(false).map_err(|e| e.to_string())?;
    let line = read_request_line(&mut stream, deadline.min(Instant::now() + REQUEST_TIMEOUT));
    let target = line.split_whitespace().nth(1).unwrap_or("");
    let (path, query) = target.split_once('?').unwrap_or((target, ""));
    let (status, body, found) = if path != "/callback" {
        ("404 Not Found", "", false)
    } else if query
        .split('&')
        .any(|p| p.strip_prefix("state=") == Some(state))
    {
        ("200 OK", PAGE, true)
    } else {
        ("400 Bad Request", "", false)
    };
    write!(
        stream,
        "HTTP/1.1 {status}\r\nContent-Type: text/html; charset=utf-8\r\n\
         Content-Length: {}\r\nConnection: close\r\n\r\n{body}",
        body.len()
    )
    .map_err(|e| e.to_string())?;
    Ok(found.then(|| query.to_string()))
}

/// Reads up to the first line break. It stops early, with what it has read, at the
/// deadline, at [`MAX_REQUEST_LINE`] bytes, or when the connection closes or fails.
fn read_request_line(stream: &mut TcpStream, deadline: Instant) -> String {
    let mut line = Vec::new();
    let mut buf = [0u8; 1024];
    while line.len() < MAX_REQUEST_LINE {
        let remaining = deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() || stream.set_read_timeout(Some(remaining)).is_err() {
            break;
        }
        match stream.read(&mut buf) {
            Ok(0) | Err(_) => break,
            Ok(n) => {
                line.extend_from_slice(&buf[..n]);
                if let Some(end) = line.iter().position(|&b| b == b'\n') {
                    line.truncate(end);
                    break;
                }
            }
        }
    }
    line.truncate(MAX_REQUEST_LINE);
    String::from_utf8_lossy(&line).into_owned()
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
        let query = accept(&listener, "xyz", Duration::from_secs(5)).unwrap();
        let (missing, page) = client.join().unwrap();
        assert_eq!(query, "code=abc&state=xyz");
        assert!(missing.starts_with("HTTP/1.1 404"));
        assert!(page.starts_with("HTTP/1.1 200") && page.contains("Login complete"));
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
        assert_eq!(respond(stream, "xyz", deadline).unwrap(), None);
        flood.join().unwrap();
    }

    #[test]
    fn times_out_without_a_redirect() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let err = accept(&listener, "xyz", Duration::from_millis(100)).unwrap_err();
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
        let query = accept(&listener, "xyz", Duration::from_secs(5)).unwrap();
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
        let err = accept(&listener, "xyz", Duration::from_millis(300)).unwrap_err();
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
        let err = accept(&listener, "xyz", Duration::from_millis(300)).unwrap_err();
        drop(stalled);
        assert_eq!(err, "the login timed out");
        assert!(start.elapsed() < REQUEST_TIMEOUT);
    }
}
