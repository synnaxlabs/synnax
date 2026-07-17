# Slack Status Integration

## 0 - Summary

Synnax posts a message to a Slack channel whenever a watched status changes. It follows
the PagerDuty integration (`core/pkg/service/pagerduty/`), with one difference: Slack
authenticates a **workspace**, so that connection is a reusable **device** (like HTTP)
rather than a per-task credential.

- A **Slack device** (`make: "slack"`) holds the bot token for one workspace.
- A **`slack_alert` task** references a device, targets one channel, and lists the
  statuses to watch. A change to any watched status posts to that channel. Fan-out to
  several channels = several tasks.

Integration name: `slack` (task type: `slack_alert`).

## 1 - Motivation

Synnax statuses (`success`/`info`/`warning`/`error`/`loading`/`disabled`) report the
health of tasks, racks, calculations, Arc programs, and custom events. The operator
running a test is usually at the Console already; Slack's value is reaching everyone who
isn't — a manager, a remote teammate, an on-call engineer. SY-3938 scoped this alongside
PagerDuty as the "status integrations" family; PagerDuty shipped, Slack is the sibling.

The API difference drives the shape. PagerDuty's `routing_key`
(`core/pkg/service/pagerduty/alert_task.go:54-62`) is not a connection — it is an event
key posted to a fixed endpoint, so it lives on each task. Slack's bot token
authenticates one **workspace** that many channels share — a connection to establish
once and reuse, which is what a device is for.

## 2 - Vocabulary

- **Slack device** — a logical `device.Device` (`make: "slack"`) whose `properties` hold
  the bot token, exactly as the HTTP "device" is a logical connection
  (`console/src/feature/http/device/types.ts:14-16`).
- **Watched status** — a status key on a `slack_alert` task; a change to it posts.
- **Sender** — the injected seam for the outbound Slack call, mirroring PagerDuty's
  `EventSender` (`core/pkg/service/pagerduty/event_sender.go:19-32`).

## 3 - Principles

- **Conform to the PagerDuty precedent.** Slack follows its factory/task/sender shape;
  divergences are justified below.
- **Model the connection where the API puts one.** Slack authenticates at the workspace,
  so that connection is a device and per-task fields (channel, statuses) sit on the task
  — the HTTP device↔task split (`console/src/feature/http/task/types.ts:254`). PagerDuty
  has no connection, so its token stays on the task: same principle, different API.
- **Deep module, dumb transport.** The task owns the decision (match, format); the
  sender is a thin seam over `chat.postMessage`. Tests use a fake sender, not mocks of
  our own modules.
- **Reactive integrations belong in Core, not edge Drivers.** Reacting to statuses and
  notifying externally is a Core concern; edge Drivers do data acquisition and control.
  A C++ Driver could subscribe to `sy_status_set`, but Core is the right home — and gets
  in-process `status.Service.Observe()` (`core/pkg/service/status/service.go:130`).

## 4 - Design

### Placement

Slack is a **Go embedded-Driver factory** inside Core, like PagerDuty and Arc — not a
C++ edge Driver like HTTP (see the Core-vs-Driver principle above). It is a new
combination: PagerDuty's Core-side task shape plus HTTP's device-credential resolution.

```
core/pkg/service/slack/            (NEW — models core/pkg/service/pagerduty/)
├── factory.go        # NewFactory → driver.Factory; ConfigureTask; Name() == "slack"
├── alert_task.go     # slack_alert task: observe statuses, match, format, send
├── sender.go         # Sender interface + default chat.postMessage impl (seam)
├── message.go        # Block Kit message builder (variant → color/emoji)
├── config.go         # FactoryConfig{ Status, Device, Sender } + Validate/Override
└── *_test.go         # constructed with a fake Sender

console/src/feature/slack/         (NEW — models console/src/feature/pagerduty/)
├── device/           # types.ts, useConnectModal.tsx (auth.test), commands.tsx
└── task/             # types.ts, Alert.tsx (Task.wrapForm), commands.tsx
```

### Data shapes

The Slack device is a generic `device.Device`
(`core/pkg/service/device/types.gen.go:41-65`) — no new Oracle schema, since
`Properties` is an opaque JSON blob. Task config is parsed by both the Go factory and
the Console form.

```ts
// device properties
{ botToken: string, team?: string, version: number }   // make "slack", model "Slack workspace"

// slack_alert task config
{ device: string, channel: string, statuses: string[] }
```

Go structs mirror these, decoded from the task's `Config msgpack.EncodedJSON`
(`core/pkg/service/task/types.gen.go:50-51`) — the mechanism PagerDuty uses.

### Runtime flow

1. Factory built at `core/pkg/service/layer.go:~576` with
   `FactoryConfig{ Status: l.Status, Device: l.Device }`, appended to the `Factories`
   slice at `layer.go:591`. `Name()` returns `"slack"`, which auto-advertises the rack
   integration (`driver.go:69-72`).
2. `ConfigureTask` decodes the config, retrieves the device by key, and reads the token
   from `properties` — the Go analog of HTTP's `retrieve_connection`
   (`driver/http/device/device.cpp:88-103`).
3. `start()` subscribes via `Status.Observe().OnChange` (mirrors
   `pagerduty/alert_task.go:125`); the handler skips deletes, matches keys against
   `statuses`, builds a message, and calls `Sender.Post(ctx, channel, message)`.
4. On failure (bad token, Slack error) the task sets its own status to `error`,
   surfacing in the Console.

### Message format (D6)

Block Kit message with a severity-colored attachment (green success 🟢, blue info 🔵,
orange warning 🟠, red error 🔴, grey loading ⏳ / disabled ⚪). Headline = `name`, body
= `message`, context = `description` + `time`. No deep link in v1.

### Slack API

The default `Sender` calls `chat.postMessage` with the device token; the connect modal
validates via `auth.test`. Client library is a parameter (Open Questions) — the seam is
identical either way.

## 5 - Implementation Phases

**Phase 1 — Core (Go).** The `slack` package (factory, task, sender + default impl,
message builder, device resolution) and `layer.go` wiring. Unit-tested with a fake
`Sender`. Green state: a device + task created via the client posts end-to-end, no UI
yet.

**Phase 2 — Console (TS).** `feature/slack/device/` connect modal (`auth.test` in
`beforeSave`, per `console/src/feature/http/device/useConnectModal.tsx:67-109`) and
`feature/slack/task/` form (`Task.wrapForm`, per
`console/src/feature/pagerduty/task/Alert.tsx`), plus palette commands and registration
in `console/src/feature/task/external.tsx` as PagerDuty is wired. Depends on Phase 1.

Two phases: the split isolates backend risk from UI with a green state between. Finer
splits buy no reviewability and are merged.

## 6 - Resolved Decisions

- **D1 — Device + tasks, not task-only.** Connection on a reusable device; tasks
  reference it by key. Not a knock on PagerDuty's inline token — its routing key has no
  connection to authenticate, while a Slack token authenticates a workspace many tasks
  share. Inlining the token per task would re-auth the same workspace and re-enter the
  secret each time. Trade: adds a device type and a retrieve step.
- **D2 — Specific statuses only.** A task watches explicitly-chosen keys, all to one
  channel. Deferred: variant/label filters ("all errors") — dynamically-keyed statuses
  can't be caught by a blanket rule yet (Open Questions).
- **D3 — Bot token + `chat.postMessage`.** Rejected webhooks: one webhook = one channel,
  incompatible with a device fanning out to many channels. Trade: a Slack app with
  `chat:write`, invited per channel.
- **D4 — Stateless, fresh message per change.** Deferred update-in-place/threading —
  needs a status-key → `ts` map surviving restarts. Additive later.
- **D5 — Post on all six variants.** No filtering in v1. Trade: `loading` is transient,
  so it can be chatty; a per-task variant filter is the escape hatch.
- **D6 — Block Kit, colored attachment, no deep link.**
- **D7 — Plaintext token on the device.** Matches HTTP tokens and the PagerDuty key.
  Rejected encryption-at-rest: no secret store exists, and encrypting only Slack while
  neighbors sit in plaintext is incoherent. Trade: the token is readable in the DB; do
  encryption once across all integrations if wanted.

## 7 - Open Questions

- **Slack Go client library** — `github.com/slack-go/slack` vs a raw `net/http` POST.
- **Variant/label filter** (from D2) — match on variant/label instead of enumerating
  keys.
- **Per-task variant filter** (from D5) — if all-variants posting proves noisy.
- **Update-in-place / threading** (from D4) — needs a status-key → `ts` store.
- **Deep link to Synnax** — pending a stable Console URL scheme.
- **Cross-cutting secret encryption** (from D7) — one mechanism for all credentials.

## 8 - What This RFC Does Not Cover

- Encrypted credential storage (D7).
- Inbound Slack (slash commands, acknowledging alerts).
- Slack as an Arc action node — a distinct, larger feature.
- PagerDuty changes — Slack is additive.
