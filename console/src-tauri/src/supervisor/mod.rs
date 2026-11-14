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

pub mod commands;
mod diagnostics;
pub mod discovery;
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

/// The name of the Core config file in the work directory.
const CONFIG_FILE: &str = "core.json";

/// The exit code of a Core whose data directory another process holds, `DataLocked` in
/// the Core's `cmd/exit` package.
const LOCKED_CODE: i32 = 75;

/// Configures a [`Supervisor`].
#[derive(Clone)]
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
    /// The pause between two starts while another process holds the data directory.
    pub wait_interval: Duration,
    /// The longest the supervisor waits for another process to free the data directory.
    pub wait_timeout: Duration,
    /// The pause between two probes of a Core that is not ready.
    pub probe_interval: Duration,
    /// The pause between two probes of a Core that is ready.
    pub liveness_interval: Duration,
    /// The number of consecutive failed probes after which a ready Core is stopped.
    pub liveness_failures: u32,
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
            wait_interval: Duration::from_secs(1),
            wait_timeout: Duration::from_secs(60),
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
    /// Another process holds the data directory. A Core starts once it is free.
    Waiting,
    /// The supervisor gave up on the Core. `message` describes the last failure.
    Failed { reason: Reason, message: String },
    /// The Core was asked to stop, and no new one follows.
    Stopping,
    /// The Core stopped on request.
    Stopped,
    /// The Core stops, or has stopped, because the app exits. No status follows.
    Exiting,
}

/// Why a Core exited without a stop request.
#[derive(Clone, Copy, Debug, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum Reason {
    /// The process never spawned.
    FailedToStart,
    /// The process exited on its own.
    Crashed,
    /// The process never passed the readiness probe.
    NotReady,
    /// A ready process stopped answering the probe.
    Unresponsive,
    /// Another process held the data directory for longer than the wait timeout.
    Locked,
}

/// How the last Core that was not asked to stop ended.
#[derive(Clone, Debug, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Exit {
    pub reason: Reason,
    /// What went wrong, for a person to read.
    pub message: String,
    /// How long the Core had run.
    pub uptime_seconds: u64,
    /// The exit code of a process that exited on its own.
    pub exit_code: Option<i32>,
}

/// What the Cores of one launch have done so far.
#[derive(Clone, Debug, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct History {
    /// The number of Cores started in this launch.
    pub starts: u32,
    /// The number of Cores that exited without a stop request. A wait for the data
    /// directory counts once, when it times out.
    pub exits: u32,
    /// The number of times the restart policy gave up on the Core.
    pub failures: u32,
    /// The number of Cores that became ready.
    pub readies: u32,
    /// When the current Core became ready, in milliseconds since the Unix epoch.
    pub ready_at: Option<u64>,
    /// How long the last Core that became ready took to do so.
    pub time_to_ready_ms: Option<u64>,
    /// How the last Core exited without a stop request.
    pub last_exit: Option<Exit>,
    /// The number of waits for the data directory that ended.
    pub waits: u32,
    /// How long the last wait for the data directory took.
    pub last_wait_ms: Option<u64>,
}

enum Request {
    /// Start a new Core, after a stop of the one that runs.
    Restart,
    /// Stop the Core.
    Stop(Stop),
    /// Stop the Core and erase what it stored. The sender resolves with the result.
    Reset(oneshot::Sender<io::Result<()>>),
}

struct Stop {
    /// Resolves once the Core process has exited.
    ack: oneshot::Sender<()>,
    /// The app exits after the stop.
    exit: bool,
}

impl Stop {
    fn stopping(&self) -> Status {
        if self.exit {
            Status::Exiting
        } else {
            Status::Stopping
        }
    }

    fn stopped(&self) -> Status {
        if self.exit {
            Status::Exiting
        } else {
            Status::Stopped
        }
    }
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
        // A config that cannot be read only costs the preference: the spawn writes a
        // new one.
        let port = launch::saved_port(&cfg.work_dir.join(CONFIG_FILE)).unwrap_or_else(|err| {
            eprintln!("failed to read the port of the last launch: {err}");
            None
        });
        std::fs::create_dir_all(&cfg.work_dir)?;
        std::fs::create_dir_all(&cfg.data_dir)?;
        std::fs::create_dir_all(&cfg.log_dir)?;
        let (status_tx, status) = watch::channel(Status::Starting);
        let (history_tx, history) = watch::channel(History::default());
        let (requests, requests_rx) = mpsc::channel(8);
        let task = Task {
            cfg,
            password,
            port,
            status: status_tx,
            history: history_tx,
            requests: requests_rx,
            waiting: None,
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
        self.request_stop(false).await;
    }

    /// Stops the Core because the app exits, and resolves once its process has exited.
    /// The status stays `Exiting`.
    pub async fn stop_for_exit(&self) {
        self.request_stop(true).await;
    }

    async fn request_stop(&self, exit: bool) {
        let (ack, rx) = oneshot::channel();
        if self
            .requests
            .send(Request::Stop(Stop { ack, exit }))
            .await
            .is_ok()
        {
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
    Exited {
        uptime: Duration,
        reason: Reason,
        message: String,
        code: Option<i32>,
    },
    /// The Core exited because another process holds the data directory.
    Locked,
    /// The Core exited after a stop request.
    Stopped(Option<Stop>),
    /// The Core exited after a restart request.
    Restart,
    /// The Core exited after a reset request.
    Reset(oneshot::Sender<io::Result<()>>),
}

struct Task {
    cfg: Config,
    password: String,
    /// The port of the last Core that became ready, first from the last launch. The
    /// next Core reuses it, so clients reconnect to the address they know.
    port: Option<u16>,
    status: watch::Sender<Status>,
    history: watch::Sender<History>,
    /// Closes when every `Supervisor` handle is dropped.
    requests: mpsc::Receiver<Request>,
    /// When the current wait for the data directory began.
    waiting: Option<Instant>,
}

impl Task {
    async fn run(mut self) {
        let mut policy = restart::Policy::new(self.cfg.restart);
        loop {
            let outcome = self.run_once().await;
            if !matches!(outcome, Outcome::Locked) {
                self.end_wait();
            }
            match outcome {
                Outcome::Stopped(stop) => {
                    let status = stop.as_ref().map_or(Status::Stopped, Stop::stopped);
                    self.status.send_replace(status);
                    if let Some(stop) = stop {
                        let _ = stop.ack.send(());
                    }
                }
                Outcome::Restart => {
                    policy.reset();
                    continue;
                }
                Outcome::Reset(ack) => self.erase(ack).await,
                Outcome::Locked => {
                    let since = *self.waiting.get_or_insert_with(Instant::now);
                    if since.elapsed() < self.cfg.wait_timeout {
                        self.status.send_replace(Status::Waiting);
                        if self.backoff(self.cfg.wait_interval).await {
                            continue;
                        }
                        self.end_wait();
                    } else {
                        self.end_wait();
                        let message = format!(
                            "the data directory stayed locked for {:?}",
                            self.cfg.wait_timeout
                        );
                        self.record_exit(Exit {
                            reason: Reason::Locked,
                            message: message.clone(),
                            uptime_seconds: 0,
                            exit_code: Some(LOCKED_CODE),
                        });
                        self.give_up(Reason::Locked, message);
                    }
                }
                Outcome::Exited {
                    uptime,
                    reason,
                    message,
                    ..
                } => match policy.decide(uptime) {
                    restart::Decision::Restart(backoff) => {
                        self.status.send_replace(Status::Restarting);
                        if self.backoff(backoff).await {
                            continue;
                        }
                    }
                    restart::Decision::GiveUp => self.give_up(reason, message),
                },
            }
            if !self.await_restart().await {
                return;
            }
            policy.reset();
            self.status.send_replace(Status::Starting);
        }
    }

    fn give_up(&self, reason: Reason, message: String) {
        self.history.send_modify(|h| h.failures += 1);
        self.status.send_replace(Status::Failed { reason, message });
    }

    fn record_exit(&self, exit: Exit) {
        self.history.send_modify(move |h| {
            h.exits += 1;
            h.last_exit = Some(exit);
        });
    }

    /// Records the end of the current wait for the data directory, if one runs.
    fn end_wait(&mut self) {
        let Some(since) = self.waiting.take() else {
            return;
        };
        let took = since.elapsed().as_millis() as u64;
        self.history.send_modify(|h| {
            h.waits += 1;
            h.last_wait_ms = Some(took);
        });
    }

    /// Waits out a restart backoff, which a restart request cuts short. It returns false
    /// when a stop request arrived first.
    async fn backoff(&mut self, backoff: Duration) -> bool {
        tokio::select! {
            _ = tokio::time::sleep(backoff) => true,
            req = self.requests.recv() => match req {
                Some(Request::Restart) => true,
                Some(Request::Stop(stop)) => {
                    self.status.send_replace(stop.stopped());
                    let _ = stop.ack.send(());
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
                Some(Request::Stop(stop)) => {
                    let _ = stop.ack.send(());
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
        if let Outcome::Exited {
            uptime,
            reason,
            message,
            code,
        } = &outcome
        {
            self.record_exit(Exit {
                reason: *reason,
                message: message.clone(),
                uptime_seconds: uptime.as_secs(),
                exit_code: *code,
            });
        }
        self.history.send_modify(|h| h.ready_at = None);
        outcome
    }

    async fn supervise(&mut self, started: Instant) -> Outcome {
        let exited = |reason: Reason, message: String, code: Option<i32>| Outcome::Exited {
            uptime: started.elapsed(),
            reason,
            message,
            code,
        };
        let cfg = self.cfg.clone();
        let sweep = move || {
            // A directory that a reset retired holds no live data, so one that cannot be
            // removed yet does not block the start.
            if let Err(err) = reset::sweep(&cfg) {
                eprintln!("failed to remove erased data: {err}");
            }
        };
        if let Err(err) = tokio::task::spawn_blocking(sweep).await {
            return exited(
                Reason::FailedToStart,
                format!("failed to remove erased data: {err}"),
                None,
            );
        }
        let (mut child, port) = match self.spawn() {
            Ok(v) => v,
            Err(err) => {
                return exited(
                    Reason::FailedToStart,
                    format!("failed to start: {err}"),
                    None,
                );
            }
        };
        // A wait for the data directory counts as one start.
        if self.waiting.is_none() {
            self.history.send_modify(|h| h.starts += 1);
        }
        // Child::wait closes the stdin it still holds, and a closed stdin stops the
        // Core, so the pipe lives outside the child.
        let mut stdin = child.stdin.take();
        let addr = SocketAddr::from((launch::HOST, port));
        let deadline = tokio::time::sleep(self.cfg.start_timeout);
        tokio::pin!(deadline);
        let mut probes = Prober::start(self.cfg.probe.clone(), addr, self.cfg.probe_interval);
        let mut ready = false;
        let mut missed = 0;
        let mut slowest = Duration::ZERO;
        loop {
            tokio::select! {
                status = child.wait() => {
                    return match status {
                        Ok(status) if status.code() == Some(LOCKED_CODE) => Outcome::Locked,
                        Ok(status) => {
                            exited(Reason::Crashed, format!("exited with {status}"), status.code())
                        }
                        Err(err) => exited(
                            Reason::Crashed,
                            format!("failed to wait for exit: {err}"),
                            None,
                        ),
                    };
                }
                req = self.requests.recv() => match req {
                    Some(Request::Restart) => {
                        self.status.send_replace(Status::Restarting);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Restart;
                    }
                    Some(Request::Stop(stop)) => {
                        self.status.send_replace(stop.stopping());
                        self.stop(&mut child, stdin.as_mut()).await;
                        return Outcome::Stopped(Some(stop));
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
                    return exited(
                        Reason::NotReady,
                        format!("not ready after {:?}", self.cfg.start_timeout),
                        None,
                    );
                }
                Some((ok, took)) = probes.results.recv() => {
                    if ok && !ready {
                        ready = true;
                        self.end_wait();
                        self.port = Some(port);
                        probes = Prober::start(
                            self.cfg.probe.clone(),
                            addr,
                            self.cfg.liveness_interval,
                        );
                        let elapsed = started.elapsed().as_millis() as u64;
                        self.history.send_modify(|h| {
                            h.readies += 1;
                            h.ready_at = Some(now_millis());
                            h.time_to_ready_ms = Some(elapsed);
                        });
                        self.status.send_replace(Status::Running {
                            connection: Connection {
                                host: launch::HOST.to_string(),
                                port,
                                username: launch::USERNAME.to_string(),
                                password: self.password.clone(),
                            },
                        });
                    }
                    if !ready {
                        continue;
                    }
                    if ok {
                        missed = 0;
                        slowest = Duration::ZERO;
                    } else {
                        missed += 1;
                        slowest = slowest.max(took);
                    }
                    if missed >= self.cfg.liveness_failures {
                        // Built before the stop, or its wait would count as uptime
                        // and a Core that hangs after each start would look healthy.
                        let outcome = exited(
                            Reason::Unresponsive,
                            format!(
                                "stopped answering after {missed} probes, slowest {} ms",
                                slowest.as_millis()
                            ),
                            None,
                        );
                        self.status.send_replace(Status::Restarting);
                        self.stop(&mut child, stdin.as_mut()).await;
                        return outcome;
                    }
                }
            }
        }
    }

    fn spawn(&self) -> io::Result<(Child, u16)> {
        let port = launch::port(self.port)?;
        let config = self.cfg.work_dir.join(CONFIG_FILE);
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
    /// Whether each probe passed and how long it took.
    results: mpsc::Receiver<(bool, Duration)>,
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
                let started = Instant::now();
                let ok = probe(addr).await;
                if tx.send((ok, started.elapsed())).await.is_err() {
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
    use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};

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

    /// A probe that passes once the file at `path` exists.
    fn exists(path: PathBuf) -> probe::Probe {
        Arc::new(move |_| {
            let path = path.clone();
            let answer: Pin<Box<dyn Future<Output = bool> + Send>> =
                Box::pin(async move { path.exists() });
            answer
        })
    }

    /// A stand-in Core that appends a line to `runs` on each start and stops on the
    /// stop line, which it records in `stops`, or when its stdin closes.
    const OBEDIENT: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
while read line; do
  [ "$line" = stop ] && { echo stop >> "$(dirname "$0")/stops"; exit 0; }
done
exit 0
"#;

    /// A stand-in Core that exits with a failure at once.
    const CRASHING: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
exit 3
"#;

    /// A stand-in Core that finds its data directory locked on its first three starts.
    /// Then it creates `ready` and behaves as `OBEDIENT`.
    const LOCKED_THRICE: &str = r#"#!/bin/sh
dir="$(dirname "$0")"
echo run >> "$dir/runs"
[ "$(wc -l < "$dir/runs" | tr -d ' ')" -le 3 ] && exit 75
touch "$dir/ready"
while read line; do
  [ "$line" = stop ] && { echo stop >> "$dir/stops"; exit 0; }
done
exit 0
"#;

    /// A stand-in Core that always finds its data directory locked.
    const LOCKED: &str = r#"#!/bin/sh
echo run >> "$(dirname "$0")/runs"
exit 75
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
                    dir.path().join("work"),
                    dir.path().join("data"),
                    dir.path().join("logs"),
                )
            };
            Self { dir, cfg }
        }

        fn lines(&self, name: &str) -> usize {
            std::fs::read_to_string(self.dir.path().join(name))
                .map(|s| s.lines().count())
                .unwrap_or(0)
        }

        fn runs(&self) -> usize {
            self.lines("runs")
        }

        fn stops(&self) -> usize {
            self.lines("stops")
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

    /// Returns the probe time, in milliseconds, of an unresponsive exit message.
    fn slowest_ms(message: &str) -> u128 {
        message
            .strip_prefix("stopped answering after 3 probes, slowest ")
            .and_then(|rest| rest.strip_suffix(" ms"))
            .unwrap()
            .parse()
            .unwrap()
    }

    /// Collects each status change of `sup` from now until the Core runs.
    fn record_until_running(sup: &Supervisor) -> JoinHandle<Vec<Status>> {
        let mut rx = sup.subscribe();
        rx.mark_unchanged();
        tokio::spawn(async move {
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
        })
    }

    /// Waits until the supervisor has recorded `n` exits.
    async fn wait_for_exits(sup: &Supervisor, n: u32) {
        let poll = async {
            while sup.history().exits < n {
                tokio::time::sleep(Duration::from_millis(5)).await;
            }
        };
        tokio::time::timeout(Duration::from_secs(10), poll)
            .await
            .expect("timed out waiting for the exit");
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
        assert_ne!(connection.port, 0);
        assert_eq!(sup.history().readies, 1);
        assert!(sup.history().time_to_ready_ms.is_some());
        sup.stop().await;
        assert!(sup.history().time_to_ready_ms.is_some());
    }

    #[tokio::test]
    async fn keeps_the_port_of_the_last_launch() {
        let f = Fixture::new(OBEDIENT, always(true));
        let port = |status: Status| match status {
            Status::Running { connection } => connection.port,
            other => panic!("expected running, got {other:?}"),
        };
        let first = Supervisor::open(f.cfg.clone()).unwrap();
        let before = port(wait_for(&first, |s| matches!(s, Status::Running { .. })).await);
        first.stop().await;
        let second = Supervisor::open(f.cfg.clone()).unwrap();
        let after = port(wait_for(&second, |s| matches!(s, Status::Running { .. })).await);
        second.stop().await;
        assert_eq!(after, before);
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
        let Status::Failed { reason, message } = status else {
            unreachable!()
        };
        assert_eq!(reason, Reason::Crashed);
        assert!(message.contains("exit status: 3"), "{message}");
        assert_eq!(f.runs(), 3);
        assert_eq!(sup.history().last_exit.unwrap().exit_code, Some(3));
        sup.restart().await;
        wait_for(&sup, |s| matches!(s, Status::Starting)).await;
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        assert_eq!(f.runs(), 6);
    }

    #[tokio::test]
    async fn counts_each_give_up_on_a_core_that_never_spawns() {
        let mut f = Fixture::new(CRASHING, always(false));
        f.cfg.program = f.dir.path().join("missing");
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        let first = sup.history();
        assert_eq!(first.starts, 0);
        assert_eq!(first.failures, 1);
        assert_eq!(first.last_exit.unwrap().reason, Reason::FailedToStart);
        sup.restart().await;
        wait_for(&sup, |s| matches!(s, Status::Starting)).await;
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        assert_eq!(sup.history().failures, 2);
    }

    #[tokio::test]
    async fn waits_for_a_locked_data_directory_without_using_up_the_restart_policy() {
        let mut f = Fixture::new(LOCKED_THRICE, always(false));
        f.cfg.probe = exists(f.dir.path().join("ready"));
        f.cfg.wait_interval = Duration::from_millis(5);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let seen = record_until_running(&sup);
        let seen = tokio::time::timeout(Duration::from_secs(10), seen)
            .await
            .expect("timed out waiting for the Core")
            .unwrap();
        assert!(seen.contains(&Status::Waiting), "{seen:?}");
        assert!(!seen.contains(&Status::Restarting), "{seen:?}");
        // More starts than the restart policy allows.
        f.wait_for_runs(4).await;
        let history = sup.history();
        assert_eq!(history.exits, 0);
        assert_eq!(history.starts, 1);
        assert_eq!(history.waits, 1);
        assert!(history.last_wait_ms.is_some());
        sup.stop().await;
    }

    #[tokio::test]
    async fn reports_exiting_through_a_stop_for_an_exit() {
        let f = Fixture::new(OBEDIENT, always(true));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        let mut rx = sup.subscribe();
        rx.mark_unchanged();
        let seen = tokio::spawn(async move {
            let mut seen = Vec::new();
            while rx.changed().await.is_ok() {
                seen.push(rx.borrow_and_update().clone());
            }
            seen
        });
        sup.stop_for_exit().await;
        assert_eq!(sup.status(), Status::Exiting);
        drop(sup);
        let seen = seen.await.unwrap();
        assert!(!seen.contains(&Status::Stopping), "{seen:?}");
        assert!(!seen.contains(&Status::Stopped), "{seen:?}");
    }

    #[tokio::test]
    async fn stops_while_it_waits_for_a_locked_data_directory() {
        let mut f = Fixture::new(LOCKED, always(false));
        f.cfg.wait_interval = Duration::from_secs(60);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Waiting)).await;
        sup.stop().await;
        assert_eq!(sup.status(), Status::Stopped);
        assert_eq!(sup.history().waits, 1);
    }

    #[tokio::test]
    async fn gives_up_on_a_data_directory_that_stays_locked() {
        let mut f = Fixture::new(LOCKED, always(false));
        f.cfg.wait_interval = Duration::from_millis(5);
        f.cfg.wait_timeout = Duration::from_millis(100);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        let status = wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        let Status::Failed { reason, .. } = status else {
            unreachable!()
        };
        assert_eq!(reason, Reason::Locked);
        let history = sup.history();
        assert_eq!(history.failures, 1);
        assert_eq!(history.exits, 1);
        assert_eq!(history.waits, 1);
        assert!(history.last_wait_ms.unwrap() >= 100);
        assert_eq!(history.last_exit.unwrap().exit_code, Some(LOCKED_CODE));
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
                reason: Reason::NotReady,
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
        wait_for_exits(&sup, 1).await;
        let history = sup.history();
        assert_eq!(history.exits, 1);
        let exit = history.last_exit.unwrap();
        assert_eq!(exit.reason, Reason::Unresponsive);
        ok.store(true, Ordering::SeqCst);
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(2).await;
        sup.stop().await;
    }

    #[tokio::test]
    async fn stops_a_core_that_stops_answering_through_its_stdin() {
        let ok = Arc::new(AtomicBool::new(true));
        let f = Fixture::new(OBEDIENT, following(ok.clone()));
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        let seen = record_until_running(&sup);
        ok.store(false, Ordering::SeqCst);
        wait_for_exits(&sup, 1).await;
        assert_eq!(f.stops(), 1);
        ok.store(true, Ordering::SeqCst);
        let seen = seen.await.unwrap();
        // A window keeps its workspace through a restart, but not through a stop.
        assert!(!seen.contains(&Status::Stopping), "{seen:?}");
        sup.stop().await;
    }

    #[tokio::test]
    async fn reports_the_slowest_failed_probe_of_a_core_that_stops_answering() {
        let ok = Arc::new(AtomicBool::new(true));
        let answer = ok.clone();
        let probe: probe::Probe = Arc::new(move |_| {
            let ok = answer.clone();
            let result: Pin<Box<dyn Future<Output = bool> + Send>> = Box::pin(async move {
                let ok = ok.load(Ordering::SeqCst);
                if !ok {
                    tokio::time::sleep(Duration::from_millis(40)).await;
                }
                ok
            });
            result
        });
        let f = Fixture::new(OBEDIENT, probe);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        f.wait_for_runs(1).await;
        ok.store(false, Ordering::SeqCst);
        wait_for_exits(&sup, 1).await;
        let message = sup.history().last_exit.unwrap().message;
        assert!(slowest_ms(&message) >= 40, "{message}");
        ok.store(true, Ordering::SeqCst);
        wait_for(&sup, |s| matches!(s, Status::Running { .. })).await;
        sup.stop().await;
    }

    #[tokio::test]
    async fn forgets_a_slow_probe_once_the_core_answers_again() {
        let calls = Arc::new(AtomicUsize::new(0));
        let probe: probe::Probe = Arc::new(move |_| {
            let call = calls.fetch_add(1, Ordering::SeqCst);
            let result: Pin<Box<dyn Future<Output = bool> + Send>> = Box::pin(async move {
                if call == 1 {
                    tokio::time::sleep(Duration::from_millis(300)).await;
                }
                matches!(call, 0 | 2)
            });
            result
        });
        let f = Fixture::new(OBEDIENT, probe);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for_exits(&sup, 1).await;
        let message = sup.history().last_exit.unwrap().message;
        assert!(slowest_ms(&message) < 300, "{message}");
        sup.stop().await;
    }

    #[tokio::test]
    async fn gives_up_on_a_core_that_hangs_after_each_start_despite_a_long_stop() {
        let calls = Arc::new(AtomicUsize::new(0));
        let probe: probe::Probe = Arc::new(move |_| {
            let ok = calls.fetch_add(1, Ordering::SeqCst) % 4 == 0;
            Box::pin(async move { ok })
        });
        let mut f = Fixture::new(STUBBORN, probe);
        f.cfg.stop_timeout = Duration::from_millis(600);
        f.cfg.restart.healthy_uptime = Duration::from_millis(500);
        let sup = Supervisor::open(f.cfg.clone()).unwrap();
        wait_for(&sup, |s| matches!(s, Status::Failed { .. })).await;
        assert_eq!(f.runs(), 3);
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
        let seen = record_until_running(&sup);
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
        assert_eq!(second.readies, 2);
        let exit = second.last_exit.unwrap();
        assert_eq!(exit.reason, Reason::Crashed);
        assert!(exit.message.contains("signal"));
        assert_eq!(exit.exit_code, None);
        sup.stop().await;
        assert_eq!(sup.history().ready_at, None);
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
