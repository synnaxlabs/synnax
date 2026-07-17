# Slack Status Integration

## 0 - Summary

This RFC specifies a **Slack status integration**: Synnax posts a message to a Slack
channel whenever a watched status changes. It follows the existing PagerDuty integration
(`core/pkg/service/pagerduty/`) closely, with one structural difference that follows
from how the two APIs differ: Slack requires authenticating a connection to a
**workspace**, so that connection is modeled as a reusable **device** (like the HTTP
integration), while each task picks a channel and statuses within it.

Two user-facing objects:

- A **Slack device** (`make: "slack"`) holds the bot token for one workspace. Connected
  once, reused by many tasks.
- A **`slack_alert` task** references a device, targets a single Slack channel, and
  holds a list of statuses to watch. When any watched status changes, the task posts to
  that channel. Fanning one status out to several channels means several tasks.

Integration name: `slack` (task type: `slack_alert`).

## 1 - Motivation

Synnax statuses (`success`/`info`/`warning`/`error`/`loading`/`disabled`) report the
health of tasks, racks, calculations, Arc programs, and custom events. An operator
running a test is usually at the Console already; the value of Slack is reaching
everyone who isn't — a manager, a remote teammate, an on-call engineer — in a channel
they already watch. SY-3938 scoped this alongside a PagerDuty integration as the "status
integrations" family; PagerDuty shipped, Slack is the sibling.

Slack and PagerDuty differ in a way that dictates the shape. PagerDuty's `routing_key`
(`core/pkg/service/pagerduty/alert_task.go:54-62`) is not a connection — it is an event
key posted to a fixed public Events API endpoint, identifying a destination rather than
authenticating one, so it lives naturally on each task. Slack's bot token authenticates
the app to a specific **workspace**, and many channels live inside that one
authenticated connection. That is a genuine connection to establish once and reuse —
exactly what a device is for. So the workspace connection is a device, and a task
selects a channel and the statuses to post there.

## 2 - Vocabulary

- **Slack device** — a logical `device.Device` with `make: "slack"` whose `properties`
  hold the bot token. No hardware; a logical connection, exactly as the HTTP "device"
  (`make: "http"`) is (`console/src/feature/http/device/types.ts:14-16`).
- **Watched status** — a status key listed on a `slack_alert` task. A change to any
  watched status posts to the task's channel.
- **Sender** — the injected seam that performs the outbound Slack call, mirroring
  PagerDuty's `EventSender` (`core/pkg/service/pagerduty/event_sender.go:19-32`).

## 3 - Principles

- **Conform to the PagerDuty precedent.** PagerDuty is a proven, reviewed integration
  spanning Core + Console. Slack follows its factory/task/sender shape, and every
  divergence is called out and justified here. (Root CLAUDE.md: "absence of a pattern is
  a principle.")
- **Model the connection where the API puts one.** Slack authenticates at the workspace
  level, so the workspace connection is a device and the per-task fields (channel,
  watched statuses) sit on the task. This is the HTTP device↔task split
  (`console/src/feature/http/task/types.ts:254`, `device: z.string()`). PagerDuty has no
  such connection, so its token stays on the task — same principle, different API.
- **Deep module, dumb transport.** The `slack_alert` task owns the decision (match
  status, build message); the sender is a thin seam over `chat.postMessage`. Tests
  construct the real task with a fake sender — no mocking our own modules
  (`feedback_mock_at_external_boundary`).
- **Reactive integrations belong in Core, not on edge Drivers.** Slack lives in Core as
  a Go embedded-Driver factory (alongside PagerDuty and Arc), not on a C++ edge Driver.
  An edge Driver exists for data acquisition and control at the hardware edge; reacting
  to status changes and notifying an external service is a Core concern. A C++ Driver
  _could_ subscribe to the `sy_status_set` channel, but that is the wrong home. Living
  in Core also gives the factory direct in-process access to `status.Service.Observe()`
  (`core/pkg/service/status/service.go:130`) and keeps the match/format logic in one
  place rather than replicated across clients.

## 4 - Design

### Placement

Slack is a **Go embedded-Driver factory** that runs inside Core, like PagerDuty and Arc
— _not_ a C++ edge Driver integration like HTTP. The reason is role, not capability:
reacting to status changes and posting to an external service is a Core concern, while
edge Drivers exist for data acquisition and control. A C++ Driver could technically
stream the `sy_status_set` channel, but Core is the right home. Running in Core also
lets the factory read `status.Service.Observe()` directly, in-process. Credentials still
come from a device, like HTTP — so Slack is a new combination: PagerDuty's Core-side
task shape plus HTTP's device-credential resolution.

```
core/pkg/service/slack/            (NEW — models core/pkg/service/pagerduty/)
├── factory.go        # NewFactory → driver.Factory; ConfigureTask; Name() == "slack"
├── alert_task.go     # slack_alert task: observe statuses, match, format, send
├── sender.go         # Sender interface + default chat.postMessage impl (seam)
├── message.go        # Block Kit message builder (variant → color/emoji)
├── config.go         # FactoryConfig{ Status, Device, Sender } + Validate/Override
└── *_test.go         # constructed with a fake Sender

console/src/feature/slack/         (NEW — models console/src/feature/pagerduty/)
├── device/
│   ├── types.ts          # make "slack", properties { botToken }, zod schemas
│   ├── useConnectModal.tsx  # connect modal; beforeSave runs Slack auth.test
│   └── commands.tsx      # "Connect a Slack workspace"
└── task/
    ├── types.ts          # slackAlertTaskConfigZ { device, channel, statuses[] }, ZERO_*
    ├── Alert.tsx         # Task.wrapForm: device select + channel + watched-status list
    └── commands.tsx      # "Create a Slack Alert Task"
```

### Data shapes

**Slack device** — generic `device.Device`
(`core/pkg/service/device/types.gen.go:41-65`; schema
`schemas/synnax/device.oracle:42-115`), no new Oracle schema needed (Device is generic
over `Properties`, stored as an opaque JSON blob):

```ts
// console/src/feature/slack/device/types.ts (NEW)
make  = "slack"
model = "Slack workspace"
properties = { botToken: string, team?: string, version: number }
// location = workspace/team id (or placeholder), mirroring how HTTP puts host in location
```

**`slack_alert` task config** (parsed by both the Go factory and the Console form):

```ts
// console/src/feature/slack/task/types.ts (NEW)
slackAlertTaskConfigZ = {
  device:   string,    // device key
  channel:  string,    // channel name or id
  statuses: string[],  // watched status keys
}
```

Go structs mirror these in `config.go`/`alert_task.go`, parsed from the task's
`Config msgpack.EncodedJSON` (`core/pkg/service/task/types.gen.go:50-51`) — the same
mechanism PagerDuty uses.

### Runtime flow

1. Factory built at `core/pkg/service/layer.go:~576` alongside the PagerDuty block, with
   `FactoryConfig{ Status: l.Status, Device: l.Device }` (`l.Device` at `layer.go:174`),
   appended to the `Factories` slice at `layer.go:591`. `Name()` returns `"slack"`,
   which auto-advertises the rack integration via `driver.go:69-72`.
2. On `ConfigureTask`, the task decodes its config, retrieves the referenced device via
   `Device.NewRetrieve().WhereKeys(cfg.device)`, and reads the bot token from
   `properties` — the Go analog of HTTP's `retrieve_connection`
   (`driver/http/device/device.cpp:88-103`).
3. `start()` subscribes with `factoryCfg.Status.Observe().OnChange(handleStatusChange)`
   (mirrors `pagerduty/alert_task.go:125`).
4. `handleStatusChange` ranges the `gorp.TxReader`, skips `change.VariantDelete`, and
   for each set whose key is in the task's `statuses`, builds a Block Kit message and
   calls `Sender.Post(ctx, cfg.channel, message)`.
5. The task reports its own health as a task status (bad token, Slack API error → the
   task goes `error`), so failures surface in the Console rather than vanishing.

### Message format (D6)

Block Kit message with a severity-colored attachment:

| Variant  | Color  | Emoji |
| -------- | ------ | ----- |
| success  | green  | 🟢    |
| info     | blue   | 🔵    |
| warning  | orange | 🟠    |
| error    | red    | 🔴    |
| loading  | grey   | ⏳    |
| disabled | grey   | ⚪    |

Headline = status `name`; body = `message`; context line = `description` + `time`. No
deep link back into Synnax in v1 (the desktop Console has no stable URL scheme to
target).

### Slack API

Outbound calls go through the `Sender` seam. Default impl calls `chat.postMessage` with
the device's bot token; the connect modal validates the token via `auth.test`. Client
library is a parameter (see Open Questions) — `github.com/slack-go/slack` mirrors
PagerDuty's SDK choice; a raw `net/http` POST is a dependency-free alternative. Either
way the seam is identical.

## 5 - Implementation Phases

**Phase 1 — Core (Go).** The `slack` package: factory, `slack_alert` task, `Sender`
seam + default impl, Block Kit message builder, device-credential resolution, and wiring
into `layer.go`. Unit-tested by constructing the real task with a fake `Sender` and
driving `status.Service` sets. **Green intermediate state:** a Slack device +
`slack_alert` task created through the client posts to Slack end-to-end, with no Console
UI yet. Risk-isolated to the backend; reviewable on its own.

**Phase 2 — Console (TS).** `feature/slack/device/` connect modal (`auth.test` in
`beforeSave`, mirroring HTTP's connection test at
`console/src/feature/http/device/useConnectModal.tsx:67-109`) and `feature/slack/task/`
alert form (`Task.wrapForm` with a device select + channel + watched-status list,
mirroring `console/src/feature/pagerduty/task/Alert.tsx`), plus palette commands and
registration into `console/src/feature/task/external.tsx` / `Selector.tsx` /
`layouts.ts` / `types.tsx` exactly as PagerDuty is wired. Depends on Phase 1's task type
existing.

Two phases only: the split isolates backend risk from UI and gives a green,
client-testable state between them. Finer splits (e.g. device plumbing vs task) buy no
reviewability or greenness and are merged.

## 6 - Resolved Decisions

- **D1 — Device + tasks, not task-only.** The workspace connection lives on a reusable
  `slack` device; tasks reference it by key. This is not a judgment that PagerDuty's
  inline-token model is worse — the two APIs differ: a PagerDuty routing key is an event
  destination with no connection to authenticate, so it belongs on the task, whereas a
  Slack bot token authenticates one workspace that many channels and tasks share.
  Rejected: copying PagerDuty's inline token onto each Slack task — it would
  re-authenticate the same workspace per task and re-enter the secret each time. The
  trade is real: the device model adds a device type and a retrieve step.
- **D2 — Specific statuses only.** A task watches explicitly-chosen status keys and
  posts them all to its one channel. Rejected for v1: variant/label filters ("all
  errors", "all `flight-critical`"). The trade: dynamically-keyed statuses
  (runtime-minted task/rack statuses) can't be caught by a blanket rule yet. Deferred to
  Open Questions.
- **D3 — Bot token + `chat.postMessage`.** Rejected: incoming webhooks — one webhook is
  bound to one channel, incompatible with one device fanning out to many task channels
  (D1). The trade: users must create/install a Slack app with `chat:write` and invite
  the bot per channel.
- **D4 — Stateless, fresh message per change.** Rejected for v1:
  update-in-place/threading, which would require persisting a status-key → Slack-`ts`
  map surviving task restarts. Additive to the sender later; not a rework.
- **D5 — Post on all six variants.** No variant filtering in v1. The trade: `loading` is
  transient, so `loading → success` posts twice and busy tasks can be chatty. A per-task
  variant filter is the escape hatch if it bites.
- **D6 — Block Kit, colored attachment, no deep link.** See table above.
- **D7 — Plaintext token on the device.** Matches HTTP bearer tokens
  (`console/src/feature/http/device/types.ts:63-68`) and the PagerDuty routing key.
  Rejected for v1: encrypted-at-rest storage — no secret store, vault, or keyring exists
  in Core; building one is new cross-cutting infra far beyond this ticket, and
  encrypting only the Slack token while neighbors sit in plaintext is incoherent. The
  trade is real and acknowledged: the token is readable in the DB. If encryption is
  wanted, do it once across all integrations.

## 7 - Open Questions

- **Slack Go client library** — `github.com/slack-go/slack` (mirrors PagerDuty's SDK
  choice) vs a dependency-free `net/http` POST. Parameter; the `Sender` seam is
  identical either way.
- **Variant/label filter** (deferred from D2) — a future rule kind matching on variant
  and/or label, for "post every error" without enumerating status keys.
- **Per-task variant filter** (escape hatch for D5) — if all-variants posting proves
  noisy.
- **Update-in-place / threading** (deferred from D4) — edit or thread a status's message
  on subsequent changes; needs a status-key → `ts` store.
- **Deep link back to Synnax** — pending a stable Console URL scheme.
- **Cross-cutting secret encryption** (deferred from D7) — one mechanism for all
  integration credentials.

## 8 - What This RFC Does Not Cover

- Encrypted credential storage (D7 — future cross-cutting effort).
- Inbound Slack (slash commands, interactivity, acknowledging alerts from Slack).
- Slack as an Arc action node (composable side-effect in automation graphs) — a
  distinct, larger feature; this RFC is the standalone-integration path.
- PagerDuty changes — Slack is additive and does not touch the PagerDuty package.
