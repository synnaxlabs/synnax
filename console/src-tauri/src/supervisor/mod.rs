// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Supervises the embedded Core: starts it, reports when it is ready, restarts it after
//! a crash, and stops it on request.

pub mod commands;
mod launch;
pub mod probe;
pub mod restart;

use std::io;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::process::Stdio;
use std::time::{Duration, Instant};

use serde::Serialize;
use tokio::io::AsyncWriteExt;
use tokio::process::{Child, ChildStdin, Command};
use tokio::sync::{mpsc, oneshot, watch};

/// The line that asks the Core to stop.
const STOP_LINE: &[u8] = b"stop\n";

/// Configures a [`Supervisor`].
#[derive(Clone, Debug)]
pub struct Config {
    /// The Core executable.
    pub program: PathBuf,
    /// The directory that receives the Core config file.
    pub work_dir: PathBuf,
    /// The Core data directory.
    pub data_dir: PathBuf,
    /// The directory that receives the Core logs.
    pub log_dir: PathBuf,
    /// The longest a spawned Core may take to pass the probe before it is killed.
    pub start_timeout: Duration,
    /// The longest a Core may take to exit after a stop request before it is killed.
    pub stop_timeout: Duration,
    /// The pause between two probes of a Core that is not ready.
    pub probe_interval: Duration,
    /// The restart policy.
    pub restart: restart::Config,
    /// The readiness probe.
    pub probe: probe::Probe,
}

impl Config {
    /// Returns the production configuration for the given executable and directories.
    pub fn new(program: PathBuf, work_dir: PathBuf, data_dir: PathBuf, log_dir: PathBuf) -> Self {
        Self {
            program,
            work_dir,
            data_dir,
            log_dir,
            start_timeout: Duration::from_secs(60),
            stop_timeout: Duration::from_secs(30),
            probe_interval: Duration::from_millis(100),
            restart: restart::Config::default(),
            probe: probe::check,
        }
    }
}

/// How a client reaches the embedded Core.
#[derive(Clone, Debug, PartialEq, Serialize)]
pub struct Connection {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
}

/// The state of the embedded Core.
#[derive(Clone, Debug, PartialEq, Serialize)]
#[serde(tag = "state", rename_all = "snake_case")]
pub enum Status {
    /// The first Core of the launch is not ready yet.
    Starting,
    /// The Core serves its API.
    Running { connection: Connection },
    /// The Core exited unexpectedly and a new one is on its way.
    Restarting,
    /// The restart policy gave up. `message` describes the last failure.
    Failed { message: String },
    /// The Core was asked to stop.
    Stopping,
    /// The Core stopped on request.
    Stopped,
}

enum Request {
    /// Start the Core again from `Failed` or `Stopped`.
    Restart,
    /// Stop the Core. The sender resolves once the Core process has exited.
    Stop(oneshot::Sender<()>),
}

/// A handle to the supervisor task. Dropping every handle stops the Core.
#[derive(Clone)]
pub struct Supervisor {
    status: watch::Receiver<Status>,
    requests: mpsc::Sender<Request>,
}

impl Supervisor {
    /// Starts the supervisor and the first Core on the current Tokio runtime.
    pub fn open(cfg: Config) -> io::Result<Self> {
        let password = launch::password()?;
        std::fs::create_dir_all(&cfg.work_dir)?;
        std::fs::create_dir_all(&cfg.data_dir)?;
        std::fs::create_dir_all(&cfg.log_dir)?;
        let (status_tx, status) = watch::channel(Status::Starting);
        let (requests, requests_rx) = mpsc::channel(8);
        let task = Task {
            cfg,
            password,
            port: None,
            status: status_tx,
            requests: requests_rx,
        };
        tokio::spawn(task.run());
        Ok(Self { status, requests })
    }

    /// Returns the current status.
    pub fn status(&self) -> Status {
        self.status.borrow().clone()
    }

    /// Returns a receiver that observes every status change.
    pub fn subscribe(&self) -> watch::Receiver<Status> {
        self.status.clone()
    }

    /// Starts the Core again when it is `Failed` or `Stopped`. It has no effect in any
    /// other state.
    pub async fn restart(&self) {
        // A closed channel means the task is gone, and so is the Core.
        let _ = self.requests.send(Request::Restart).await;
    }

    /// Stops the Core and resolves once its process has exited.
    pub async fn stop(&self) {
        let (tx, rx) = oneshot::channel();
        if self.requests.send(Request::Stop(tx)).await.is_ok() {
            // A dropped sender means the task is gone, and so is the Core.
            let _ = rx.await;
        }
    }
}

/// How one run of the Core ended.
enum Outcome {
    /// The Core exited without a stop request.
    Exited { uptime: Duration, message: String },
    /// The Core exited after a stop request.
    Stopped(Option<oneshot::Sender<()>>),
}

struct Task {
    cfg: Config,
    password: String,
    /// The port of the last Core that became ready. The next Core reuses it, so the
    /// clients of the launch reconnect to the address they know.
    port: Option<u16>,
    status: watch::Sender<Status>,
    /// Closes when every `Supervisor` handle is dropped.
    requests: mpsc::Receiver<Request>,
}

impl Task {
    async fn run(mut self) {
        let mut policy = restart::Policy::new(self.cfg.restart);
        loop {
            match self.run_once().await {
                Outcome::Stopped(ack) => {
                    self.status.send_replace(Status::Stopped);
                    if let Some(ack) = ack {
                        let _ = ack.send(());
                    }
                }
                Outcome::Exited { uptime, message } => match policy.decide(uptime) {
                    restart::Decision::Restart(backoff) => {
                        self.status.send_replace(Status::Restarting);
                        if self.backoff(backoff).await {
                            continue;
                        }
                    }
                    restart::Decision::GiveUp => {
                        self.status.send_replace(Status::Failed { message });
                    }
                },
            }
            if !self.await_restart().await {
                return;
            }
            policy.reset();
            self.status.send_replace(Status::Starting);
        }
    }

    /// Waits out a restart backoff. It returns false when a stop request arrived first.
    async fn backoff(&mut self, backoff: Duration) -> bool {
        let sleep = tokio::time::sleep(backoff);
        tokio::pin!(sleep);
        loop {
            tokio::select! {
                _ = &mut sleep => return true,
                req = self.requests.recv() => match req {
                    Some(Request::Restart) => {}
                    Some(Request::Stop(ack)) => {
                        self.status.send_replace(Status::Stopped);
                        let _ = ack.send(());
                        return false;
                    }
                    None => return false,
                },
            }
        }
    }

    /// Waits for a restart request while no Core runs. It returns false when every
    /// handle is gone.
    async fn await_restart(&mut self) -> bool {
        loop {
            match self.requests.recv().await {
                Some(Request::Restart) => return true,
                Some(Request::Stop(ack)) => {
                    let _ = ack.send(());
                }
                None => return false,
            }
        }
    }

    /// Runs one Core process from spawn to exit.
    async fn run_once(&mut self) -> Outcome {
        let started = Instant::now();
        let exited = |message: String| Outcome::Exited {
            uptime: started.elapsed(),
            message,
        };
        let (mut child, port) = match self.spawn() {
            Ok(v) => v,
            Err(err) => return exited(format!("failed to start: {err}")),
        };
        // Child::wait closes the stdin it still holds, and a closed stdin stops the
        // Core, so the pipe lives outside the child.
        let mut stdin = child.stdin.take();
        let addr = SocketAddr::from((launch::HOST, port));
        let deadline = tokio::time::sleep(self.cfg.start_timeout);
        tokio::pin!(deadline);
        let mut probe = tokio::time::interval(self.cfg.probe_interval);
        probe.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);
        let mut ready = false;
        loop {
            tokio::select! {
                status = child.wait() => {
                    return exited(match status {
                        Ok(status) => format!("exited with {status}"),
                        Err(err) => format!("failed to wait for exit: {err}"),
                    });
                }
                req = self.requests.recv() => match req {
                    Some(Request::Restart) => {}
                    Some(Request::Stop(ack)) => {
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Stopped(Some(ack));
                    }
                    None => {
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Stopped(None);
                    }
                },
                _ = &mut deadline, if !ready => {
                    kill(&mut child).await;
                    return exited(format!("not ready after {:?}", self.cfg.start_timeout));
                }
                _ = probe.tick(), if !ready => {
                    if (self.cfg.probe)(addr).await {
                        ready = true;
                        self.port = Some(port);
                        self.status.send_replace(Status::Running {
                            connection: Connection {
                                host: launch::HOST.to_string(),
                                port,
                                username: launch::USERNAME.to_string(),
                                password: self.password.clone(),
                            },
                        });
                    }
                }
            }
        }
    }

    fn spawn(&self) -> io::Result<(Child, u16)> {
        let port = launch::port(self.port)?;
        let config = self.cfg.work_dir.join("core.json");
        launch::write_config(&config, port, &self.cfg.data_dir, &self.cfg.log_dir)?;
        let mut cmd = Command::new(&self.cfg.program);
        cmd.arg("start")
            .arg("--config")
            .arg(&config)
            .env(launch::PASSWORD_ENV, &self.password)
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(self.stderr()?)
            .kill_on_drop(true);
        // The Core gets its own process group, so a Ctrl+C in a development terminal
        // reaches the supervisor alone and the Core still stops in order.
        #[cfg(unix)]
        cmd.process_group(0);
        #[cfg(windows)]
        {
            const CREATE_NEW_PROCESS_GROUP: u32 = 0x0000_0200;
            const CREATE_NO_WINDOW: u32 = 0x0800_0000;
            cmd.creation_flags(CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW);
        }
        Ok((cmd.spawn()?, port))
    }

    /// Opens the file that receives the Core's stderr, where a Go panic lands. The file
    /// of the previous Core is kept under a second name.
    fn stderr(&self) -> io::Result<Stdio> {
        let path = self.cfg.log_dir.join("core-stderr.log");
        if path.exists() {
            std::fs::rename(&path, self.cfg.log_dir.join("core-stderr.previous.log"))?;
        }
        Ok(std::fs::File::create(path)?.into())
    }

    /// Asks the Core to stop and kills it when it outlives the stop timeout.
    async fn stop(&self, child: &mut Child, stdin: Option<&mut ChildStdin>) {
        self.status.send_replace(Status::Stopping);
        let asked = match stdin {
            Some(stdin) => stdin.write_all(STOP_LINE).await.is_ok(),
            None => false,
        };
        if asked
            && tokio::time::timeout(self.cfg.stop_timeout, child.wait())
                .await
                .is_ok()
        {
            return;
        }
        kill(child).await;
    }
}

async fn kill(child: &mut Child) {
    // kill fails only when the process has already exited.
    let _ = child.kill().await;
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use std::future::Future;
    use std::os::unix::fs::PermissionsExt;
    use std::pin::Pin;

    fn ready(_: SocketAddr) -> Pin<Box<dyn Future<Output = bool> + Send>> {
        Box::pin(async { true })
    }

    fn never_ready(_: SocketAddr) -> Pin<Box<dyn Future<Output = bool> + Send>> {
        Box::pin(async { false })
    }

    /// A stand-in Core that appends a line to `runs` on each start and stops on the
    /// stop line or when its stdin closes.
    const OBEDIENT: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
while read line; do [ "$line" = stop ] && exit 0; done
exit 0
"#;

    /// A stand-in Core that exits with a failure at once.
    const CRASHING: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
exit 3
"#;

    /// A stand-in Core that ignores every stop request.
    const STUBBORN: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
while true; do sleep 1; done
"#;

    struct Fixture {
        dir: tempfile::TempDir,
        cfg: Config,
    }

    impl Fixture {
        fn new(script: &str, probe: probe::Probe) -> Self {
            let dir = tempfile::tempdir().unwrap();
            let program = dir.path().join("core");
            std::fs::write(&program, script).unwrap();
            std::fs::set_permissions(&program, std::fs::Permissions::from_mode(0o755)).unwrap();
            let cfg = Config {
                start_timeout: Duration::from_secs(5),
                stop_timeout: Duration::from_millis(300),
                probe_interval: Duration::from_millis(5),
                restart: restart::Config {
                    base_interval: Duration::from_millis(5),
                    scale: 1.0,
                    max_retries: 2,
                    healthy_uptime: Duration::from_secs(60),
                },
                probe,
                ..Config::new(
                    program,
                    dir.path().join("work"),
                    dir.path().join("data"),
                    dir.path().join("logs"),
                )
            };
            Self { dir, cfg }
        }

        fn runs(&self) -> usize {
            std::fs::read_to_string(self.dir.path().join("runs"))
                .map(|s| s.lines().count())
                .unwrap_or(0)
        }

        /// Waits for the script to record its nth start. The injected probe passes
        /// before the script has run its first line.
        async fn wait_for_runs(&self, n: usize) {
            let poll = async {
                while self.runs() < n {
                    tokio::time::sleep(Duration::from_millis(5)).await;
                }
            };
            tokio::time::timeout(Duration::from_secs(10), poll)
                .await
                .expect("timed out waiting for the script to start");
        }
    }

    async fn wait_for(sup: &Supervisor, pred: impl Fn(&Status) -> bool) -> Status {
        let mut rx = sup.subscribe();
        tokio::time::timeout(Duration::from_secs(10), rx.wait_for(|s| pred(s)))
            .await
            .expect("timed out waiting for status")
            .unwrap()
            .clone()
    }

    #[tokio::test]
    async fn reports_running_with_the_connection_once_the_probe_passes() {
        let f = Fixture::new(OBEDIENT, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let status = wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let Status::Running { connection } = status else {
            unreachable!()
        };
        assert_eq!(connection.host, "127.0.0.1");
        assert_eq!(connection.username, "synnax");
        assert_eq!(connection.password.len(), 64);
        assert_ne!(connection.port, 0);
        sup.stop().await;
    }

    #[tokio::test]
    async fn stops_the_core_through_its_stdin() {
        let f = Fixture::new(OBEDIENT, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let started = Instant::now();
        sup.stop().await;
        assert!(started.elapsed() < f.cfg.stop_timeout);
        assert_eq!(sup.status(), Status::Stopped);
    }

    #[tokio::test]
    async fn kills_a_core_that_outlives_the_stop_timeout() {
        let f = Fixture::new(STUBBORN, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let started = Instant::now();
        sup.stop().await;
        assert!(started.elapsed() >= f.cfg.stop_timeout);
        assert_eq!(sup.status(), Status::Stopped);
    }

    #[tokio::test]
    async fn gives_up_on_a_crash_loop_and_starts_again_on_request() {
        let f = Fixture::new(CRASHING, never_ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let status = wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        let Status::Failed { message } = status else {
            unreachable!()
        };
        assert!(message.contains("exit status: 3"), "{message}");
        assert_eq!(f.runs(), 3);
        sup.restart().await;
        wait_for(&sup, |s| matches!(s, Status::Starting)).await;
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        assert_eq!(f.runs(), 6);
    }

    #[tokio::test]
    async fn kills_a_core_that_is_never_ready_and_counts_a_failed_run() {
        let mut f = Fixture::new(OBEDIENT, never_ready);
        f.cfg.start_timeout = Duration::from_millis(50);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let status = wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        assert_eq!(
            status,
            Status::Failed {
                message: "not ready after 50ms".to_string()
            }
        );
    }

    #[tokio::test]
    async fn restarts_a_core_that_exits_while_running() {
        let f = Fixture::new(OBEDIENT, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        std::process::Command::new("pkill")
            .args(["-f", f.cfg.program.to_str().unwrap()])
            .status()
            .unwrap();
        wait_for(&sup, |s| matches!(s, Status::Restarting)).await;
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(2).await;
        sup.stop().await;
    }

    #[tokio::test]
    async fn keeps_one_connection_for_every_core_of_a_launch() {
        let f = Fixture::new(OBEDIENT, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let connection = |s: Status| match s {
            Status::Running { connection } => connection,
            other => panic!("unexpected status {other:?}"),
        };
        let first = connection(wait_for(&sup, |s| matches!(s, Status::Running { .. })).await);
        sup.stop().await;
        sup.restart().await;
        wait_for(&sup, |s| matches!(s, Status::Starting)).await;
        let second = connection(wait_for(&sup, |s| matches!(s, Status::Running { .. })).await);
        assert_eq!(first, second);
        sup.stop().await;
    }

    #[tokio::test]
    async fn stops_the_core_when_every_handle_is_dropped() {
        let f = Fixture::new(OBEDIENT, ready);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let mut rx = sup.subscribe();
        drop(sup);
        let stopped = tokio::time::timeout(
            Duration::from_secs(10),
            rx.wait_for(|s| matches!(s, Status::Stopped)),
        )
        .await;
        assert!(stopped.is_ok());
    }
}
