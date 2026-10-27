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

/// Waits for the browser to return to the listener and returns the query string of
/// the redirect, which carries the code and state. Fails after five minutes.
#[tauri::command]
pub async fn oidc_wait(state: tauri::State<'_, Loopback>) -> Result<String, String> {
    let listener = state
        .0
        .lock()
        .map_err(|e| e.to_string())?
        .take()
        .ok_or("no login is waiting for a redirect")?;
    tauri::async_runtime::spawn_blocking(move || accept(&listener, TIMEOUT))
        .await
        .map_err(|e| e.to_string())?
}

/// Accepts connections until one requests the callback path, answers it with a page
/// that tells the person to return to the Console, and returns its query string.
fn accept(listener: &TcpListener, timeout: Duration) -> Result<String, String> {
    listener.set_nonblocking(true).map_err(|e| e.to_string())?;
    let deadline = Instant::now() + timeout;
    loop {
        match listener.accept() {
            Ok((stream, _)) => {
                if let Some(query) = respond(stream)? {
                    return Ok(query);
                }
            }
            Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                if Instant::now() >= deadline {
                    return Err("the login timed out".into());
                }
                std::thread::sleep(Duration::from_millis(50));
            }
            Err(e) => return Err(e.to_string()),
        }
    }
}

/// Reads one request line, within [`REQUEST_TIMEOUT`] and [`MAX_REQUEST_LINE`] bytes,
/// so a local process cannot hold the listener or grow its memory. A request for
/// `/callback` gets the page and yields its query; any other request, such as a favicon,
/// gets a 404 and yields nothing.
fn respond(mut stream: TcpStream) -> Result<Option<String>, String> {
    stream.set_nonblocking(false).map_err(|e| e.to_string())?;
    let line = read_request_line(&mut stream, Instant::now() + REQUEST_TIMEOUT);
    let target = line.split_whitespace().nth(1).unwrap_or("");
    let (path, query) = target.split_once('?').unwrap_or((target, ""));
    let found = path == "/callback";
    let (status, body) = if found {
        ("200 OK", PAGE)
    } else {
        ("404 Not Found", "")
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
    use std::io::Read;

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
        let query = accept(&listener, Duration::from_secs(5)).unwrap();
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
        assert_eq!(respond(stream).unwrap(), None);
        flood.join().unwrap();
    }

    #[test]
    fn times_out_without_a_redirect() {
        let listener = TcpListener::bind("127.0.0.1:0").unwrap();
        let err = accept(&listener, Duration::from_millis(100)).unwrap_err();
        assert_eq!(err, "the login timed out");
    }
}
