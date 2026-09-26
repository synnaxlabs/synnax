// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Supervises the embedded Core: starts it, reports when it is ready, restarts it after
//! a crash or a hang, and stops it on request.

mod backup;
pub mod commands;
mod diagnostics;
mod launch;
pub mod probe;
mod reset;
pub mod restart;

use std::io;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::process::Stdio;
use std::sync::Arc;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use serde::Serialize;
use tokio::io::AsyncWriteExt;
use tokio::process::{Child, ChildStdin, Command};
use tokio::sync::{mpsc, oneshot, watch};
use tokio::task::JoinHandle;

/// The line that asks the Core to stop.
const STOP_LINE: &[u8] = b"stop\n";

/// Configures a [`Supervisor`].
#[derive(Clone)]
pub struct Config {
    /// The Core executable.
    pub program: PathBuf,
    /// The version of the app, and so of the bundled Core.
    pub version: String,
    /// The directory that receives the Core config file and the data backups.
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
    /// The pause between two probes of a Core that is ready.
    pub liveness_interval: Duration,
    /// The number of consecutive failed probes after which a ready Core is killed.
    pub liveness_failures: u32,
    /// The restart policy.
    pub restart: restart::Config,
    /// The readiness probe.
    pub probe: probe::Probe,
}

impl Config {
    /// Returns the production configuration for the given executable, app version, and
    /// directories.
    pub fn new(
        program: PathBuf,
        version: String,
        work_dir: PathBuf,
        data_dir: PathBuf,
        log_dir: PathBuf,
    ) -> Self {
        Self {
            program,
            version,
            work_dir,
            data_dir,
            log_dir,
            start_timeout: Duration::from_secs(60),
            stop_timeout: Duration::from_secs(30),
            probe_interval: Duration::from_millis(100),
            liveness_interval: Duration::from_secs(5),
            liveness_failures: 3,
            restart: restart::Config::default(),
            probe: Arc::new(probe::check),
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
    /// The version of the Core, which is the version of the app.
    pub version: String,
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

/// What the Cores of one launch have done so far.
#[derive(Clone, Debug, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct History {
    /// The number of Cores started in this launch.
    pub starts: u32,
    /// When the current Core became ready, in milliseconds since the Unix epoch.
    pub ready_at: Option<u64>,
    /// Why the last Core exited without a stop request.
    pub last_exit: Option<String>,
}

enum Request {
    /// Start a new Core, after a stop of the one that runs.
    Restart,
    /// Stop the Core. The sender resolves once the Core process has exited.
    Stop(oneshot::Sender<()>),
    /// Stop the Core and erase what it stored. The sender resolves with the result.
    Reset(oneshot::Sender<io::Result<()>>),
}

/// A handle to the supervisor task. Dropping every handle stops the Core.
#[derive(Clone)]
pub struct Supervisor {
    status: watch::Receiver<Status>,
    history: watch::Receiver<History>,
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
        let (history_tx, history) = watch::channel(History::default());
        let (requests, requests_rx) = mpsc::channel(8);
        let task = Task {
            cfg,
            password,
            port: None,
            status: status_tx,
            history: history_tx,
            requests: requests_rx,
        };
        tokio::spawn(task.run());
        Ok(Self {
            status,
            history,
            requests,
        })
    }

    /// Returns what the Cores of this launch have done so far.
    pub fn history(&self) -> History {
        self.history.borrow().clone()
    }

    /// Returns the current status.
    pub fn status(&self) -> Status {
        self.status.borrow().clone()
    }

    /// Returns a receiver that observes every status change.
    pub fn subscribe(&self) -> watch::Receiver<Status> {
        self.status.clone()
    }

    /// Starts a new Core. A Core that runs stops in order first, and its clients see
    /// `Restarting`.
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

impl Supervisor {
    /// Stops the Core, erases its data and the backups, and leaves it `Stopped`.
    pub async fn reset(&self) -> io::Result<()> {
        let (tx, rx) = oneshot::channel();
        let gone = || io::Error::other("the supervisor has stopped");
        self.requests
            .send(Request::Reset(tx))
            .await
            .map_err(|_| gone())?;
        rx.await.map_err(|_| gone())?
    }
}

/// How one run of the Core ended.
enum Outcome {
    /// The Core exited without a stop request.
    Exited { uptime: Duration, message: String },
    /// The Core exited after a stop request.
    Stopped(Option<oneshot::Sender<()>>),
    /// The Core exited after a restart request.
    Restart,
    /// The Core exited after a reset request.
    Reset(oneshot::Sender<io::Result<()>>),
}

struct Task {
    cfg: Config,
    password: String,
    /// The port of the last Core that became ready. The next Core reuses it, so the
    /// clients of the launch reconnect to the address they know.
    port: Option<u16>,
    status: watch::Sender<Status>,
    history: watch::Sender<History>,
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
                Outcome::Restart => {
                    policy.reset();
                    continue;
                }
                Outcome::Reset(ack) => self.erase(ack).await,
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

    /// Waits out a restart backoff, which a restart request cuts short. It returns false
    /// when a stop request arrived first.
    async fn backoff(&mut self, backoff: Duration) -> bool {
        tokio::select! {
            _ = tokio::time::sleep(backoff) => true,
            req = self.requests.recv() => match req {
                Some(Request::Restart) => true,
                Some(Request::Stop(ack)) => {
                    self.status.send_replace(Status::Stopped);
                    let _ = ack.send(());
                    false
                }
                Some(Request::Reset(ack)) => {
                    self.erase(ack).await;
                    false
                }
                None => false,
            },
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
                Some(Request::Reset(ack)) => self.erase(ack).await,
                None => return false,
            }
        }
    }

    /// Erases the stored data while no Core runs and reports the result.
    async fn erase(&self, ack: oneshot::Sender<io::Result<()>>) {
        let cfg = self.cfg.clone();
        let result = tokio::task::spawn_blocking(move || reset::run(&cfg))
            .await
            .unwrap_or_else(|err| Err(io::Error::other(err)));
        self.status.send_replace(Status::Stopped);
        let _ = ack.send(result);
    }

    /// Runs one Core process from spawn to exit.
    async fn run_once(&mut self) -> Outcome {
        let started = Instant::now();
        let outcome = self.supervise(started).await;
        if let Outcome::Exited { message, .. } = &outcome {
            self.history
                .send_modify(|h| h.last_exit = Some(message.clone()));
        }
        self.history.send_modify(|h| h.ready_at = None);
        outcome
    }

    async fn supervise(&mut self, started: Instant) -> Outcome {
        let exited = |message: String| Outcome::Exited {
            uptime: started.elapsed(),
            message,
        };
        let cfg = self.cfg.clone();
        let prepare = move || {
            // A directory that a reset retired holds no live data, so one that cannot be
            // removed yet does not block the start.
            if let Err(err) = reset::sweep(&cfg) {
                eprintln!("failed to remove erased data: {err}");
            }
            backup::run(&cfg)
        };
        match tokio::task::spawn_blocking(prepare).await {
            Ok(Ok(())) => {}
            Ok(Err(err)) => return exited(format!("failed to back up the data: {err}")),
            Err(err) => return exited(format!("failed to back up the data: {err}")),
        }
        let (mut child, port) = match self.spawn() {
            Ok(v) => v,
            Err(err) => return exited(format!("failed to start: {err}")),
        };
        self.history.send_modify(|h| h.starts += 1);
        // Child::wait closes the stdin it still holds, and a closed stdin stops the
        // Core, so the pipe lives outside the child.
        let mut stdin = child.stdin.take();
        let addr = SocketAddr::from((launch::HOST, port));
        let deadline = tokio::time::sleep(self.cfg.start_timeout);
        tokio::pin!(deadline);
        let mut probes = Prober::start(self.cfg.probe.clone(), addr, self.cfg.probe_interval);
        let mut ready = false;
        let mut missed = 0;
        loop {
            tokio::select! {
                status = child.wait() => {
                    return exited(match status {
                        Ok(status) => format!("exited with {status}"),
                        Err(err) => format!("failed to wait for exit: {err}"),
                    });
                }
                req = self.requests.recv() => match req {
                    Some(Request::Restart) => {
                        self.status.send_replace(Status::Restarting);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Restart;
                    }
                    Some(Request::Stop(ack)) => {
                        self.status.send_replace(Status::Stopping);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Stopped(Some(ack));
                    }
                    Some(Request::Reset(ack)) => {
                        self.status.send_replace(Status::Stopping);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Reset(ack);
                    }
                    None => {
                        self.status.send_replace(Status::Stopping);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Stopped(None);
                    }
                },
                _ = &mut deadline, if !ready => {
                    kill(&mut child).await;
                    return exited(format!("not ready after {:?}", self.cfg.start_timeout));
                }
                Some(ok) = probes.results.recv() => {
                    if ok && !ready {
                        ready = true;
                        self.port = Some(port);
                        probes = Prober::start(
                            self.cfg.probe.clone(),
                            addr,
                            self.cfg.liveness_interval,
                        );
                        self.history.send_modify(|h| h.ready_at = Some(now_millis()));
                        self.status.send_replace(Status::Running {
                            connection: Connection {
                                host: launch::HOST.to_string(),
                                port,
                                username: launch::USERNAME.to_string(),
                                password: self.password.clone(),
                                version: self.cfg.version.clone(),
                            },
                        });
                    }
                    if !ready {
                        continue;
                    }
                    missed = if ok { 0 } else { missed + 1 };
                    if missed >= self.cfg.liveness_failures {
                        kill(&mut child).await;
                        return exited(format!("stopped answering after {missed} probes"));
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

fn now_millis() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

/// Probes the Core on an interval from its own task, so a slow probe never delays a
/// stop request or the notice of an exit. The task ends when the prober is dropped.
struct Prober {
    results: mpsc::Receiver<bool>,
    task: JoinHandle<()>,
}

impl Prober {
    fn start(probe: probe::Probe, addr: SocketAddr, interval: Duration) -> Self {
        let (tx, results) = mpsc::channel(1);
        let task = tokio::spawn(async move {
            let mut ticks = tokio::time::interval(interval);
            ticks.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);
            loop {
                ticks.tick().await;
                if tx.send(probe(addr).await).await.is_err() {
                    return;
                }
            }
        });
        Self { results, task }
    }
}

impl Drop for Prober {
    fn drop(&mut self) {
        self.task.abort();
    }
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use std::future::Future;
    use std::os::unix::fs::PermissionsExt;
    use std::pin::Pin;
    use std::sync::atomic::{AtomicBool, Ordering};

    fn always(ok: bool) -> probe::Probe {
        Arc::new(move |_| Box::pin(async move { ok }))
    }

    /// A probe that answers with the current value of `ok`.
    fn following(ok: Arc<AtomicBool>) -> probe::Probe {
        Arc::new(move |_| {
            let ok = ok.clone();
            let answer: Pin<Box<dyn Future<Output = bool> + Send>> =
                Box::pin(async move { ok.load(Ordering::SeqCst) });
            answer
        })
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
                liveness_interval: Duration::from_millis(5),
                liveness_failures: 3,
                restart: restart::Config {
                    base_interval: Duration::from_millis(5),
                    scale: 1.0,
                    max_retries: 2,
                    healthy_uptime: Duration::from_secs(60),
                },
                probe,
                ..Config::new(
                    program,
                    "0.0.0".to_string(),
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
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let status = wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let Status::Running { connection } = status else {
            unreachable!()
        };
        assert_eq!(connection.host, "127.0.0.1");
        assert_eq!(connection.username, "synnax");
        assert_eq!(connection.password.len(), 64);
        assert_eq!(connection.version, "0.0.0");
        assert_ne!(connection.port, 0);
        sup.stop().await;
    }

    #[tokio::test]
    async fn stops_the_core_through_its_stdin() {
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let started = Instant::now();
        sup.stop().await;
        assert!(started.elapsed() < f.cfg.stop_timeout);
        assert_eq!(sup.status(), Status::Stopped);
    }

    #[tokio::test]
    async fn kills_a_core_that_outlives_the_stop_timeout() {
        let f = Fixture::new(STUBBORN, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let started = Instant::now();
        sup.stop().await;
        assert!(started.elapsed() >= f.cfg.stop_timeout);
        assert_eq!(sup.status(), Status::Stopped);
    }

    #[tokio::test]
    async fn gives_up_on_a_crash_loop_and_starts_again_on_request() {
        let f = Fixture::new(CRASHING, always(false));
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
        let mut f = Fixture::new(OBEDIENT, always(false));
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
        let f = Fixture::new(OBEDIENT, always(true));
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
    async fn kills_and_replaces_a_core_that_stops_answering() {
        let ok = Arc::new(AtomicBool::new(true));
        let f = Fixture::new(STUBBORN, following(ok.clone()));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        ok.store(false, Ordering::SeqCst);
        wait_for(&sup, |s| matches!(s, Status::Restarting)).await;
        assert_eq!(
            sup.history().last_exit,
            Some("stopped answering after 3 probes".to_string())
        );
        ok.store(true, Ordering::SeqCst);
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(2).await;
        sup.stop().await;
    }

    #[tokio::test]
    async fn keeps_a_core_that_misses_fewer_probes_than_the_limit() {
        let ok = Arc::new(AtomicBool::new(true));
        let mut f = Fixture::new(OBEDIENT, following(ok.clone()));
        f.cfg.liveness_failures = u32::MAX;
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        ok.store(false, Ordering::SeqCst);
        tokio::time::sleep(Duration::from_millis(100)).await;
        assert!(matches!(sup.status(), Status::Running { .. }));
        assert_eq!(f.runs(), 1);
        sup.stop().await;
    }

    #[tokio::test]
    async fn replaces_a_running_core_on_request_without_a_stop_in_between() {
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        let mut rx = sup.subscribe();
        rx.mark_unchanged();
        let seen = tokio::spawn(async move {
            let mut seen = Vec::new();
            while rx.changed().await.is_ok() {
                let status = rx.borrow_and_update().clone();
                let done = matches!(status, Status::Running { .. });
                seen.push(status);
                if done {
                    break;
                }
            }
            seen
        });
        sup.restart().await;
        let seen = tokio::time::timeout(Duration::from_secs(10), seen)
            .await
            .expect("timed out waiting for the new Core")
            .unwrap();
        assert_eq!(seen[0], Status::Restarting);
        assert!(matches!(seen[seen.len() - 1], Status::Running { .. }));
        assert_eq!(seen.len(), 2);
        f.wait_for_runs(2).await;
        assert_eq!(sup.history().last_exit, None);
        sup.stop().await;
    }

    #[tokio::test]
    async fn records_the_starts_and_the_last_exit_of_the_launch() {
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        let first = sup.history();
        assert_eq!(first.starts, 1);
        assert!(first.ready_at.is_some());
        assert_eq!(first.last_exit, None);
        std::process::Command::new("pkill")
            .args(["-f", f.cfg.program.to_str().unwrap()])
            .status()
            .unwrap();
        wait_for(&sup, |s| matches!(s, Status::Restarting)).await;
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let second = sup.history();
        assert_eq!(second.starts, 2);
        assert!(second.last_exit.unwrap().contains("signal"));
        sup.stop().await;
        assert_eq!(sup.history().ready_at, None);
    }

    #[tokio::test]
    async fn backs_up_the_store_before_the_core_of_a_new_version_starts() {
        let f = Fixture::new(OBEDIENT, always(true));
        std::fs::create_dir_all(f.cfg.data_dir.join("kv")).unwrap();
        std::fs::write(f.cfg.data_dir.join("kv").join("MANIFEST"), "old").unwrap();
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let backups: Vec<_> = std::fs::read_dir(f.cfg.work_dir.join("backups"))
            .unwrap()
            .collect();
        assert_eq!(backups.len(), 1);
        sup.stop().await;
    }

    #[tokio::test]
    async fn erases_the_data_of_a_running_core_and_leaves_it_stopped() {
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        let file = f.cfg.data_dir.join("kv").join("MANIFEST");
        std::fs::create_dir_all(file.parent().unwrap()).unwrap();
        std::fs::write(&file, "data").unwrap();
        sup.reset().await.unwrap();
        assert_eq!(sup.status(), Status::Stopped);
        assert!(!file.exists());
        assert_eq!(f.runs(), 1);
        sup.restart().await;
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        sup.stop().await;
    }

    #[tokio::test]
    async fn erases_the_data_after_the_restart_policy_gave_up() {
        let f = Fixture::new(CRASHING, always(false));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        let file = f.cfg.data_dir.join("MANIFEST");
        std::fs::write(&file, "data").unwrap();
        sup.reset().await.unwrap();
        assert_eq!(sup.status(), Status::Stopped);
        assert!(!file.exists());
    }

    #[tokio::test]
    async fn keeps_one_connection_for_every_core_of_a_launch() {
        let f = Fixture::new(OBEDIENT, always(true));
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
        let f = Fixture::new(OBEDIENT, always(true));
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
