# 63 Synnax Desktop

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-21
- **Related**: [RFC 0018 - Embed local server in Console](0018-console-server-embed.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md)

## 0 Summary

Synnax Desktop is a standalone desktop product: a build of the Console code that bundles
a Core as a Tauri sidecar and manages it for the user. Desktop connects only to its own
Core. It has no login screen and no user management, and its interface never names the
Core, a host, or a port. A supervisor in the Rust shell starts the Core on a free
loopback port with a root password made for each launch, watches it, restarts it after a
crash, and stops it when the app exits. Desktop and the Console are one package: static
build flags select the features that each app wires in. The sidecar is the same Core
binary that the release pipeline already builds and signs, with the Driver embedded. The
Core gets two small changes: a loopback bind option and a flag that stops the Core when
its stdin closes.

## 1 Motivation

A new user installs two programs today. The installation page
(`hub/src/pages/reference/installation.mdx`) has four steps: install the Core from a
terminal, start it with three flags, install the Console, and log in with `synnax` and
`seldon`. Three of the four steps exist only because the Core is a separate install. An
instrumentation engineer at a bench wants to plug in a LabJack or NI device and see
data. If that engineer stops at the terminal step, the product never gets evaluated.

RFC 0018 proposed a sidecar in 2024 and stopped before a design. Two implementations
followed, both in TypeScript through the shell plugin: `console/src/cluster/local.ts`
(on `main` from February to July 2024, removed by `4aea485432`) and the
`cluster/embedded` package on the unmerged branch `sy-1494-synnax-community-edition`.
Both kept the Core process ID in Redux, because a webview reload lost the child handle
and the next page load had to find and kill the old process. Both also added an optional
local Core to a Console that still connected to remote Cores, so login, the Core list,
and credentials all stayed.

This RFC replaces RFC 0018. It makes Desktop a product of its own, and the Console
becomes part of the enterprise platform.

## 2 Vocabulary

- **Desktop**: The Synnax Desktop app. One installer, one process tree, one Core.
- **Console**: The existing app that connects to standalone Cores. It is unchanged by
  this RFC.
- **Embedded Core**: The Core process that Desktop starts. It is private to Desktop.
- **Supervisor**: The Rust module in the Desktop shell that owns the embedded Core
  process.
- **Launch**: One run of the Desktop app, from start to exit. The root password and the
  port live for one launch.
- **Wiring site**: A place in `console/src/app/` where a feature's exports enter a
  registry or wrap the window tree.

## 3 Prior art

Tauri v2 bundles a sidecar through `bundle.externalBin` but gives no lifecycle
management. Apps write their own spawn, health check, restart, and orphan cleanup
(tauri-apps/plugins-workspace issue 3062, tauri-apps/tauri issue 1896). Ollama's desktop
app has the shape this RFC uses: a main process spawns and supervises a Go server, and
the interface talks to it over a local port. Docker Desktop and Postgres.app do the same
and show the engine to the user; Desktop differs on purpose, because its user did not
ask for a server.

The closest prior art is in the repository. The Core supervises the Driver in
`core/pkg/driver/driver.go`: it spawns the child in its own process group (`unix.go:22`,
`windows.go:23`), writes `STOP` to stdin on cancel and escalates to a kill after
`StopTimeout` (`driver.go:532-533`), waits for a readiness message, and restarts through
a policy that gives up after repeated short runs (`internal/restart/restart.go:91`). The
Driver also stops when its stdin closes (`x/cpp/shutdown/unix.cpp:50`). The supervisor
follows this shape.

## 4 Principles

1. **Desktop is one product, not a Console mode**: Every surface that exists because a
   Console can reach many Cores is absent from the Desktop bundle. It is not hidden at
   run time.
2. **The embedded Core is an implementation detail**: No Desktop text names the Core, a
   host, a port, or credentials. A Core failure is an app failure.
3. **The shell owns the process**: The process handle, the port, and the password live
   in Rust for the full launch. No window owns the Core, and a webview reload changes
   nothing.
4. **The operating system cleans up orphans**: The Core stops when its stdin closes.
   Desktop never saves a process ID.
5. **No secret is saved**: The root password is made on each launch and stays in memory.
6. **One Core binary**: Desktop bundles the released Core. Differences arrive as
   configuration, never as a build tag.
7. **Only `app/` knows the build**: Build flags are read at wiring sites in the
   composition root. The `session`, `platform`, and `feature` layers stay the same for
   both apps.

## 5 Design

### 5.0 Process model

```
Desktop (Tauri, Rust)
  supervisor ── stdin ── Core (sidecar)
  webview(s) ── 127.0.0.1:<port> ──┘   └── Driver (embedded, existing)
```

The supervisor starts in the Tauri `setup` hook, before any window loads. The
`tauri-plugin-single-instance` plugin (`main.rs:107`) already keeps one Desktop process
per user, so one Core runs per user. The Core takes an exclusive lock on its data
directory (`core/pkg/storage/layer.go:319`), which guards the data if a second Core
starts anyway.

### 5.1 The supervisor

The supervisor is a Rust module behind a `desktop` Cargo feature
(`console/src-tauri/src/supervisor/`). It is one Tokio task that owns the Core process,
with a `watch` channel for the state and a request channel for restart and stop. It
spawns the Core with `tokio::process::Command`. Tauri still bundles the binary as a
sidecar through `bundle.externalBin`, but the process API of `tauri-plugin-shell` does
not fit: it gives no control of when stdin closes, and its children are not killed when
the handle drops. The supervisor holds the Core's stdin outside the child handle,
because `Child::wait` closes the stdin that it still holds, and a closed stdin stops the
Core (§5.2). The readiness probe is an injected function, so the tests run the real
lifecycle against a shell script that stands in for the Core.

**Start, in order:**

1. Make the launch password, 32 random bytes, once per launch.
2. Pick a free port: bind `127.0.0.1:0`, read the port, close the socket.
3. Write the Core config file (§5.2) and spawn the sidecar with `start --config <file>`.
   The password goes in the `SYNNAX_PASSWORD` environment variable
   (`core/cmd/cmd.go:57-58`), never in `argv`.
4. Poll `POST /api/v1/connectivity/check` (`core/pkg/transport/http/http.go:98`) until
   it answers. The endpoint needs no token (`core/pkg/api/layer.go:252`).
5. Report `running` with the connection parameters.

**States:**

| State        | Meaning                                    | Leaves on                   |
| ------------ | ------------------------------------------ | --------------------------- |
| `starting`   | First spawn of the launch, not yet ready   | Ready, or exit              |
| `running`    | Ready; connection parameters are available | Exit, hang, or a restart    |
| `restarting` | The Core is replaced; backoff, then spawn  | Ready, or policy gives up   |
| `failed`     | The restart policy gave up                 | The user asks for a restart |
| `stopping`   | A stop was requested                       | Process exit or kill        |
| `stopped`    | The Core exited on request                 | The app asks for a restart  |

**Restart policy.** The policy copies the Driver defaults (`driver.go:164-168`): a 2 s
base interval that scales by 1.1, and a run of 1 min counts as healthy and resets the
count. Desktop gives up after 5 failed runs in a row, not 100, because a person waits at
the screen. A Core that is not ready after 60 s is killed and counts as a failed run. A
restart keeps the launch password, and it keeps the port of the last Core that became
ready while that port is still free. The connection parameters then stay equal, so Pluto
keeps its client (`pluto/src/synnax/Provider.tsx:149`) and the client reconnects by
itself. When the port is taken, the supervisor picks a new one and Pluto builds a new
client.

**Liveness.** A Core that runs is probed every 5 s, each probe with a 2 s limit. After 3
failed probes in a row the supervisor kills the Core and treats the run as an unexpected
exit, so the restart policy covers a hang as it covers a crash. The probes run in their
own task, so a slow probe never delays a stop request.

**Manual restart.** `supervisor_restart` works in every state. A Core that runs stops in
order first, the state goes to `restarting` and not to `stopped`, so the workspace stays
mounted (§5.5), and the failure count resets.

**Backup before a version change.** The supervisor records the app version in
`core.version` beside the config file. Before it spawns a Core whose version differs, it
copies the metadata store (`core/kv`) to `backups/<unix seconds>-<old version>` and
keeps the newest 3. The copy is written under a `.partial` name and renamed when it is
complete. Telemetry (`core/cesium`) is left out: it is large, and the metadata
(channels, schematics, ranges, workspaces) is what a user cannot record again. A backup
that fails counts as a failed run, so a Core never migrates data that has no copy.

**Stop.** On `RunEvent::Exit` the supervisor writes `stop\n` to the Core's stdin and
waits up to 30 s, then kills. The Core stops the Driver in its own close path. If
Desktop dies without a clean exit, the Core's stdin closes and the Core stops itself
(§5.2), and the Driver stops the same way. The Core runs in its own process group, so a
Ctrl+C in a development terminal reaches the supervisor alone.

**Logs.** The config file points `log-file-path` into the app log directory. The Core's
stderr, where a Go panic lands, goes to `core-stderr.log` there, and the file of the
previous Core is kept under a second name.

**History.** The supervisor keeps, for the launch, the number of Cores started, the time
the current Core became ready, and the reason for the last unexpected exit.

**Surface to the webview.** Eight commands and one event, the first custom Tauri surface
in the Console shell:

- `supervisor_status`: Returns the state and, when `running`, the host, port, username,
  and password.
- `supervisor_restart`: Starts a new Core, after a stop of the one that runs.
- `supervisor_stop`: Stops the Core and resolves when its process has exited.
- `supervisor_show_logs`: Opens the log directory in the file manager.
- `supervisor_show_data`: Opens the data directory in the file manager.
- `supervisor_diagnostics`: Returns the app version, the history, the data and log
  directories, and the size of the data directory.
- `supervisor_log_tail`: Returns the last whole lines of `core.log`, at most 64 kB.
- `supervisor_export_diagnostics`: Writes a zip archive to a given path: a summary and
  every file of the log directory. The archive never holds the launch password.
- `supervisor://status`: Emitted to every window on each state change.

### 5.2 Core changes

**Loopback bind.** `net.Listen("tcp", l.Address.PortString())`
(`core/pkg/server/server.go:145`) drops the host, so `--listen localhost:9090` binds
every interface. Released deployments depend on that: the Docker image listens on the
default address and serves other hosts. The default does not change. `listener.Config`
(`core/cmd/listener/listener.go:41`) and `server.Listener` (`server.go:33`) gain a
`Loopback` boolean, spelled `loopback` in the list form of `listen`. A loopback listener
binds `127.0.0.1` and the same port, and its host must be `localhost` or `127.0.0.1`.

**Stop when stdin closes.** On `main`, end of input on stdin has no effect. The
`--stop-on-stdin-close` flag of `start` makes end of input send the same interrupt that
the `stop` line sends (`core/cmd/start/internal/stdin`). The flag defaults to off,
because a Core under systemd or the Windows service has a closed stdin from the start.

**Driver credentials.** The Core wrote the Driver's credentials under
`connection.credentials`, but the Driver reads `connection.username` and
`connection.password` (`client/cpp/synnax.h:82-83`). The embedded Driver thus always
logged in with the default password, and a Core with any other root password lost its
Driver. `core/pkg/driver/driver.go` now writes the keys that the Driver reads. Desktop
depends on the fix, because its password is never the default.

**Config file.** The supervisor writes `core.json` on each spawn, as the Windows service
writes its config in `WriteConfig` (`core/cmd/service/service_windows.go:64`). The file
holds no secret:

```json
{
  "listen": [{ "address": "127.0.0.1:<port>", "loopback": true }],
  "insecure": true,
  "username": "synnax",
  "data": "<app local data dir>/core",
  "log-file-path": "<app log dir>/core.log",
  "stop-on-stdin-close": true
}
```

The Core runs insecure. Traffic never leaves the loopback interface, and TLS there would
need a certificate that the webview trusts. `--insecure` changes TLS only; the token
middleware still guards every endpoint except login and the connectivity check
(`layer.go:239-258`). The Core reconciles the root user from its config on every start
(`core/pkg/service/user/root.go:50`), so a new password each launch needs no stored
state and no new Core mechanism. The root user holds the Owner role, so the permission
checks in about 40 Console files pass without change.

### 5.3 Build flavor

`vite.config.ts` reads `VITE_DESKTOP` and defines a `DESKTOP` constant, the same way
`VITE_IS_DEV` becomes `IS_DEV` (`vite.config.ts:17,157`). The constant is static, so the
bundler removes the dead branch and every module that only it imports. The built bundles
confirm it in both directions: the Desktop bundle holds no connect modal and no user
registration, and the Console bundle holds no supervisor code.

Only files under `console/src/app/` read `DESKTOP`. The login, Core, and user features
enter the app at eight wiring sites, and each one picks its entries with the constant:

| Wiring site              | Console                                | Desktop                  |
| ------------------------ | -------------------------------------- | ------------------------ |
| `window/Guard.tsx`       | `Auth.Guard`, `Auth.ConnectionGuard`   | `Embedded.Guard`         |
| `nav/bar/Top.tsx`        | `Core.Badge`                           | `Embedded.Indicator`     |
| `command/List.tsx`       | `Core.COMMANDS`, `User.COMMANDS`       | `Embedded.COMMANDS`      |
| `toolbars/toolbars.ts`   | `User.TOOLBAR`                         | None                     |
| `tree/Context.tsx`       | `Access.TREE_ITEMS`, `User.TREE_ITEMS` | None                     |
| `notifications/Feed.tsx` | `Core.NOTIFICATIONS`                   | None                     |
| `link/useDeep.ts`        | `Core.useLink`                         | None                     |
| `pluto/Context.tsx`      | The selected Core record               | `Embedded.useConnParams` |
| `App.tsx`                | None                                   | The Desktop providers    |

Both window shells mount `window/Guard.tsx`, which is the one place that picks the
guards. Login surfaces below `app/` become inputs, and three exist:

- `Project.Guard` takes a `standalone` flag. The project splash then shows no "Log out"
  action and no connection island.
- `Link.Disabled` is a context that removes every copy-link control below it. Desktop
  registers no URL scheme, so a copied link can open nothing.
- `Version.InstallProvider` wraps the install of a downloaded update in a middleware.
  Desktop stops the Core before the install and starts it again if the install fails
  (§5.6).

The Tauri side uses the same split. `tauri build --config` merges
`tauri.desktop.conf.json` over `tauri.conf.json`: product name, identifier,
`bundle.externalBin`, and the updater endpoint. CI already edits the config for one
build variant (`build.synnax.yaml:895`); the overlay file replaces that style of edit
for Desktop. The `desktop` Cargo feature compiles the supervisor and registers its
commands. `console/package.json` gains `dev-desktop` and `build-desktop` scripts that
set the variable and the overlay together, and the overlay turns the feature on through
`build.features`. The overlay also sets the Desktop icon, the gradient Synnax mark, so
the two apps differ in the Dock and the taskbar.

### 5.4 The `feature/embedded` package

One new leaf feature holds everything Desktop adds to the interface. No other feature
imports it, so it is a feature and not a platform package.

- `Embedded.Provider`: Reads `supervisor_status` once, follows the `supervisor://status`
  event, and provides the state. It listens before it reads, and an event beats an older
  answer.
- `Embedded.useConnParams`: The connection parameters of the last `running` state. They
  outlive a restart, so the client keeps its intent to connect.
- `Embedded.Guard`: Takes the place of both auth guards. It draws the states in §5.5
  and, on `running`, sets and selects the embedded Core record.
- `Embedded.Indicator`: The top bar status, shown only while the connection is not
  healthy.
- `Embedded.COMMANDS`: "Open diagnostics", "Restart Synnax", "Show logs", and "Show data
  folder".
- `Embedded.useDiagnosticsModal`: Opens the diagnostics dialog in §5.5.
- `Embedded.installMiddleware`: The update middleware in §5.3.

**The Core record.** Session state is split by cluster key, which the `core` slice
caches from the selected record (`console/src/session/store.ts:90-93`), so Desktop keeps
one record. `session/core/slice.ts` gains `EMBEDDED_KEY`, next to `LOCAL_KEY`,
`DEMO_KEY`, and `SERVED_KEY` (`slice.ts:36-45`). `SERVED_KEY` is the precedent: a record
that the runtime supplies and the user never edits
(`platform/core/detectConnection.ts:25-37`). The record holds a name and an empty
password. The launch password never enters Redux, because the `core` slice is saved to
disk in the `global` scope (`store.ts:53-57`). `app/pluto/Context.tsx` passes
`Embedded.useConnParams` to Pluto as `connParams`, so the client never reads the record.

Desktop has its own app identifier, so its `session.json` is separate from the Console's
and the two apps never share session state.

### 5.5 What the user sees

The role is an instrumentation engineer at first launch, and a test operator in a run.

- **Boot**: A splash that says "Starting Synnax", with no address and no retry count. It
  stays until the supervisor reports `running` and the connection settles
  (`Session.useSettled`). The Core waits up to 10 s for the Driver before it is ready
  (`driver.go:164`), and the splash covers that wait.
- **Project selection**: After boot the user sees the project selector
  (`feature/project/Guard.tsx`), as in the Console. Desktop does not create a project
  for the user.
- **Crash in a session**: The supervisor restarts the Core without a prompt. The
  workspace stays mounted and draws cached data, per RFC 0049 §4 "never block when
  settled". The client retries on its own at intervals of 5 s or less
  (`client/ts/src/connection/client.ts:130-131`). The indicator shows only while the
  connection is not healthy. The operator sees a short gap in live data and no dialog.
- **Restart gave up**: A full-window screen that says "Synnax stopped unexpectedly",
  with "Restart" and "Diagnostics".
- **Diagnostics**: A dialog for the person who helps a user, reached from the command
  palette and from the screen above. It shows the state, the version, the starts of the
  session, the last problem, the place and size of the data, and the end of the log in
  readable lines. Its buttons show the data folder, show the logs, export an archive for
  support, and restart. A restart of a Core that runs asks first, because it interrupts
  every task and every live plot. The dialog gives no raw access to the Core's stdin:
  the Core reads one word from it, and a button covers that word.

### 5.6 Build and release

Desktop ships where the Console ships: macOS arm64 and Windows x64. The Core is already
built, signed, and notarized for both with `-tags console,driver`. The Desktop job runs
after the Core job in `build.synnax.yaml`, downloads the `synnax-core-<os>` artifact,
and places it at `console/src-tauri/binaries/synnax-core-<target-triple>`, which Git
already ignores (`.gitignore:236`). Tauri signs the app bundle with the sidecar inside.

Desktop takes its version from `tauri.conf.json`, as the Console does.
`scripts/check_versions.sh` already holds that version to the Core's major and minor
numbers, and the bundle always carries the Core from the same commit, so the client and
the Core always match. Desktop has its own updater manifest, because an update for one
identifier must never install the other app: `console/release-spec.desktop.json`,
written by the same deploy job. Desktop assets go to the `console-v*` release under
names without a space (`Synnax-Desktop_*`), because GitHub rewrites a space in an asset
name.

An update stops the Core first. The Windows installer ends the app with no exit event,
and it cannot replace the executable of a Core that still runs. The updater plugin has
no hook that an app can set for that moment, so the update flow in the webview splits
into download, stop, and install (§5.3). The new Core migrates the data directory on its
next start.

`console/scripts/build_desktop.sh` makes a local build: it builds the Core from source
into `binaries/`, then an unsigned app with no updater artifacts. With `--dev` it runs
the app with hot reload. The Rust tests of the supervisor run in `test.console.yaml` on
macOS, which compiles the Tauri shell with no extra system packages.

## 6 Implementation phases

- **Phase 1: Core options.** The `loopback` listener field and the stop-on-stdin-close
  flag, with tests. No Console change. The Core stays green and its defaults hold.
- **Phase 2: Desktop.** The supervisor, the `desktop` feature and config overlay, the
  `DESKTOP` constant and wiring sites, `feature/embedded`, `EMBEDDED_KEY`, and the
  splash inputs. One phase, because no part runs without the others and the Console
  bundle is the green state at every commit.
- **Phase 3: Release.** The Desktop CI job, the updater manifest, and signing. The
  installation page waits for the first release, because its download link needs a
  published manifest.

## 7 What this RFC does not cover

- Portal account sign-in and the license that Desktop runs under. RFC 0062 defines both.
  The embedded Core refuses work until a license applies, and the license gate sits
  outside the embedded Core guard in `app/window/Guard.tsx`.
- Access to the embedded Core from the Python client, an external Driver, or a browser.
- A Linux build. The Console has no Linux bundle.
- A move of data between Desktop and a standalone Core.
- A backup of telemetry, and a restore flow in the interface. A person restores a
  metadata backup by hand.

## 8 Resolved decisions

1. **A Console mode with an optional embedded Core**: Rejected. Both earlier attempts
   had this shape, and login, the Core list, and saved credentials all stayed. The trade
   is real: a Desktop user who buys a standalone Core installs a second app.
2. **A no-auth mode in the Core**: Rejected. It adds a second permission path next to
   RBAC in the Core and a no-credential path in four clients. A password per launch
   needs no Core change, because the Core already reconciles root from config. The trade
   is real: the password sits in the memory of three processes and in the Driver config
   file for the length of a launch.
3. **A supervisor in TypeScript**: Rejected. A webview reload loses the child handle,
   which forced a saved process ID in both earlier attempts, and app exit events arrive
   only in Rust. The trade is real: the Console shell gets its first Rust module and its
   first custom commands.
4. **A saved process ID to clean up orphans**: Rejected for stop-on-stdin-close. The
   operating system closes the pipe on any parent death, and the Driver already works
   this way.
5. **A fixed port**: Rejected. Port 9090 collides with a standalone Core on the same
   machine, and a private Core needs no stable address. The supervisor picks the port,
   so the Core needs no way to report one. The trade is real: another process can take
   the port between the pick and the bind; the restart path covers it.
6. **A change to the default bind**: Rejected. Released deployments rely on
   `localhost:9090` binding every interface.
7. **A second package or a second `app/` root**: Rejected. About 20 `app/` directories
   would fork for eight wiring sites, and the copies would drift apart.
8. **A Desktop build of the Core**: Rejected. One binary means one build matrix and one
   signing path. The trade is real: the installer carries the embedded Console assets,
   which Desktop never serves.
9. **A project made on first launch**: Rejected. The user sees the project selector.
10. **A dialog on each Core crash**: Rejected. A restart that works needs no decision
    from the operator, and the give-up state catches a Core that keeps crashing.

11. **A terminal on the Core's stdin in the diagnostics dialog**: Rejected. The Core
    reads only `stop` from stdin, and a raw pipe shows the user the process that Desktop
    hides.
12. **A data guard in the uninstaller**: Not needed. The Tauri NSIS uninstaller deletes
    the app data only when the user ticks "Delete the application data", never in an
    update, and macOS leaves the data when the app goes to the trash.

## 9 Open questions

- The product name and identifier. The build uses "Synnax Desktop" and
  `com.synnaxlabs.desktop`.
- The installer size. The macOS arm64 Core with the Driver is about 195 MB before
  compression.
- The copy in shared surfaces that still says "Console", such as the update dialog.
