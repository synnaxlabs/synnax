# 62 Portal and licensing

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-17
- **Related**: [RFC 0011 - Alamos instrumentation](0011-alamos-instrumentation.md),
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md),
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md)

## 0 Summary

Synnax has no account. A user downloads a public binary, and an enterprise customer
pastes a key that encodes an expiry and a channel count with arithmetic anyone can
reverse (`core/pkg/service/channel/verification/verification.go:28-110`). Nothing
connects a person to a machine or a machine to a license.

This RFC turns docs.synnaxlabs.com into that connection. The site moves from `docs/site`
to a top-level `hub/`, gains accounts through Clerk, and organizations and licenses in
Neon Postgres, all inside the existing Astro server deployment on Vercel, with every
cloud resource declared in Terraform. The Core gains one license primitive: a JWT signed
with Ed25519 and bound to a machine, verified offline with public keys compiled into the
binary. Two paths issue that token. The free edition, Synnax Desktop, signs the user in
through the system browser and issues itself a short-lived license that renews while
signed in. The enterprise edition, the standalone Core, activates through a start flag
or the Console against a license that staff issued in the portal, on a subscription or
perpetual term. Downloads stay public, the Core never phones home, a running Core never
stops because of time, and the old key format is deleted.

## 1 Motivation

- **The current key has no integrity.** `parse` decodes the date and channel count with
  `crypto.Cipher` and checks a self-consistency checksum
  (`verification/verification.go:88-110`). Anyone who reads the package mints keys.
- **Nothing surfaces license state.** The Console and clients have no license UI. An
  over-cap Core fails channel creation (`core/pkg/service/channel/writer.go:489-492`)
  and channel retrieval (`core/pkg/service/channel/service.go:230`) with a plain string
  error that no client recognizes.
- **The free edition is changing shape.** Synnax Desktop, a Console build with an
  embedded Core, becomes the free product. It needs an account to link a machine, so the
  keyless 50-channel tier has no product left to serve.
- **A perpetual deal already exists** and the current key cannot express it: the key has
  an expiry and nothing else.
- **Installs cannot be counted.** Download anchors fire no event, and GitHub asset
  counters are the only signal.

## 2 Vocabulary

- **Hub**: The Astro site at `hub/`, formerly `docs/site`: the docs, blog, releases, and
  the portal. One deployment on Vercel.
- **Portal**: The signed-in surface of the hub: account, organizations, and licenses. It
  is the same Astro site, not a second deployment.
- **Organization**: The owner of every license. A row in the portal's own table. A
  personal organization exists for every user; a team organization is additionally
  backed by a Clerk organization for membership, invitations, and roles.
- **Edition**: `desktop` or `enterprise`. Desktop is the feature-flagged Console build
  with an embedded Core. Enterprise is the standalone Core, Console, and Driver.
- **License**: A signed JWT naming an organization, an edition, the machine it binds to,
  a node count, a channel cap, and a term. The Core verifies it offline.
- **Term**: Either a subscription, which carries an expiry, or perpetual, which carries
  a maximum Core version instead. A subscription may also carry a maximum version as its
  fallback after expiry.
- **Fingerprint**: The set of per-interface hashes of a machine's physical network
  addresses. A license binds to a fingerprint, or to none.
- **Floating license**: A license with no fingerprint. Valid on any machine. Staff-only.
  Used by CI.
- **Activation**: The act of storing a license in a Core, and the portal's record that a
  machine holds a seat under a license.
- **Activation ledger**: The portal's list of activations per license, with release and
  reactivate.
- **Grace window**: The period after a subscription expires during which a Core still
  starts, with warnings.

## 3 Principles

1. **One license primitive**: Desktop sign-in and enterprise activation produce the same
   token, verified by the same code. There is no second mechanism.
2. **The running Core never phones home**: Verification is offline. Network calls happen
   in the Console or the portal, never in the server. This continues RFC 0011 §4.4.0 and
   RFC 0020 §1.
3. **Bypass requires editing code**: The public keys are constants in source. No flag,
   environment variable, or build tag substitutes them. Published source can always be
   patched; the bar is that a clone does not run unlicensed by accident.
4. **Time never stops a running Core**: Expiry, the grace window, and the version
   ceiling are checked when a Core starts and when a license is activated, never against
   a process that is already serving. A test operator loses a test if a license lapses
   mid-run; they lose nothing if the Core refuses to start the next morning.
5. **Enforcement lives in the license, not the download**: Every artifact stays public.
   Desktop sign-in counts free users; activation counts enterprise machines.
6. **Buy the standard parts, build the Synnax parts**: Identity, teams, email, and key
   custody come from Clerk, Resend, and AWS KMS. The license token, the fingerprint, the
   verifier, the organization model, and the activation ledger are ours.
7. **Vendor ids never enter a stored format**: The license names the portal's own
   organization key. The Clerk id is a column on that row, replaceable without reissuing
   anything.
8. **Infrastructure is code**: Every cloud resource the hub needs is declared in
   Terraform under `infra/`, one root per lifecycle. What a vendor cannot expose is a
   documented manual step, never an undocumented click.

## 4 Current mechanics

The `--license-key` flag is declared with an obfuscated name
(`core/cmd/start/flags.go:111-118`), resolves from `SYNNAX_LICENSE_KEY` through the
viper environment prefix (`core/cmd/cmd.go:57-59`), and reaches
`service.LayerConfig.Verifier` (`core/pkg/service/layer.go:96-99`).
`verification.OpenService` (`layer.go:309-313`) parses the key, stores it in the KV
under a fixed key (`verification/service.go:58`), and hands `IsOverflowed` to the
channel service as `IntOverflowCheck` (`layer.go:324`). With no key the cap is
`FreeCount = 50` (`verification/service.go:33`). A background loop logs expiry warnings
(`verification/service.go:225`). Nothing binds the key to a machine, signs it, or
exposes its state over the API.

That KV is `cfg.Distribution.DB.KV()`, the Aspen store that gossip replicates to every
node (`aspen/db.go:88`). A single fixed key therefore holds one license for the whole
cluster, which is wrong once licenses bind to machines.

The Core already signs and verifies JWTs: `token.Service` uses `golang-jwt/jwt/v5` and
picks `jwt.SigningMethodEdDSA` for an Ed25519 key
(`core/pkg/service/auth/token/token.go:187-188`). The cluster key is a UUID minted once
at bootstrap (`aspen/internal/cluster/cluster.go:122`) and returned unauthenticated by
the connectivity check (`core/pkg/api/connectivity/connectivity.go:35-44`, bound to the
insecure middleware at `core/pkg/api/layer.go:252-256`). The Core also serves the web
Console on its own listener (`core/cmd/start/start.go:290-310`), which is where an
operator without a desktop install activates.

The docs site runs Astro 7 with `output: "server"` and the Vercel adapter
(`docs/site/astro.config.ts:21-22`), imports Pluto CSS through a cascade layer
(`docs/site/src/styles/main.css`), loads PostHog
(`docs/site/src/components/analytics/PostHog.astro:65-68`), sets a Content Security
Policy in `docs/site/src/middleware.ts:12-21`, and connects its live plot to a Core at
`demo.synnaxlabs.com` (`docs/site/src/components/pluto/Plot.tsx:18`). It already depends
on `@synnaxlabs/client` (`docs/site/package.json`). The Console registers the
`synnax://` scheme (`console/src-tauri/tauri.conf.json:41`) and dispatches deep links
through a registry (`console/src/app/link/useDeep.ts`).

## 5 Design

### 5.0 Editions and ownership

Every license belongs to an organization, and the portal owns the organization record. A
user who signs up gets a personal organization whose only member is that user; no Clerk
organization is created for it. An enterprise customer gets a team organization backed
by a Clerk organization, which supplies membership, invitations, and the `admin` and
`member` roles. Staff create it in the Clerk dashboard when the deal closes and invite
the customer's admin there; customers never create teams, and a personal account shows
no organization concept at all. Admins manage the team and invite members. Moving a
license from a personal to a team organization is a transfer of one row.

An account is therefore personal or enterprise. A personal user sees their Desktop
machines and, where a team feature would sit, one Enterprise panel that names the
feature and offers Talk to us, linking to the contact form. A user in at least one team
sees that team's licenses and members, and an organization switcher only when they
belong to more than one.

Clerk prices organizations at 100 monthly retained organizations free and then one
dollar each per month. Desktop makes every free user an organization owner, so personal
organizations must not be Clerk organizations. Team organizations will sit under that
allowance for a long time.

Desktop is free and is the only free edition. The standalone Core is enterprise and
requires a license before it does anything but report its fingerprint and accept a
license (§5.3). The keyless 50-channel tier is removed.

### 5.1 The license token

A license is a JWS compact token: a JWT signed with `EdDSA` over Ed25519, the format
Grafana Enterprise ships as `license.jwt`. The Core verifies it with `golang-jwt`, which
it already depends on; the portal signs it with `jose`. Both libraries handle the
header, the base64url segments, `exp`, and `iat`, so no custom framing exists.

The header carries `alg: EdDSA` and `kid`. The Core embeds a small set of public keys by
`kid`. Rotation adds a key to the set in one release and moves signing to it; old tokens
keep verifying until they expire. A key compromise retires that `kid` in the next
release and reissues the licenses signed under it.

The claims are defined once, in `schemas/synnax/verification.oracle`, with
`@go output "core/pkg/service/channel/verification"`,
`@ts output "client/ts/src/license"`, and `@py output "client/py/synnax/license"`. The
Go type keeps a neutral name (`verification.Grant`); `@ts name` and `@py name` expose it
to clients as `License` (§5.10). The TypeScript output is both the client's type and the
portal signer's type, since the site already imports `@synnaxlabs/client`. Keys are
short in the JWT tradition, which also keeps the Core's struct tags free of license
vocabulary:

- **`jti`**: License UUID. The portal's primary key.
- **`iat`**: Issued-at, seconds since the epoch, as JWT defines it.
- **`exp`** (optional): Expiry. Absent on a perpetual license.
- **`v`**: Claim set version. Starts at `1`.
- **`org`**: The portal's organization UUID. Never a Clerk id.
- **`ed`**: Edition, `d` for desktop or `e` for enterprise.
- **`fp`**: List of per-interface hashes (§5.2). Empty for a floating license.
- **`fs`**: The fingerprint scheme. Starts at `1`.
- **`n`**: Machines that may activate under this license.
- **`ch`**: Channel cap per Core. Zero means unlimited.
- **`mv`** (optional): Maximum Core minor version, as `"0.62"`. Absent means any
  version. Required on a perpetual license. On a subscription it is the fallback that
  applies after `exp`.

The three terms fall out of `exp` and `mv`:

| Term                       | `exp`  | `mv`   | Behavior                                  |
| -------------------------- | ------ | ------ | ----------------------------------------- |
| Subscription               | set    | absent | Any version until expiry                  |
| Perpetual                  | absent | set    | Any time, up to that minor version        |
| Subscription with fallback | set    | set    | Any version until expiry, then up to `mv` |

The Go package exports `Sign(priv, kid, grant)` and `Verify(keys, token)`. `Sign` exists
for tests; the private key never lives in the repository. Production signing happens in
AWS KMS under an `ECC_NIST_EDWARDS25519` key, so the private key never exists in
plaintext anywhere. The portal calls KMS `Sign` with `MessageType: RAW` over the signing
input `jose` produces. A Vercel environment secret is the fallback if KMS proves
unworkable from the Vercel runtime, and is what tests and the CI bootstrap script use.

### 5.2 Fingerprint

Scheme 1 enumerates `net.Interfaces()`, drops loopback, point-to-point, and interfaces
without a hardware address, and hashes each remaining address with SHA-256 to lowercase
hex. The fingerprint is that set. The license carries the set observed at issuance, and
the verifier accepts the token if the intersection with the current set is non-empty.
This is FlexNet's "any listed host id present" rule, and it means a USB network adapter
or a swapped Wi-Fi card does not invalidate a license. A machine whose set is empty,
such as a container started with `--network none`, cannot activate, and the activation
error says so.

The Core prints the fingerprint at start and returns it from the retrieve operation
(§5.4). Docker assigns a random address per container start, so the Docker install docs
pin one with `--mac-address`. A multi-node cluster has one fingerprint per node; each
node activates against the same license, and the portal counts activations against `n`.
A cold standby node consumes a seat; the docs say so.

The scheme is a deterrent, not a wall, and every vendor surveyed accepts that (§8). The
control that holds is the activation ledger (§5.7). Recording `fs` in the token lets a
later scheme ship without invalidating issued licenses.

### 5.3 Core verifier and enforcement

`core/pkg/service/channel/verification` keeps its name and location (§5.10). The service
holds the parsed grant, the machine fingerprint, the public key set, and the clock
high-water mark. The public keys are package constants. `ServiceConfig` takes an
optional `Keys` override for tests only; `start` never populates it, and no environment
variable maps to it. Internal string literals keep the existing base64 convention.

The KV holds one entry per activated license, keyed by `jti`, under the existing
obfuscated prefix. Because Aspen replicates the KV, every node sees every activation and
picks the entry whose `fp` intersects its own hardware. Activating all nodes of a
cluster through any one node therefore works without a second mechanism.

The service also persists a clock high-water mark: the latest time it has observed,
written on start and hourly. A start whose clock is earlier than the mark by more than a
tolerance treats every license as expired, which defeats the rollback that offline
verification would otherwise accept.

At start, the service verifies the signature, the fingerprint, and the term, and records
one of three states:

- **Unlicensed**: No matching entry, a bad signature, an unknown `kid`, or a fingerprint
  mismatch. The Core starts. The connectivity check and the license operations work.
  Every other API operation fails with `verification.ErrMissing`. The start log prints
  the fingerprint and the Console URL: open `http://<listen>` to activate.
- **Expired**: `exp` plus the grace window is in the past and `mv` does not cover this
  version, or `mv` alone does not cover this version. Same behavior as unlicensed, with
  `verification.ErrExpired` and, for the version case, the ceiling in the message.
- **Licensed**: Everything works. Inside the grace window, or past `exp` but covered by
  `mv`, the state is licensed with a warning that the log loop repeats and the Console
  shows. The channel cap applies through `IntOverflowCheck` exactly as today
  (`channel/writer.go:489-492`, `channel/service.go:230`). A zero cap disables it.

The state is fixed for the life of the process. A Core that started licensed stays
licensed until it restarts, whatever the clock does, by principle 4. The expiry log loop
stays and gains the version ceiling, since it is the only warning an offline machine
gets.

Gating "every other operation" lives in one place. `BindTo` in `core/pkg/api/layer.go`
already keeps two rosters of endpoints: the two that skip the token check, and the rest.
It gains a third: every endpoint except `license.retrieve` and `license.activate` also
carries `verification.Middleware`, which returns `ErrMissing` or `ErrExpired` without
calling the handler while the state is not licensed. The roster is the allowlist; the
middleware never inspects the request target, which is a path on HTTP and a method name
on gRPC. Errors register with freighter through `errors.Register` in an `init()`,
matching `core/pkg/api/arc/errors.go:43` and `core/pkg/service/auth/errors.go:86`, so
the Console and clients decode them by type instead of by message.

The connectivity check's `ClusterInfo` (`core/pkg/api/auth/auth.go:32`) gains a
`verification` field holding `ok`, `missing`, or `expired`, returned unauthenticated by
`/connectivity/check` and by login. It is one word: the fingerprint and the grant stay
behind the authenticated retrieve. Every client learns the state on the first round trip
it already makes, and the Docker health check, which calls this endpoint, keeps working
on an unlicensed container.

### 5.4 Core API and start flags

Two authenticated operations, under `core/pkg/api/verification`, wired at the five
transport sites like every other endpoint. On the wire they are `license.retrieve` and
`license.activate`; the Core holds the route strings as base64 literals (§5.10):

- **`license.retrieve`**: Returns the state (`unlicensed`, `expired`, `licensed`), any
  warning, the fingerprint, and the decoded grant when present. Requires an
  authenticated user; the fingerprint is not sensitive, but the grant names the
  organization.
- **`license.activate`**: Accepts a token, verifies it against the key set and the
  fingerprint, stores it in the KV, and flips an unlicensed or expired process to
  licensed without a restart. A machine-bound token is safe to email or paste, because
  it is useless anywhere else.

Permission is a first-class RBAC object. `schemas/synnax/ontology.oracle` gains a
`verification` resource type, a permission-only type in the manner of `framer`. The
built-in Owner role holds every action on it and the Viewer role holds retrieve; the
Engineer edit list does not include it, so Engineers read license state and Owners
activate. `license.retrieve` enforces the retrieve action on the type and
`license.activate` enforces update, through the existing enforcer. Built-in policies are
rewritten on every start (`rbac/builtin/provision.go:33`), so existing clusters need no
migration. The Console labels the type "License" in its permission views.

The `--license-key` flag and `SYNNAX_LICENSE_KEY` remain, now accepting a token, and a
`--license-file` flag reads one from a path, which is what systemd units and the Windows
service want. Both activate at start with no interactive step, which is how a
provisioned server and the CI runner (§5.9) run. A running Core is activated from the
Console, desktop or embedded (§5.5). There is no `synnax license` command: nothing in
`core/cmd` talks to a running Core today and there is no Go client, so a CLI would be a
client library built for one command. Scripts that want the state call the retrieve
endpoint with `curl` and a bearer token; the docs show the one-liner.

The TypeScript and Python clients gain a `license` module wrapping both operations, and
the C++ client gains the `sy.verification` error family so the Driver, including the
copy the Core bundles and spawns, retries the gate error the way it retries an
unreachable Core instead of exiting.

### 5.5 Console

The connection lifecycle from RFC 0049 reads the `verification` word from the
connectivity check and gains an `unlicensed` reason beside `auth` and `incompatible`. In
that state the client never opens the change stream, so the epoch never advances and no
synchronizer or Flux query runs; unary calls other than check, login, and the two
license operations are refused client-side with the typed error. The check loop keeps
polling, so a Core activated by a start flag is noticed by a Console left open. The
layout shows an activation screen in place of the workspace, behind login and ahead of
the settled gate: the fingerprint from `license.retrieve` with a copy button, a paste
field and file picker for the token, and a link to the portal's activation page. On
success the next check flips the state and the screen dismisses without reconnecting.
When licensed, the version info modal (`console/src/platform/version/useInfoModal.tsx`)
shows the edition, organization, term, node count, and channel usage, and a warning
banner appears inside the grace window or under a version fallback.

The same screen serves the embedded web Console the Core hosts, which is how a headless
server is activated without a desktop install.

Desktop builds add a sign-in state (§5.8) above the activation screen; enterprise builds
never show it. The Desktop build is the one RFC 0063 defines, selected by its `DESKTOP`
build-time constant, which only `console/src/app` reads. `app/window/Guard.tsx` puts the
license gate in both guard trees. In Desktop the gate sits outside the embedded Core
guard, because an unlicensed Core never settles, and its activation screen is
standalone: no connection island and no log out action.

### 5.6 Portal identity and organizations

Clerk provides sign-up, sign-in, sessions, and, for team organizations, membership,
invitations, and roles through `@clerk/astro`, on a direct Clerk account. The portal
renders every one of those screens itself on Clerk's client API (§5.11); no Clerk widget
appears on the site. The site's middleware gains the Clerk handler ahead of the existing
CSP handler, and the CSP allowlist gains Clerk's domains. After sign-in, PostHog
identifies the user, which the site's `person_profiles: "identified_only"` setting
already anticipates. The portal's organization rows follow Clerk at the seams that read
them: resolving a session upserts a row for every team the user belongs to, and issuing
a license upserts the row for the team the staff member chose from Clerk's organization
list. A Clerk webhook mirrors the same creations as a fast path; it is idempotent and
nothing depends on its delivery.

Staff are members of the Synnax Labs team organization with the `owner` or `admin` role;
the staff area checks membership of that one organization key, held in an environment
variable.

Inside a team, every member activates machines, releases seats, and downloads tokens,
because the person at the test stand is rarely the account admin. Only admins invite and
remove members and change roles. No license action is admin-only.

Neon Postgres, on a direct Neon account, holds the portal's tables through Drizzle:

- **`organization`**: `key`, `kind` (`personal` or `team`), `name`, `clerk_org_id` (team
  only), `owner_user_id` (personal only).
- **`license`**: `key`, `organization`, `edition`, `term` (`subscription` or
  `perpetual`), `nodes`, `channels`, `expires_at`, `max_version`, `label`, `issued_by`,
  `revoked_at`. The token is regenerated from the row, never stored.
- **`activation`**: `license`, `fingerprint` (the hash set), `first_seen`, `last_seen`,
  `released_at`.
- **`event`**: Append-only audit log: who issued, activated, renewed, released,
  transferred, or revoked what, and when.

Resend sends transactional mail that Clerk does not: expiry warnings at 30, 7, and 1
days, and revocation notices. A Vercel Cron job runs the expiry sweep daily.

`infra/hub/` declares in Terraform what has a provider: the KMS signing key and the IAM
identity the Vercel runtime signs with, the Vercel environment variables and domain, and
the GitHub Actions secret for CI. State lives in the HCP Terraform free tier. Cron
schedules stay in `vercel.json`. Neon and Clerk are installed through the Vercel
Marketplace, which injects the connection string and the Clerk keys into the project;
their remaining dashboard steps, and Resend, which has no provider, are documented in
`infra/README.md`. The layout is one Terraform root per lifecycle, so `infra/runners/`
can later provision integration test runners without sharing state with the signing key.

### 5.7 Portal licenses and the activation ledger

An organization's licenses page lists its licenses with edition, term, node count, and
activations. Each license opens to its activation ledger: machines that hold seats by
the name given at activation, when they were last issued a token, and a release action
that frees the seat. A released machine can reactivate, which is the Ignition shape for
a hardware change.

Issuing a token is one endpoint, `POST /api/portal/licenses/:key/activate`, taking a
fingerprint and the name the machine goes by. The name is required, because the only
moment anyone knows which box a set of hashes belongs to is the moment they activate it.
It checks that the caller is a member of the owning organization, that the license is
not revoked or expired, and that active activations are below `nodes` (or that this
fingerprint already holds a seat), then inserts or touches the activation row, writes
the event, and returns the signed token. It is rate limited per caller and per
organization. Enterprise users reach it through the offline page (paste the fingerprint
the Core printed, download the token) and Desktop's renewal (§5.8).

Staff issue every enterprise license, trials included, from the staff area: pick the
organization, set node count, channel cap, a label, and the term. A subscription takes
an expiry and an optional fallback version; a perpetual license takes a maximum version
and no expiry; a trial is a short subscription. No self-serve trial and no payments
exist in this version.

### 5.8 Desktop sign-in

Desktop follows RFC 8252 with a custom scheme of its own, `synnax-desktop://`,
registered in `tauri.desktop.conf.json`. The Console keeps `synnax://`, so the two apps
installed side by side never claim each other's links. The Desktop deep link handler
replaces the Console's link registry in the Desktop build; no Console link can reach
Desktop.

The app reads the fingerprint from its embedded Core, mints a one-time `state` value,
and opens `docs.synnaxlabs.com/desktop/sign-in?state=&fp=&name=&v=` in the system
browser, `name` being the machine's hostname and `v` the app version. The user signs in
with Clerk; the page then calls `POST /api/portal/desktop/link`, which issues a desktop
license for the user's personal organization, records the activation against the
fingerprint, mints an opaque renewal secret, and answers the token beside the secret.
The page opens `synnax-desktop://activate?state=&token=&secret=` and shows an "Open
Synnax Desktop" button for a browser that blocks the navigation. The app refuses a link
whose `state` it did not mint, calls `license.activate` on the embedded Core with the
token, and keeps the secret and the activation key in its account slice, persisted with
the rest of the session. No code exchange exists: the token is bound to the host, so a
captured link licenses nothing else, and the portal session never leaves the browser.

Each machine holds its own desktop license: `edition: desktop`, `term: subscription`,
`nodes: 1`, `channels: 0`, and an expiry one term out. The activation row stores the
hash of the renewal secret and the machine name. A machine that signs in again
supersedes its earlier link: any unreleased activation in the organization that shares a
host hash is released and its license revoked, so one machine is one entry. On launch
and every six hours, while online, the app renews when the license is within the renewal
threshold of expiry: `POST /api/portal/desktop/renew` with the secret as a bearer token
slides the license's expiry, touches the activation, and answers a fresh token that the
app activates. The route answers any origin, since the app calls it from its own; the
secret is the guard. Unlinking the machine in the portal releases the activation, clears
the secret hash, and revokes the license, so the next renewal is refused and the license
lapses at expiry plus the grace window. The app clears its account slice on a refused
renewal. A machine that never reaches the portal runs until the lapse. The expiry
notices (§5.7) skip the desktop edition.

In Desktop the license gate shows a sign-in screen in place of the enterprise activation
screen: one line and a Sign in button, a waiting state while the browser is open, an
offline state with Try again, and a link, Use a license file, that opens the
paste-and-file screen standalone. Desktop's unlicensed messages use plain words ("Sign
in to continue", "Your sign-in has lapsed") per RFC 0063 §5.5. There is no sign-out in
the app: the version modal shows the signed-in email with a link to the portal, and
switching accounts is Unlink in the portal or Erase all data.

### 5.9 Development, CI, and hosted Cores

Every build enforces, so development needs real licenses, and the Synnax Labs
organization provides them:

- **Engineer machines**: Each engineer activates once with a long-lived internal license
  through the ordinary offline page. Local builds then run licensed until it expires.
- **CI**: Runners have random hardware. Staff issue a floating license with a term of a
  few months and store its token as the GitHub secret `SYNNAX_LICENSE_TOKEN`, which the
  workflows pass through `SYNNAX_LICENSE_KEY`. The secret is new because a released Core
  refuses to start on a token, so the old secret stays only for the job that runs the
  released image. The floating token is the one license that is a secret.
- **Before the portal deploys**: The KMS key is created first, and the hub's signing
  module runs as a local script so staff can sign the engineer and CI tokens while the
  portal is still on a branch. There is no bootstrap key and no rotation.
- **Hosted Cores**: The demo Core behind the docs live plot and any other Core Synnax
  runs hold ordinary machine-bound licenses from the internal organization, listed in
  the cutover checklist (§7.0).
- **Go tests**: Tests of the verification package and of gated endpoints construct the
  service with a throwaway keypair through `ServiceConfig.Keys` and sign their own
  grants with `Sign`. No other test touches licensing, because the test fixtures that
  open a service layer inject a signed test license the same way.

### 5.10 Obfuscation

The Core hides where enforcement lives. Release builds strip symbols with `-w -s`
(`.github/workflows/build.synnax.yaml:627`), but Go keeps package paths and function
names in the binary for stack traces, so identifiers matter as much as strings. The
existing convention in `verification` and `core/cmd/start/flags.go` sets the rule, and
this RFC extends it to every new piece in the Core:

- **Identifiers are neutral**: The package stays `channel/verification`. The grant type
  is `verification.Grant`, the fingerprint is `verification.Host`, the key set is
  `verification.anchors`, the gating middleware is `verification.Middleware`, the API
  package is `core/pkg/api/verification`, and the transport field is `Verification`.
  Nothing in `core/` is named `license`, `activate`, or `fingerprint`.
- **Strings are base64 literals**: Route paths, flag names and help text, log lines,
  error messages, and the KV prefix are `base64.MustDecode` literals decoded at package
  or command initialization. Wire error types and the ontology resource type use the
  neutral word `verification`.
- **Claims are short**: The JWT keys in §5.1 keep the struct tags free of license
  vocabulary, and a JWT in a binary is unremarkable because the Core already issues
  them.
- **Readable names live outside the Core**: The clients, the Console, the docs, the
  portal, and the wire paths say `license`. Oracle's `@ts name` and `@py name` produce
  `License` from `Grant`. None of those surfaces enforce anything.

Obfuscation raises the cost of a casual `strings` or `grep`; it does not stop a reader
of the source. Principle 3 sets the bar it works toward.

### 5.11 Portal interface

The portal is a section of the hub, not a second shell. The header gains a fourth entry,
Portal, beside Reference, Blog, and Releases, visible signed out; a signed-out click
lands on sign-in and returns to the page asked for. Signed in, the header's sign-in
button becomes an avatar menu with Portal, Account, and Sign out. Below the header, a
portal page takes the docs' left rail for its own sidebar: the organization switcher on
top for a user in more than one team, then Licenses for a team member or Desktop for a
personal user, and Account, and for staff a second group with Licenses. The content
column runs at Pluto's own type scale and radius, scoped to the portal frame the way the
feedback modal already scopes them, so the portal reads as an application while the docs
keep their editorial scale. Theme follows the operating system, as the docs do. Below
the mobile breakpoint the sidebar folds into the existing drawer and tables collapse to
cards.

Every portal page is an Astro page that loads its data on the server through the
existing server functions and renders one React island with that data as props, hydrated
on load so the first paint is server HTML. Actions open Pluto modal dialogs with the
Console's anatomy: a header bar carrying the title and a close button, a body, and a
footer bar with the primary action and the save shortcut. A dialog posts JSON to the
existing `/api/portal/...` route, shows a route error inline as a status summary, and on
success navigates to the same URL so the page reloads its data. The `?error=` query
channel and the page-level form posts are deleted. That anatomy moves from the Console
into Pluto as `Modal` (frame, header, body, footer), and the Console's `platform/modals`
keeps only what binds it to the session: the factory and the stack.

- **Sign-in and sign-up**: `/sign-in`, `/sign-up`, `/sign-in/reset`, and
  `/sso-callback`, built on `signIn`, `signUp`, `setActive`, and
  `handleRedirectCallback` from Clerk's client. Sign-in offers email and password, a
  Google button, and a Microsoft button; a second factor renders when the account has
  one. Reset sends an email code and takes a new password. Sign-up takes name, email,
  and password, verifies the email with a six-digit code, and lands on `/portal`. OAuth
  redirects to `/sso-callback` and completes to the requested page. Clerk's error codes
  map to field help text; nothing else surfaces raw vendor copy.
- **Licenses** (`/portal`, team members): The organization's licenses as a table of
  label, edition, term, seats in use, and a status tag. Activate a machine opens a
  dialog: the license to activate, if the page did not name one, a name for the machine,
  a field for the fingerprint the Console copied, and an Activate action that downloads
  the token and leaves the dialog in a done state with Download again.
  `/portal/licenses/activate`, the page the Console links, is the licenses page with
  that dialog open, taking `?license=`. An organization with no licenses sees why.
- **Desktop** (`/portal`, personal users): The machines signed in through Desktop (§5.8)
  by name, with first seen and last renewal, an Unlink action per machine, and an empty
  state. The Enterprise panel sits beneath the list.
- **License** (`/portal/licenses/<key>`): The label, status tag, and actions on top:
  Activate a machine, and for staff Edit, Floating token, and Revoke, the last a hold to
  confirm. Edit changes the terms of the license in place, keeping its key so seats and
  history survive a renewal; it refuses a seat count below the machines holding one, and
  the machines take the new terms on their next token. A facts grid for edition, term,
  seats, channels, issued, and key. The machines table with first seen, last token, and
  a per-machine menu of Download token and Release, the latter confirmed. The license's
  activity from the event table beneath, newest first.
- **Account** (`/portal/account`): Profile with name, email, avatar, and password change
  through Clerk's user API. For a personal user, the Enterprise panel in place of a
  teams section. For each team the user belongs to, its members with their roles, and
  for an admin an Invite dialog taking email and role, with remove and role change in a
  per-member menu. Membership goes through Clerk's organization API; there is no team
  creation on the site.
- **Staff licenses** (`/portal/staff/licenses`): Every license with its organization,
  and Issue license as a dialog: the organization chosen from Clerk's organization list,
  label, term, nodes, channels, and the expiry or maximum version the term needs,
  validated before the post. Issuing upserts the organization row (§5.6).

## 6 What this RFC does not cover

- Support. The docs feedback form stays on Formspree, and the portal has no support
  threads, inbox, or help desk integration.
- Building Synnax Desktop itself: embedding a Core in the Console bundle and the feature
  flag surface beyond the sign-in state. RFC 0063 covers that build. This RFC defines
  the license path Desktop uses.
- Payments and self-serve purchase. The license table is shaped so a Stripe flow can
  create rows later; nothing here depends on it.
- The privacy policy. Accounts, Clerk, and PostHog identification change the data
  processing it describes, and it is updated alongside Phase 2, outside this RFC.
- Merging the landing page into the hub. It lives in the separate `synnaxlabs/landing`
  repository on the same stack, and moving it in, with the docs under one domain, needs
  permanent redirects, an Algolia rebuild, and every docs link in the Console updated.
  It is a pull request of its own after this work.
- Integration test runner infrastructure. `infra/runners/` is where it goes; the layout
  is chosen here, the resources are not.

Considered and left additive, because JWT claims and the portal tables absorb each
without a format change:

- **Feature entitlements** as a claim list, for integrations or Arc.
- **Per-user seats** on a license.
- **Single sign-on** for customer portal accounts, a Clerk add-on.
- **Warm failover** without a seat, which would need a "standby" activation kind.

## 7 Implementation phases

A rename lands first on its own, then Phases 1 and 2 stack on `rc` and merge as one
unit, so no Core on `rc` demands a token before the key that signs it exists. Phase 4
lands with the Desktop bundle.

- **Phase 0: Rename.** `git mv docs/site hub`, package `@synnaxlabs/hub`, and the twelve
  references outside the directory. Merged alone, because a directory move carried
  inside a stack rebases badly. Boundary earned by risk isolation.
- **Phase 1: Core license primitive.** Two pull requests. The first: the Oracle grant
  schema, the `verification` resource type through `oracle migrate ontology`, `Sign` and
  `Verify` in `verification`, the key set, the fingerprint, the clock high-water mark,
  the term checks including `mv`, the gated roster, error registration, the two API
  operations at all five transport sites, the `verification` word on the connectivity
  check, the `--license-file` flag, the start log, a `core/pkg/service/mock` fixture
  that opens a licensed layer for the five test suites that open one today, the
  TypeScript `license` module and `unlicensed` connection state, the Python module, the
  C++ error family with the Driver retry, and the workflows switched to the new secret.
  Deletes the old parser, `FreeCount`, the free-tier default, and the channel-count-only
  `info`. The flag keeps its obfuscated name. The second: the Console activation screen
  and guard, the info modal block, the warning badge, and the four docs pages. Boundary
  between the two earned by risk isolation: wire and enforcement apart from UX.
- **Phase 2: Portal accounts and licenses.** One pull request: `infra/hub/`, Clerk,
  Neon, and Resend on direct accounts, the middleware and CSP changes, the `astro:env`
  schema, the static-check exclusions for session-bound routes, the webhook, the four
  tables, the organization pages, KMS signing, the activation ledger, the activation
  endpoint with rate limits, the offline page, the expiry cron, and the staff area with
  all three terms. Its infrastructure step is applied first of the whole unit, because
  the key and the CI secret must exist before Phase 1's Core runs in CI. Boundary earned
  by a green intermediate state: after this phase staff issue the cutover licenses
  (§7.0) and the release can ship.
- **Phase 3: Portal interface.** Two pull requests. The first moves the modal anatomy
  from the Console into Pluto and migrates the Console's callers, a mechanical change
  kept apart by risk isolation. The second replaces every portal page: the header entry
  and avatar menu, the sidebar, the custom sign-in, sign-up, reset, and callback pages,
  the licenses, license, account, and staff pages with their dialogs, the `/portal`
  routes with the Console's activation link moved, and the deletion of the form-post
  pages, the `?error=` channel, and the portal stylesheet. Boundary earned by
  reviewability: Phase 2 is reviewed on its server behavior, and this one on its
  interface. Before it deploys, the production Clerk instance needs the name attribute,
  organizations, and the Microsoft connection enabled, and the Neon database needs the
  Drizzle migration applied; neither the build nor the deploy runs it.
- **Phase 4: Desktop sign-in.** One pull request on top of Phase 3: the `synnax-desktop`
  scheme, the sign-in page and its link route, the renew route with the renewal secret
  on the activation row, the machine name, the Desktop sign-in screen and deep link
  handler, the account slice, and the renewal loop. The Desktop bundle and its build
  flag exist (RFC 0063). Before it deploys, the Neon database needs the Drizzle
  migration that adds the two activation columns.

### 7.0 Compatibility

The old key format is dropped without a grace window. Before the release that carries
Phase 1 ships from `main`, staff work a checklist through the Phase 2 staff area: every
subscription holder gets a subscription license; the existing perpetual customer gets a
perpetual license whose maximum version their contract sets, or no ceiling if the
contract grants all future versions; the demo Core, the engineer machines, and CI get
internal licenses. Each activation is confirmed before release. The release notes state
that the release requires a license to start and link the activation page. Data on disk
is unaffected: the old KV entry is ignored and removed on first licensed start.

Old clients connecting to an unlicensed Core see a generic error, since they predate the
registered error types. New clients decode it.

## 8 Resolved decisions

1. **Organizations own every license, with a personal organization per user**: A
   polymorphic owner (user or organization) forks every query and permission check in
   two. The trade is real: a solo user sees one more concept.
2. **One signed offline token with expiry and renewal**: Online-only validation makes a
   running Core depend on our uptime and cannot run air-gapped. Extending the opaque key
   keeps it forgeable. Keygen would remove the signer but not the portal, the verifier,
   or the identity model, and adds a vendor for the life of the product; the token stays
   standard enough that issuance could move to a vendor behind the same endpoint. A
   Desktop-only session would leave two mechanisms.
3. **No download wall**: Every channel except the portal page is public already, and the
   Tauri updater, `pip`, and `docker pull` cannot be gated. Desktop sign-in and
   activation give the counts a wall would have given. The trade is real: nothing stops
   a direct GitHub link.
4. **Clerk and Neon inside the Astro site**: Supabase has no organization concept, so
   invitations and roles would be ours to build. PocketBase is pre-1.0, has no
   organizations, runs on one node, and needs a second host. A separate Go service adds
   a deploy for logic that has no Go consumer. The trade is real: Vercel becomes
   critical for issuance and renewal, though never for a running Core.
5. **Browser handoff for Desktop**: RFC 8252. An embedded web view is what the RFC
   forbids, and Google and Apple block their sign-in inside it. A first draft paired it
   with a device code flow in a CLI; decision 23 removed the CLI.
6. **The standalone Core requires a license**: Desktop is the free edition; anything
   standalone is enterprise. An unlicensed Core starts and only activation works,
   because refusing to start would block online activation and the embedded Console.
7. **Every build enforces**: Injecting the key only at release would make any clone an
   unlimited Core. The cost is one activation per engineer machine and one rotating CI
   secret.
8. **MAC-derived fingerprint with an activation ledger**: FlexNet uses the MAC host id
   and documents the VM and Docker holes. Ignition uses a hardware fingerprint plus a
   server-side activation count with reactivate, which this design copies. Keygen's
   random-per-boot fingerprint needs an online heartbeat. Grafana's URL binding does not
   fit Cores on private networks.
9. **Staff issue every enterprise license**: No self-serve trial. The standalone quick
   start directs readers to request a license.
10. **Activation is an API operation callable from the Console**: The Core is the
    verifier, so the worst a caller can do is install a license Synnax issued for that
    machine.
11. **Hard cutover from the old key**: A grace window keeps the forgeable parser alive
    for a release. The holder list is small and known.
12. **The Core stays obfuscated**: A first draft renamed the package to `license` on the
    grounds that the CLI and Console say the word anyway. Rejected: Go binaries keep
    package and function names even when stripped, so the identifiers are what a
    `strings` pass finds. Readable names belong to the surfaces that do not enforce
    (§5.10).
13. **A standard JWT, not a custom line**: A first draft framed the token as
    `synnax1.<payload>.<signature>`. Rejected: the Core already verifies EdDSA JWTs, a
    custom frame has no `kid` and so no key rotation, and every parsing edge the
    libraries handle would be ours.
14. **Private key in AWS KMS**: A Vercel environment secret is one leaked variable away
    from unlimited licenses until a binary ships with a new key set. KMS supports
    Ed25519 signing and the key never exists in plaintext. The trade is real: a second
    cloud account and a network call per issuance.
15. **The portal owns organizations; Clerk backs only teams**: A first draft made every
    organization a Clerk organization and put its id in the token. Rejected on cost,
    since Clerk bills per retained organization and Desktop makes every free user one,
    and on principle 7, since a vendor id would sit in every stored license.
16. **Time is checked at start, never against a running process**: A first draft
    hard-stopped an expired Core. Rejected: Grafana keeps running on an expired license
    with a banner, and a test operator's cost of a mid-run stop is a lost test. The
    trade is real: a process can outlive its license until its next restart.
17. **The fingerprint is a set matched by intersection**: A single hash over all
    addresses breaks on any adapter change. FlexNet's any-listed-host-id rule keeps the
    same deterrent with far fewer reactivations.
18. **Perpetual licenses are first-class**: An existing deal is perpetual. `mv` and the
    perpetual term ship in Phase 1 and the staff form in Phase 2, not as a reserved
    claim. A perpetual license with no ceiling can only be limited by the machine
    binding; that is the nature of the deal and a conscious choice per contract.
19. **Version ceiling by minor version, not release date**: The repository ships one
    shared minor version, patch releases stay covered automatically, and no build
    timestamp or clock is involved.
20. **License state on the connectivity check, not a retrieve after connect**: A first
    draft had the Console call `license.retrieve` once connected. Rejected: the
    Console's synchronizers run when the change stream goes live, before any
    Console-level call, and they hit gated endpoints. The connection machine must know
    first, and the check is the call it already makes. The trade is real: one word of
    state is readable without credentials.
21. **The gate is a transport roster, not target matching**: The request target is a
    path on HTTP and a method name on gRPC. The roster idiom already exempts login and
    connectivity, and a reviewer reads which endpoints are gated from one list.
22. **A `verification` resource type, not the `builtin` root and not the root-user
    flag**: Enforcing update on the ontology root node was rejected as a shortcut
    through an unrelated object. Checking the root-user flag was rejected as a second
    permission mechanism outside RBAC, since the flag is already reconciled into the
    Owner role on every start. The trade is real: an ontology version migration.
23. **No CLI**: A `synnax license` group needs a Go HTTP client that speaks TLS, and
    none exists; the freighter Go client is test-only and hard-codes plain HTTP.
    Extending it or hand-rolling `net/http` builds a client library for one command.
    Start flags, the Console screen, and `curl` cover every case it served.
24. **No bootstrap signing key**: Phases 1 through 3 merge as one unit, so the KMS key
    exists before any Core on `rc` demands a token. A first draft carried a temporary
    key and a rotation; the unit merge made both unnecessary.
25. **The Driver retries the gate error**: Its reconnect loops retry only unreachable
    Cores, so an unlicensed Core would kill the driver it spawns. Treating the
    verification error like unreachable lets it recover on activation.
26. **Terraform for the providers that exist; the Vercel Marketplace for Neon and
    Clerk**: A first draft put every vendor on a direct account under Terraform.
    Revised: Clerk's providers are community ones that cover a resource or two, Resend
    has none, and a Marketplace install bills through Vercel and injects its variables
    without a secret changing hands. The trade is real: Neon and Clerk settings are
    dashboard steps in `infra/README.md`, not code.
27. **Portal routes under one prefix**: `/portal` lands on licenses, then
    `/portal/licenses/<key>`, `/portal/licenses/activate`, `/portal/account`,
    `/portal/staff/licenses`, and `/api/portal/...` for the endpoints. Sign-in, sign-up,
    and the SSO callback stay at the root, since they are not portal pages. A first
    draft put the pages at the root; once the header named the section Portal, the URL
    had to say the same. The organization is a query parameter, not a path segment, so a
    license URL never changes when an organization is renamed.
28. **The site becomes `hub/`**: `docs/site` understates a site that carries accounts
    and licenses. `site/` and `www/` were rejected as generic, `portal/` names one
    section, `cloud/` implies a hosted service, and a coined name was offered and
    declined. The landing page merge is deferred (§6).
29. **The portal is a section, not a shell**: Tailscale, Vercel, Linear, and Stripe put
    the signed-in surface in an application shell on its own host. The hub is one
    deployment, and the portal will stay small, so it takes one header entry and a
    sidebar in the docs' left rail rather than a shell of its own. The trade is real:
    the portal inherits the docs header and footer, and its density is scoped by CSS
    rather than by a separate layout.
30. **Sign-in and the team screens are ours, on Clerk's client API**: Clerk's prebuilt
    components take an appearance object, not a design; they render their own layout and
    copy, and they would be the only surface on the site not built from Pluto. The trade
    is real: password reset, email verification, second factors, and OAuth callbacks are
    our pages to maintain against Clerk's API.
31. **One island per page over JSON**: Restyling the server-rendered forms was cheaper
    but cannot produce a dialog or an inline error, and every action would remain a
    full-page round trip through a query string. The routes already accept JSON, so the
    island model costs nothing on the server. The trade is real: the portal pages need
    React to act, where the docs pages do not.
32. **Members act on licenses, admins act on the team**: Making every license action
    admin-only would send the engineer at the stand to their manager for a token. The
    trade is real: any member can release another member's machine, and the event log is
    the recourse.
33. **Staff create teams in the Clerk dashboard**: A first draft let any user create a
    team and invite members. Every license is staff-issued (decision 9), so a
    customer-made team is an empty shell until a deal closes, and it puts a second
    concept in front of every personal user. Clerk's dashboard already creates
    organizations and sends invitations, so a creation dialog on the site would be a
    second interface over the same API. The trade is real: onboarding a customer is a
    staff step, and the site has no team creation to test.
34. **Personal accounts see no organization concept**: A personal user has one
    organization they never chose, so a switcher, a teams section, and an organization
    subtitle would name something they cannot act on. Where a team feature would sit,
    one Enterprise panel names it and offers Talk to us. The trade is real: the two
    account shapes fork the licenses page and the account page.
35. **Organization rows follow Clerk at the seams that read them**: A first draft relied
    on the webhook alone, so a dropped delivery, or a local instance the webhook cannot
    reach, left a team invisible to the portal. Session resolve and license issue now
    upsert the rows they need; the webhook remains a fast path. The trade is real: every
    session resolve costs one query per team.
36. **A custom scheme for the Desktop return path**: A loopback redirect (RFC 8252 §7.3)
    works in development and cannot be claimed by another app, but needs an HTTP
    listener in the Rust shell and is blocked by some managed browsers. A scheme of
    Desktop's own reuses the deep link plugin already in the shell and is the shape of
    every desktop app that signs in through a browser. The trade is real: a scheme fires
    only in a bundled app, so `tauri dev` on macOS activates through the file fallback.
37. **The token rides the deep link; no code exchange**: A first draft exchanged a
    one-time code for the token at the portal with PKCE. The token is bound to the host
    and only activates a Core, so a captured link licenses nothing, and the portal
    session never leaves the browser. The trade is real: a link that another app
    registered the scheme for lands a working token on that machine, which the `state`
    check the app performs does not prevent, only a foreign account's token.
38. **A per-machine renewal secret**: A long term with no renewal would make Unlink do
    nothing for a year and lose the liveness signal the free-user count rests on. The
    secret is opaque, hashed on the activation row, and renews one license on one
    fingerprint. The trade is real: a secret sits on disk in the session store, and a
    copy of it renews that one license from anywhere until the machine is unlinked.
39. **One desktop license per machine**: One license per person with a seat per machine
    needs a per-seat expiry, a new column, and a change to the token builder. One
    license per machine reuses the issue, activate, revoke, and ledger paths unchanged.
    The trade is real: a person's machines are a filter on the edition, not a row.
40. **Desktop is unlimited**: A channel cap would not move anyone to enterprise, whose
    value is a standalone Core that other people, Drivers, and scripts reach (RFC 0063
    §8), and it would be the first wall a serious evaluator hits. The trade is real:
    nothing in the free edition is metered.
41. **No sign-out in Desktop**: The Core stores every token it accepts and has no
    operation to drop one, so a sign-out could only stop renewal while the machine ran
    on under the old account for up to the term plus grace. A Core operation that
    un-licenses a running Core would be permanent surface for a case Unlink and Erase
    all data cover. The trade is real: switching accounts on one machine is a portal
    action, not an app action.

## 9 Open questions

- Grace window after `exp`. Proposed: 14 days.
- Clock rollback tolerance. Proposed: 24 hours.
- Internal engineer license term. Proposed: one year.
- CI floating license term. Proposed: 90 days.
- The maximum version, if any, for the existing perpetual customer, per contract.
