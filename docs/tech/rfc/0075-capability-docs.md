# 75 Capability-first documentation

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-29
- **Related**: [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

The docs site (`site/docs`) groups pages by program: Concepts, Core, Control, Console,
Client libraries, and Device drivers. This RFC groups them by what a user wants to do:
Channels, Hardware, Visualize, Ranges, Control, and Alarms, with Get started first and
Client libraries and Enterprise last. One tree serves both editions. Synnax Desktop is
the default reader, and a badge marks each page, tab, or paragraph that only Enterprise
has. Each capability is written once, with Console, Python, and TypeScript tabs where
more than one interface exists. The sidebar in `_nav.ts` becomes the one source of page
order and edition. Every old URL redirects to its new page.

## 1 Motivation

Synnax Desktop (RFC 0063) ships a Console with an embedded Core. It keeps almost every
capability of the enterprise platform: channels, calculated channels, ranges, projects,
all visualizations, devices, tasks, the embedded Driver, and Arc. It drops login, users,
roles, copy link, and all outside access, so it has no Python or TypeScript client, no
external Driver, and no remote Core. The docs do not fit this product:

- **The front door is a server install**: `reference/installation.mdx` downloads the
  Core binary, moves it into `/usr/local/bin`, starts it with `--mem --insecure`, then
  installs the Console and logs in with `synnax` and `seldon`. A Desktop user does none
  of this.
- **Capabilities hide inside programs**: the only Console docs for channels, calculated
  channels, and ranges are Console tabs inside `client/channels.mdx`,
  `client/calculated-channels.mdx`, and `client/ranges.mdx`. The ranges toolbar links a
  Console user into the client library section.
- **The same content is written many times**: the Core install is in three pages
  (`installation.mdx:28-226`, `core/quick-start.mdx`, `core/installation.mdx`). Control
  authority is described in seven places. Connection parameters are explained six times.
  The redline text in `console/tables.mdx:62-76` and `console/schematics.mdx:287-300` is
  identical.
- **A new user meets distributed systems first**: after installation the sidebar opens
  Concepts, which starts with "horizontally scalable streaming time-series database" and
  a clusters-and-nodes page. Hardware comes last, and every vendor page requires "a
  Synnax Core running on your network" and "a Synnax Driver running on your network"
  (for example `driver/ni/configure-device.mdx:37-39`), although
  `driver/get-started.mdx:32-36` recommends the embedded Driver.
- **Whole workflows have no page**: ranges in the Console, statuses and alarms in the
  Console, data export (one sentence at `console/line-plots.mdx:244`), manual control,
  and everything Desktop adds (data folder, erase data, diagnostics, restart).

## 2 Vocabulary

- **Capability section**: A top-level sidebar section named for a user goal, such as
  Channels or Hardware.
- **Edition**: Desktop or Enterprise. Desktop is a subset of Enterprise, except for the
  pages that manage the embedded Core.
- **Edition marker**: A badge on a page, a tab, or a paragraph that names the only
  edition it applies to. Content with no marker applies to both.
- **Concept page**: The first page of a capability section. It explains the model (what
  a channel is) before the how-to pages.

## 3 Prior art

GitLab, Grafana, and Tailscale keep one tree and badge what a tier lacks; GitLab omits a
badge where the parent section already has it, and puts no badges on tutorials. InfluxDB
3 renders a tree per edition from shared source, which pays only when much of the
content differs. Docker's Desktop manual covers only what is unique to Desktop and
documents shared capabilities once. Supabase and Stripe put the GUI and each language in
tabs on one capability page. Home Assistant keeps device integrations as a catalog of
same-shape pages. We align with the one-tree and badge model because the Desktop
difference set is small, with Supabase's tabs because 42 pages already use a
Console/Python/TypeScript switcher, and with Docker by giving each edition its own setup
pages. Ignition's model, a manual per licensed module with edition limits on one side
page, is the failure mode this RFC removes.

## 4 Principles

1. **A reader finds a capability by goal, not by program**: section names are user
   goals. A program name appears only in the Enterprise and Client libraries sections,
   where the program is the subject.
2. **One page per task**: a page covers one task, such as create, find, or delete, and
   its title names that task. No page is a catch-all "manage" page. Interfaces are tabs
   on the page, never parallel pages in separate sections. Topics that exist only in
   code (frames, iterators, time types) live in Client libraries.
3. **Desktop is the default reader**: shared pages lead with the Console, and the
   Console tab is first. Enterprise-only content carries a marker.
4. **The nav is the one source of order and edition**: `_nav.ts` defines the reading
   order, the previous and next links, and the edition of each page. Front matter does
   not repeat them.
5. **Old URLs never break**: every moved page gets a redirect, and every anchor that a
   shipped binary prints stays valid.
6. **Each phase leaves the live site whole**: every page stays reachable from the nav
   after each merge.

## 5 Design

### 5.0 The tree

| Section          | Path                    | Edition    | Holds                                                                       |
| ---------------- | ----------------------- | ---------- | --------------------------------------------------------------------------- |
| Get started      | `/reference/`           | both       | Overview, editions, Desktop install, first run, UI overview, Desktop upkeep |
| Channels         | `/reference/channels`   | both       | Concept, create, find, rename and delete, calculated                        |
| Hardware         | `/reference/hardware`   | both       | Driver overview, tasks, one page set per vendor                             |
| Visualize        | `/reference/visualize`  | both       | Projects, line plots, schematics, tables, logs                              |
| Ranges           | `/reference/ranges`     | both       | Concept, create, find, edit, metadata, aliases, data, export                |
| Control          | `/reference/control`    | both       | Overview, manual control, authority, Arc, Python sequences                  |
| Alarms           | `/reference/alarms`     | both       | Statuses, redlines, Arc alarms, PagerDuty                                   |
| Client libraries | `/reference/client`     | Enterprise | Code-only topics                                                            |
| Enterprise       | `/reference/enterprise` | Enterprise | Core setup, production, users, standalone Driver, CLI                       |

The `/reference/` prefix stays. It keeps the header tab logic
(`components/layout/Header.astro:34-56` tests `url.includes("reference")`) and the
Algolia icon matching unchanged, and a new prefix would add no value for the reader.

### 5.1 Page map

Every current page appears once. "NEW" marks a page with no current source; "merge"
marks a page built from several sources, with the duplicate text deleted.

**Get started**

| New path                                         | Source                                                |
| ------------------------------------------------ | ----------------------------------------------------- |
| `/reference/`                                    | `index.mdx`, rewritten Desktop-first                  |
| `/reference/get-started/editions`                | NEW: the edition table                                |
| `/reference/get-started/install-desktop`         | NEW (Desktop): from branch `sy-4856-desktop-download` |
| `/reference/get-started/first-run`               | NEW: connect, plot, record a range                    |
| `/reference/get-started/ui-overview`             | `console/ui-overview.mdx`                             |
| `/reference/get-started/desktop/updates`         | NEW (Desktop): updates and the metadata backup        |
| `/reference/get-started/desktop/data`            | NEW (Desktop): the data folder, erase all data        |
| `/reference/get-started/desktop/troubleshooting` | NEW (Desktop): restart, logs, diagnostics             |

**Channels**

| New path                                | Source                                                                   |
| --------------------------------------- | ------------------------------------------------------------------------ |
| `/reference/channels/`                  | merge: `concepts/channels.mdx`, `concepts/overview.mdx:68-97`            |
| `/reference/channels/create`            | `client/channels.mdx`: create, channel parameters                        |
| `/reference/channels/find`              | `client/channels.mdx`: retrieve, regular expressions                     |
| `/reference/channels/rename-and-delete` | `client/channels.mdx`: rename, delete                                    |
| `/reference/channels/calculated`        | merge: `client/calculated-channels.mdx`, `concepts/channels.mdx:104-140` |

**Hardware**

| New path                              | Source                                                                   |
| ------------------------------------- | ------------------------------------------------------------------------ |
| `/reference/hardware/`                | `driver/get-started.mdx`                                                 |
| `/reference/hardware/tasks`           | `driver/task-basics.mdx`                                                 |
| `/reference/hardware/<vendor>/<page>` | `driver/<vendor>/<page>` for ethercat, http, labjack, modbus, ni, opc-ua |
| `/reference/hardware/custom`          | `client/advanced/build-device-driver.mdx` (Enterprise)                   |

**Visualize**

| New path                          | Source                   |
| --------------------------------- | ------------------------ |
| `/reference/visualize/projects`   | `console/projects.mdx`   |
| `/reference/visualize/line-plots` | `console/line-plots.mdx` |
| `/reference/visualize/schematics` | `console/schematics.mdx` |
| `/reference/visualize/tables`     | `console/tables.mdx`     |
| `/reference/visualize/logs`       | `console/logs.mdx`       |

**Ranges**

| New path                     | Source                                                                     |
| ---------------------------- | -------------------------------------------------------------------------- |
| `/reference/ranges/`         | merge: `concepts/ranges.mdx`, `concepts/overview.mdx:99-120`               |
| `/reference/ranges/create`   | `client/ranges.mdx`: create, child ranges, range parameters                |
| `/reference/ranges/find`     | `client/ranges.mdx`: explorer, favorites, active range, overview, retrieve |
| `/reference/ranges/edit`     | `client/ranges.mdx`: update, delete                                        |
| `/reference/ranges/metadata` | `client/ranges.mdx`: metadata, labels                                      |
| `/reference/ranges/aliases`  | `client/channels.mdx` and `client/ranges.mdx`: set and read aliases        |
| `/reference/ranges/data`     | `client/ranges.mdx`: read from and write to a range                        |
| `/reference/ranges/export`   | NEW: CSV from a plot selection and from a range                            |

**Control**

| New path                         | Source                                                                        |
| -------------------------------- | ----------------------------------------------------------------------------- |
| `/reference/control/get-started` | `control/get-started.mdx`, Python marked Enterprise                           |
| `/reference/control/manual`      | NEW: control mode and control chips, from `console/schematics.mdx:91-105`     |
| `/reference/control/authority`   | merge: `concepts/writes.mdx:478-509`, `client/advanced/write-authorities.mdx` |
| `/reference/control/arc/*`       | unchanged, except `how-to/alarms` moves to Alarms                             |
| `/reference/control/python/*`    | unchanged (Enterprise)                                                        |

**Alarms**

| New path                      | Source                                                              |
| ----------------------------- | ------------------------------------------------------------------- |
| `/reference/alarms/`          | NEW: statuses and the status explorer                               |
| `/reference/alarms/redlines`  | merge: `console/tables.mdx:62-76`, `console/schematics.mdx:287-300` |
| `/reference/alarms/arc`       | `control/arc/how-to/alarms.mdx`                                     |
| `/reference/alarms/pagerduty` | `driver/pagerduty/alert-task.mdx`                                   |

**Client libraries** keep their paths: `quick-start`, `authentication`,
`series-and-frames`, `time-types`, `stream-data`, `read-data`, `write-data`,
`advanced/auto-commit`, `advanced/auto-index`, `advanced/iterators`,
`advanced/delete-data`, `examples`, and `troubleshooting`. `stream-data` absorbs
`concepts/streams.mdx`, and `read-data` absorbs `concepts/reads.mdx`. `write-data`
absorbs `concepts/writes.mdx` except its authority part, and it carries the `#rule-1`
and `#the-rules-of-writes` anchors that `cesium/writer_stream.go:775,812` prints.
`channels`, `calculated-channels`, `ranges`, and `advanced/write-authorities` move to
their capability sections.

**Enterprise**

| New path                                     | Source                                                                     |
| -------------------------------------------- | -------------------------------------------------------------------------- |
| `/reference/enterprise/`                     | merge: `installation.mdx`, `core/quick-start.mdx`, `core/installation.mdx` |
| `/reference/enterprise/console`              | `console/get-started.mdx` (install, connect a Core)                        |
| `/reference/enterprise/production`           | `core/production.mdx`                                                      |
| `/reference/enterprise/certificates`         | `core/installing-certificates.mdx` (an orphan today)                       |
| `/reference/enterprise/requirements`         | `core/requirements.mdx`                                                    |
| `/reference/enterprise/clusters`             | `concepts/clusters-and-nodes.mdx`                                          |
| `/reference/enterprise/users`                | `console/users.mdx`                                                        |
| `/reference/enterprise/driver/install`       | `driver/installation.mdx`: steps 1 to 3                                    |
| `/reference/enterprise/driver/configuration` | `driver/installation.mdx`: configuration methods                           |
| `/reference/enterprise/driver/cli-reference` | `driver/installation.mdx`: CLI reference                                   |
| `/reference/enterprise/cli-reference`        | `core/cli-reference.mdx`                                                   |
| `/reference/enterprise/systemd-service`      | `core/systemd-service.mdx`                                                 |

`concepts/overview.mdx` is split into the channel, range, and client pages above and is
deleted. `reference/installation.mdx` redirects to `/reference/enterprise/` until the
Desktop release, then to `/reference/get-started/install-desktop`.

### 5.2 Edition markers

`PageNavNode` (`components/nav/Page.tsx:37-44`) gets an `edition` field beside `flag`,
with the same inheritance: a page takes the nearest `edition` on its path through the
nav. The Client libraries and Enterprise sections set it once, and their pages carry
none, which follows GitLab's rule. The `Reference.astro` layout finds the page in
`REFERENCE_PAGES` and shows the badge under the title, so a reader who comes from search
still sees it. Front matter gets no edition field, per principle 4.

Two more marker forms, in a new `components/edition/` package beside
`components/platform/` and `components/client/`:

- **Tab marker**: the `Info` entries in `CLIENTS` (`components/client/Client.tsx:20-38`)
  get an `edition`. The Python, TypeScript, and C++ tabs show an Enterprise marker on
  every page, from one list.
- **Inline block**: `<Edition.Enterprise>` wraps a sentence or paragraph in a shared
  page, built on the Lyra `Note`. The first user is
  `components/console/ShareAndExport.astro`, whose copy link text applies only to
  Enterprise.

Desktop-only pages (`install-desktop` and the `desktop/` group) carry
`edition: "desktop"` and show a Desktop badge.

### 5.3 The client switcher remembers a choice

`Tabs` (`components/tabs/Tabs.tsx:64-88`) keeps a choice only in the `?client=` query
parameter, so the next page opens on the Console tab again. The switcher now writes the
last choice to `localStorage` and reads it when the URL has no parameter, with each
access in a `try`/`catch` because storage can be blocked. This removes the cost that
decision 8.2 accepts. The URL parameter still wins, so a shared link opens on the tab it
names. Precedent: `components/nav/Page.astro:22` keeps the sidebar scroll position in
`sessionStorage`.

### 5.4 Nav mechanics

- **Previous and next from the nav**: `Reference.astro:21-41` reads `next`, `nextURL`,
  `prev`, and `prevURL` from front matter today, in 181 hand-written lines across about
  91 files. The layout computes them from the depth-first order of `REFERENCE_PAGES`
  instead, skipping nodes with no `href` and nodes whose flag is off, and the front
  matter lines are deleted.
- **Orphan check**: `src/util/checks` gets a check that every routed page under
  `reference/` has a nav node, and that every nav `href` names a page. It fails the
  build in CI (`test.docs.yaml`).
- **Section icons**: `SECTION_ICONS` (`Page.tsx:27-35`) is keyed by section key. It gets
  the new keys and loses the old ones. The ignored `icon` string field stays out of
  scope.
- **Search**: the icon map (`components/search/Search.tsx:67-85`) and the empty-state
  suggestions (`Search.tsx:182` and after) move to the new sections. The index is
  rebuilt on each deploy and needs no migration.
- **Breadcrumb**: the casing map in `components/article/Breadcrumb.tsx:22-36` gets the
  new path segments.

### 5.5 Redirects

Every moved page gets an exact entry in the `redirects` map of `astro.config.ts`, which
holds 57 entries today. Moved pages are never kept as stubs. Existing redirects whose
target moves are pointed at the final page, so no redirect chains.

`vercel.json` gets no `/reference/driver/:slug*` wildcard, because `installation` and
`pagerduty` go to other sections; each driver page gets an exact entry instead. The old
`/reference/device-drivers/:slug*` and `/reference/cluster/:slug*` wildcards keep their
`driver` and `core` targets. A wildcard cannot rename slugs (`quick-start` becomes the
Enterprise root), so those old URLs take a second hop through the exact entries.

Three URLs that shipped binaries print are already dead, and they get redirects too:
`/reference/client/resources/troubleshooting` (`client/py/synnax/connection.py:77`,
`client/ts/src/connection/client.ts:492`, `client/cpp/connection/checker.cpp:22`),
`/reference/client/working-with-data/streaming-data` (the Python and TypeScript streamer
docstrings), and `/concepts/write` (`client/cpp/framer/framer.h:269`).

### 5.6 Links in products and other sites

Redirects keep old binaries working. The sources still move to the final URLs:

- **Console**: the docs button and palette command
  (`console/src/platform/docs/useOpen.ts:15`) opens `/reference/`. The Core notification
  (`console/src/feature/core/notifications.tsx:27`) opens `/reference/enterprise/`.
  `integration/console/docs.py:16` follows.
- **Core and Cesium**: `cesium/writer_stream.go:775,812` print
  `/reference/client/write-data#rule-1` and `#the-rules-of-writes`.
- **Driver**: `driver/daemon/daemon_linux.cpp:33` writes
  `/reference/enterprise/driver/install` into the systemd unit.
- **Clients**: the dead URLs of §5.5, and the `concepts/*` docstring links in the Python
  and TypeScript clients.
- **Other sites and files**: `site/www` (hero, automate, visualize, and review blocks),
  the root, `core`, and client READMEs, and the client example headers.

The updater endpoints (`/releases/<product>/<channel>.json`, read by
`tauri.conf.json:36` and `tauri.desktop.conf.json:19`) are not docs pages and do not
move.

### 5.7 Existing mechanisms this design uses

| Mechanism                        | Where                                     |
| -------------------------------- | ----------------------------------------- |
| Nav tree and flag filter         | `src/pages/_nav.ts:19-39`                 |
| `PageNavNode` type               | `src/components/nav/Page.tsx:37-44`       |
| Flag registry and `enabled`      | `src/flags.ts:23,33`                      |
| 404 for a page whose flag is off | `src/layouts/Root.astro:22-24`            |
| Query-synced tabs                | `src/components/tabs/Tabs.tsx:33-108`     |
| Client tab list                  | `src/components/client/Client.tsx:20-38`  |
| Right-rail switcher              | `src/components/nav/OnThisPage.tsx:40-48` |
| Release lookup with `desktop`    | `src/util/releases.ts:13-23`              |
| Site checks                      | `src/util/checks/main.ts`, `links.ts`     |

NEW: the `edition` field on `PageNavNode`, the `components/edition/` package, the
switcher's stored choice, computed previous and next links, and the orphan check. Each
names its precedent in the sections above.

### 5.8 Desktop timing

Desktop is not released. No flag hides the Desktop pages: the new sections land with the
first Desktop release.

## 6 What this RFC does not cover

- **Licensing and Desktop sign-in pages**: they document RFC 0062, which is not on
  `main`. They land with that work, in the Enterprise and Get started sections.
- **Release notes**: the releases page and a Desktop release note list, which follow
  Desktop's own version numbers (RFC 0063).
- **New content beyond the gaps named in §5.1**: channel groups, labels, workspace
  layout, and access policies need pages, but each is a content task inside a section
  that this RFC already defines.
- **Docs versioning**: the site documents the latest release only, as today.

## 7 Implementation phases

Each phase is one pull request into `main`. The work was built on one branch, and each
pull request copies its files from that branch.

- **Phase 1: Lyra.** Chip tabs, and one hover for the pointer and the arrow keys.
- **Phase 2: Studio.** The capture scripts for the docs videos.
- **Phase 3: Groundwork.** Computed previous and next links, the orphan check, edition
  markers, the client tab changes, the design system, search, the media lightbox, and
  the 404 pages. No page moves.
- **Phase 4: Sections.** Every section of §5.1, the landing page, the redirects of §5.5,
  and the links of §5.6. Ships with the first Desktop release.

Compatibility: every URL that exists on the live site today, and every URL a shipped
binary prints, resolves to a page after each phase. The anchors that Cesium prints
survive. The updater endpoints do not change.

## 8 Resolved decisions

**8.0 One tree for both editions.** Each capability page is written once, with edition
markers. Rejected: a tree per edition built from shared source (InfluxDB 3). It copies
about 70 shared pages into two URL sets, splits search results, and needs build tooling,
to cover a difference set of about 30 pages and a few tabs. The trade is real: an
Enterprise reader sees badges they never need, and a Desktop reader sees Python tabs
they cannot use.

**8.1 Capability sections, Concepts dissolved.** Each section opens with its own concept
page, and clusters and nodes move to Enterprise. Rejected: keeping Concepts as a
section. Its reads, writes, and streams pages are client API docs, and its other pages
duplicate the capability pages. Alarms is its own section although Arc supplies most of
its content, because an operator looks for alarms, not for Arc.

**8.2 Desktop first.** Get started leads with the Desktop download, and shared pages
lead with the Console. Desktop is the only free edition, and staff issue every
Enterprise license (RFC 0062), so an anonymous reader is a Desktop reader. The cost to
an Enterprise reader, one tab choice, is paid once through §5.3.

**8.3 Previous and next from the nav.** Rejected: rewriting the 181 hand-written lines
during the moves. No check keeps them in line with the sidebar, and each later move
would repeat the work. The trade: a page cannot link to a next page outside the sidebar
order. No current page does.

**8.4 Edition on the nav node, not in front matter.** One place states order and
edition, and a section states it once for all its pages. Rejected: an `edition` front
matter field, which each page in two sections would repeat.

**8.5 Client libraries and Arc keep their paths.** Only the pages that change section
move. This keeps about 40 client example headers and the Arc links in `site/www` valid
without edits.

## 9 Open questions

1. **First run device**: the tutorial needs a signal. Arc has `time.interval` but no
   signal generator, so a no-hardware path needs a counter or a ramp into a virtual
   channel. A LabJack or NI path needs a device.
2. **Badge text and look**: "Enterprise" and "Desktop", or longer text such as
   "Enterprise only".
3. **Docs button per build**: Desktop and the Console could open different start pages
   through the `DESKTOP` define.
4. **Log in or sign in**: the Desktop and portal copy uses both. The docs pick one.
5. **Linux**: Desktop ships for macOS arm64 and Windows x64. The install page says so,
   and says where a Linux user goes.
