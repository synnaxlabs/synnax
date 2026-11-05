# 102 Deployment updates

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-04
- **Related**: [RFC 0058 - Release workflow](0058-release-workflow.md),
  [RFC 0062 - Account and licensing](0062-account-and-licensing.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

A deployment is one Core with the Consoles and external Drivers that connect to it. The
Core version becomes the version of the whole deployment. An admin updates the Core with
one command, `synnax update`. Each Console then offers the update that the Core serves,
and each Driver updates itself from the Core. A Console or Driver on another minor stops
work and says how to fix it. The Core backs up its metadata when its version changes and
refuses data that a newer Core wrote. Only users with `update` on `builtin:version` can
move a Console past its Core.

## 1 Motivation

- **A Console update took a test cell offline**: At an enterprise customer, someone
  clicked "Update" in the Console. The Console moved to a minor that the Core was not
  on, and runs "just didn't do anything". The customer asked that only admins can start
  updates, because a test cell may stop only in a scheduled window.
- **A mismatch only warns**: Every client warns and stays connected
  (`pluto/src/synnax/Provider.tsx:91-109`, `client/cpp/connection/checker.cpp:134-154`).
  The Core never checks a client version.
- **Nothing ties the Console to its Core**: Every 30 s the Console checks for the newest
  stable Console of any minor, and any user can install it
  (`console/src/platform/version/Updater.tsx:22-52`).
- **A Core update is manual**: `site/docs/src/pages/reference/enterprise/upgrade.mdx`
  lists eight steps, among them a hand copy of the data directory. Linux has no service.
- **The Core has no backup and no downgrade check**: It records no version on disk. Gorp
  skips applied migrations that it does not know (`gorp/migrate.go:88-109`), and Cesium
  ignores a channel version newer than its own
  (`cesium/internal/migrate/migrate.go:72`). An older Core reads newer data wrong.
- **External Drivers are invisible and manual**: No Driver reports its version
  (`schemas/synnax/rack.oracle:40-66`). Only Linux has a supervised service; Windows and
  macOS Drivers run in a terminal (`driver/daemon/daemon_noop.cpp:29-40`).

## 2 Principles

1. **The Core version is the deployment version**: No Console or Driver changes version
   until an admin updates the Core. Then each one follows the Core.
2. **A mismatch stops work where people can see it**: A warning that lets work go on is
   the failure that this RFC removes.
3. **The Core protects its own data at start**: Every update path (`synnax update`, the
   Windows installer, a Docker pull, a binary replaced by hand, Synnax Desktop) starts
   the same Core, so the backup and the downgrade check run there.
4. **Updates work without internet**: Test-cell networks and air-gapped sites update
   from the Core and from a release bundle.

## 3 Design

### 3.0 The recorded version

The Core writes its version to `<data>/VERSION`. In `storage.OpenLayer`, after
`acquireLock` and before `openKV` (`core/pkg/storage/layer.go:223-233`), it compares
that version with its own:

| Recorded version | Action                                           |
| ---------------- | ------------------------------------------------ |
| Same             | Opens.                                           |
| Older or none    | Backs up (§3.1), writes its version, then opens. |
| Newer            | Stops with the error below.                      |

The error reads: "Data was last opened by Core 0.60.1. Run Core 0.60.1 or newer, or undo
the update with `synnax update undo`."

The Core writes the file right after the backup, before any migration runs. A crash
during a migration then resumes at the next start and does not replace the good backup.
A `0.0.0` dev Core neither checks nor writes the file, as in the client checks (RFC 0058
§2.2).

### 3.1 Backups and undo

The backup copies `kv/` and the `meta.json` of every Cesium channel to
`<data>/backups/<unix seconds>-<old version>`, with a `.partial` name until the copy is
done. It keeps the newest 3. A failed copy stops the Core from starting. Telemetry files
stay out: Cesium migrations must not write them
(`cesium/internal/migrate/migrate.go:70`), and they can be terabytes. A Cesium migration
test fails when a migration changes a telemetry file. A release that changes the
telemetry format must also handle undo for that update. The embedded Driver key stays
out too: a Driver whose saved key fails after an undo makes a new one
(`core/pkg/driver/driver.go:480-517`). This is the Synnax Desktop backup
(`console/src-tauri/src/supervisor/backup.rs`) moved into the Core, and the Desktop copy
is deleted.

`synnax update undo` runs with the Core stopped. It puts back the newest backup, or the
one it names, writes its version to `VERSION`, and prints the Core version to run. With
no name, it refuses when `VERSION` already holds the newest backup's version: the last
update is already undone. Data with no `VERSION` file gives a backup named
`<unix seconds>-unknown`. Its undo removes `VERSION` and cannot name the Core to run,
and a second undo sees the missing file as `unknown` and refuses.
`synnax update history` lists the backups. An undo writes a marker first and removes it
last. While the marker exists, the Core does not start and tells the user to run the
undo again. It moves aside the Cesium directory of each channel that has no `meta.json`
in the backup. The channel key counter lives in `kv/`
(`core/pkg/distribution/channel/counter.go:22-38`), so it gives out those keys again,
and Cesium refuses a duplicate key (`cesium/channel.go:157-164`). Telemetry written to
known channels after the backup stays.

### 3.2 `synnax update`

`synnax update <version>` downloads the signed Core for the host from `core/vX.Y.Z`.
`synnax update --file <path>` takes a Core binary or a release bundle (§3.5). It needs
admin rights on the host, which is the access control for a Core update.

1. Stops the Core service. A Core that runs in a terminal must be stopped by hand first.
2. Keeps the old binary as `<binary>.old` and puts the new one in its place.
3. Starts the Core, which backs up and migrates (§3.0).
4. Waits for `/connectivity/check` to answer
   (`core/pkg/api/connectivity/connectivity.go:76-89`). A license state of `ok` before
   the update must stay `ok`, which covers the license version cap
   (`core/pkg/service/license/service.go:426-471`). A `missing` or `expired` state has
   nothing to lose.
5. On a failure, stops the Core and puts the old binary back. If step 3 made a backup,
   it first restores that backup (§3.1). With no new backup, the data did not change. It
   then starts the old Core.

`synnax update undo` also puts `<binary>.old` back when it exists.

Linux gets `synnax service install|uninstall|start|stop|status`, a systemd unit shaped
after `core/cmd/service/service_windows.go`. The Windows installer runs the same update
code instead of its own stop, replace, and start
(`.github/scripts/windows_installer.nsi:81-133`). Docker users pull and re-create; §3.0
and §3.1 still run. When a Core release exists past the connected Core, users with
`update` on `builtin:version` see a notice with the command for their Core.

### 3.3 Console updates

The Core build downloads the updater files of the Console release whose web Console it
embeds (`.github/workflows/build.synnax.yaml:684-691`): `Synnax.app.tar.gz` (15 MB), the
NSIS setup (13 MB), and their signatures. The Core serves a manifest for them at
`<core>/console/update/latest.json`. Updates never need the internet.

While connected, the Console checks its Core's manifest. A Rust command sets the
endpoint and version comparator at runtime (`tauri-plugin-updater` 2.14.1,
`src/updater.rs:246,277`), and the rule stays in TypeScript in
`console/src/platform/version/`. The Console offers the update when its minor differs
from the Core's, in either direction, or when the Core's Console is a newer patch.
`dangerousInsecureTransportProtocol` is on, because the Windows installer starts every
Core with `--insecure` (`.github/scripts/windows_installer.nsi:117`). The updater still
requires a valid signature (`console/src-tauri/tauri.conf.json:34-38`). The signature
does not cover the version in the manifest, so on an `--insecure` link an attacker in
the middle can serve an older signed build. That risk is accepted: the Console must take
older builds to match an older Core, and that attacker can already send commands.

A user with `update` on the built-in object `builtin:version` also sees the newest
stable Console from `docs.synnaxlabs.com/releases/console/latest.json` when it is past
the Core's minor. A confirmation names the break first. Only Owner has `update` on
built-in objects by default, as for `builtin:license`
(`core/pkg/api/license/license.go:90-117`). The Console checks it with `access.granted`.
This update comes from the docs site, not the Core, so only the Console can check it.
The check stops accidents; anyone with OS rights can still run an installer.

Synnax Desktop keeps its own updater, because its Core is embedded.

### 3.4 The mismatch screen

Every check of the TypeScript client already reports `clientServerCompatible`
(`client/ts/src/connection/status.ts:307`), and the client keeps checking. When it is
false, `Version.Guard` in `console/src/app/window/Guard.tsx` shows a full window in
place of the workspace: "Console 0.61.0 cannot work with Core 0.60.1". The guard sits
outside `License.Guard`: on an unlicensed Core, the user opens the matching web Console
to activate the license. When the Core reports a matching version, the screen lifts by
itself. The actions are "Update Console" (the §3.3 update, up or down), "Open in
browser" (the Console that the Core serves, which always matches), and the Core list. A
web page that the selected Core served reloads instead, because that Core serves the
Console that matches it. A page from any other origin shows the screen. The mismatch
warning, its notification actions (`console/src/feature/core/notifications.tsx:22-47`),
and the "Version mismatch" badge are deleted.

### 3.5 Drivers

The rack gains a `version` field (`schemas/synnax/rack.oracle:40`), which the Driver
writes with the rack record at each connect (`driver/rack/remote.cpp:116`). The Console
shows it on the rack.

The Core serves the Driver release that it embeds (`core/pkg/driver/enabled_unix.go:16`)
for every platform at `<core>/driver/update/<platform>`. It downloads a platform it does
not hold from `driver/vX.Y.Z` when a Driver first asks, checks the signature, and keeps
it in `<data>/drivers/`, which backups skip. A start flag turns the download off. Each
Core release also uploads one bundle per OS: the Core binary and the four Driver
binaries (about 310 MB together). `synnax update --file` loads the Drivers from it.

At each connect, a Driver whose version differs from the Core's Driver downloads it,
checks the signature, and replaces itself. It never updates while tasks run. A Core
update ends each Driver's stream, so the Driver stops its tasks and reconnects
(`driver/task/manager.cpp:201`, `driver/rack/rack.cpp:38-53`). The version check and the
update run at that connect, before any task starts again.

| Platform    | Restart after the replace                                     |
| ----------- | ------------------------------------------------------------- |
| Linux       | Exits non-zero, and systemd restarts it.                      |
| NI Linux RT | Runs `execv` on the new binary; nothing supervises it.        |
| Windows     | A new Windows service restarts it.                            |
| macOS       | A new launchd service with `KeepAlive` restarts it.           |
| `start -s`  | No self-update; the rack status names the version to install. |

On Linux, the binary moves to a path that the `synnax` service user owns, because today
root owns `/usr/local/bin/synnax-driver` (`driver/daemon/common.cpp:26`).

Today an early shutdown exits 0 (`driver/cmd/start.cpp:64`), so systemd does not restart
it; that changes to a non-zero exit. Release builds sign the Core and Driver binaries
with minisign, the format that the Console updater already checks.

A Driver on another patch that cannot update keeps its tasks running. A Driver on
another minor starts no tasks until it updates. Its rack status says "Driver 0.59.3
cannot work with Core 0.60.1", and the description tells the admin to load the bundle.
`0.0.0` builds are exempt.

## 4 What this RFC does not cover

- **Multi-node clusters**: Each Core of a cluster is a separate concern.
- **Disaster-recovery backups**: A backup of telemetry while the Core runs needs a
  consistent Cesium snapshot. The docs tell users to stop the Core and copy the data
  directory, or to use filesystem or VM snapshots.
- **Python and TypeScript packages**: pip and npm install them in user environments,
  which the Core cannot reach. They keep their warning.

## 5 Implementation phases

Phases 1 to 4 are low risk and add no update machinery. They make a mismatch visible,
protect the data on every update path that exists today, and show each rack's Driver
version. Phase 5 removes the cause of the test-cell incident. The Driver self-update has
the most risk, so it goes last. Each phase updates the docs pages that it changes.

| Phase | Scope                                                   | Touches                                                                                                           | Depends on |
| ----- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Block the Console when its minor differs from the Core  | `pluto/src/synnax`, `console/src/app/window`, `console/src/feature/version`, `console/src/platform/version`       | -          |
| 2     | Add Core backups, undo, and the downgrade check         | `core/pkg/storage`, `core/cmd`, `console/src-tauri/src/supervisor`                                                | -          |
| 3     | Report Driver versions and stop tasks on a mismatch     | `schemas/synnax/rack.oracle`, `driver/rack`, `driver/task`, `console/src/feature/rack`                            | -          |
| 4     | Serve the matching Console update from the Core         | `core/pkg/console`, `.github/workflows/build.synnax.yaml`                                                         | -          |
| 5     | Update the Console from its Core                        | `console/src/platform/version`, `console/src-tauri`                                                               | 1, 4       |
| 6     | Update the Core from a file with `synnax update`        | `core/cmd`, `.github/scripts/windows_installer.nsi`                                                               | 2          |
| 7     | Sign releases and download them in `synnax update`      | `.github/workflows/build.synnax.yaml`, `core/cmd`, `console/src/platform/version`, `site/docs/src/pages/releases` | 6          |
| 8     | Run the Driver as a service on every platform           | `driver/daemon`, `driver/cmd`                                                                                     | -          |
| 9     | Serve Driver binaries and release bundles from the Core | `core/pkg/driver`, `core/cmd`, `.github/workflows/release.yaml`                                                   | 7          |
| 10    | Update the Driver from its Core                         | `driver/update`, `driver/rack`                                                                                    | 3, 8, 9    |

Compatibility: a data directory from any stable release has no `VERSION` file, so its
first start backs up and writes one. Consoles from before this RFC keep polling
`releases/console/latest.json`, which stays. A Driver from before this RFC needs one
manual update to a release with phase 10.

## 6 Resolved decisions

1. **The Core bounds Console updates**: The rule lives in Console TypeScript, not in the
   Tauri updater. A desktop shell that loads its UI from the Core (the Ignition launcher
   model) was rejected; the shell and UI would need a stable contract.
2. **Admins may break the pair**: A mismatch is a normal step of every update. Only
   accidental breaks are stopped. A permission on the update button alone was rejected,
   because the default path must already be safe.
3. **A mismatch blocks**: A read-only mode was rejected; a Console that cannot trust the
   wire format should not show data from it.
4. **Core updates run on the host**: A Console button was rejected. The Core cannot
   replace itself in Docker or in a terminal (TeamCity's UI update has the same Docker
   limit), and host admin rights are already the right gate. Ignition also updates
   through an installer on the Gateway host. The word is "update" everywhere, as in the
   Console.
5. **The backup runs in the Core**: A step inside `synnax update` was rejected, because
   the installer, Docker, and hand updates would skip it.
6. **Downgrades are refused by version**: Per-layer checks for unknown migrations were
   rejected; they need code in Gorp, Cesium, and Pebble for downgrades that the docs
   already do not support. Postgres refuses data from another version the same way
   (`PG_VERSION`).
7. **Undo is in, general backup is out**: The §3.0 error needs `synnax update undo`. A
   metadata-only `synnax backup` would answer disaster recovery only in part, and the
   name would promise more. Disaster recovery belongs to snapshot tools such as Veeam
   (§4).
8. **`builtin:version` is the admin permission**: A check on the Owner role name was
   rejected; no Console code checks role names, and custom admin roles would fail.
9. **The Core serves the matching Console**: A docs-site route for the newest release on
   a minor was rejected. With it, a Console patch could reach an operator PC before the
   admin updates the Core. The cost is that a Console hotfix needs a Core patch; every
   Console patch since 0.59 already had one. This replaces the independent Console
   hotfix of RFC 0058 §2.2.
10. **Drivers update themselves from the Core**: A per-host `synnax-driver update` was
    rejected; the admin would visit every DAQ PC and cRIO after each Core update.
11. **The Core downloads Drivers on demand**: Downloading all four platforms in
    `synnax update` was rejected; other update paths skip it, and every Core would keep
    about 310 MB it may not use.
12. **A Driver on another minor runs no tasks**: With hardware, a wrong output is worse
    than no output.

## 7 Open questions

- The health-check timeout of `synnax update`, and the name of the start flag that turns
  off Driver downloads.
- Whether Driver and Core signatures reuse the Console's minisign key.
- The Linux path that the `synnax` service user owns for the Driver binary.
- How a Core built without the `driver` tag learns its Driver version.
