# 62 Account and licensing

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-17
- **Related**: [RFC 0011 - Alamos instrumentation](0011-alamos-instrumentation.md),
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md),
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

Synnax has no account. A user downloads a public binary, and an enterprise customer
pastes a key that encodes an expiry and a channel count with arithmetic anyone can
reverse. Nothing connects a person to a machine or a machine to a license.

This RFC adds that connection. A portal at portal.synnaxlabs.com, an Astro app in
`site/portal/` deployed apart from the docs, holds accounts through Clerk and
organizations and licenses in Neon Postgres. The Core gains one license primitive: a JWT
signed with ML-DSA-44, a post-quantum signature, bound to a machine, and verified
offline with public keys compiled into the binary. Two paths issue that token. The free
edition, Synnax Desktop, signs the user in through the system browser and receives a
short-lived license that renews while signed in. The enterprise edition, the standalone
Core, activates through a start flag or the Console with a license that staff issued in
the portal. Downloads stay public, the Core never phones home, a running Core never
stops because of time, and the old key format is deleted.

## 1 Motivation

- **The current key has no integrity**: The Core decodes the date and channel count from
  the key and checks a self-consistency checksum. Anyone who reads the source mints
  keys.
- **Nothing surfaces license state**: The Console and clients have no license interface,
  and an over-cap Core fails channel creation with a plain string error that no client
  recognizes.
- **The free edition is changing shape**: Synnax Desktop, a Console build with an
  embedded Core, becomes the free product. It needs an account to link a machine, so the
  keyless 50-channel tier has no product left to serve.
- **A perpetual deal already exists**: The current key has an expiry and nothing else,
  so it cannot express one.
- **Installs cannot be counted**: Download links fire no event, and GitHub asset
  counters are the only signal.

## 2 Vocabulary

- **Portal**: The Astro app in `site/portal/`, served at portal.synnaxlabs.com:
  accounts, organizations, licenses, and the routes that issue tokens.
- **Scope**: The organization a portal page acts for: the user's personal organization
  or one of their teams.
- **Organization**: The owner of every license, stored by the portal. Every user has a
  personal organization. A team organization is also backed by a Clerk organization for
  membership, invitations, and roles.
- **Edition**: `desktop` or `enterprise`. Desktop is the Console build with an embedded
  Core. Enterprise is the standalone Core, Console, and Driver.
- **License**: A signed JWT that names an organization, an edition, the machines it
  binds to, a machine count, a channel cap, and a term.
- **Term**: A subscription, which carries an expiry, or perpetual, which carries a
  maximum Core version instead. A subscription can also carry a maximum version as its
  fallback after expiry.
- **Fingerprint**: The set of hashes of a machine's physical network addresses, one per
  interface.
- **Floating license**: A license with no fingerprint, valid on any machine. Staff-only.
  CI uses one.
- **Activation**: Storing a license in a Core, and the portal's record that a machine
  holds a seat under a license.
- **Activation ledger**: The portal's list of activations per license, with release and
  reactivate.
- **Grace window**: The 14 days after a subscription expires during which a Core still
  starts, with a warning.

## 3 Principles

1. **One license primitive**: Desktop sign-in and enterprise activation produce the same
   token, verified by the same code.
2. **The running Core never phones home**: Verification is offline. Network calls happen
   in the Console or the portal, never in the Core. This continues RFC 0011 §4.4.0 and
   RFC 0020 §1.
3. **Bypass requires editing code**: The public keys are constants in source. No flag,
   environment variable, or build tag replaces them. Published source can always be
   patched; the bar is that a clone does not run unlicensed by accident.
4. **Time never stops a running Core**: Expiry, the grace window, and the version
   ceiling are checked when a Core starts and when a license is activated. A test
   operator loses a test if a license lapses mid-run, and loses nothing if the Core
   refuses to start the next morning.
5. **Enforcement lives in the license, not the download**: Every artifact stays public.
   Desktop sign-in counts free users, and activation counts enterprise machines.
6. **Buy the standard parts, build the Synnax parts**: Identity, teams, email, and key
   custody come from Clerk, Resend, and AWS KMS. The token, the fingerprint, the
   verifier, the organization model, and the activation ledger are ours.
7. **Vendor ids never enter a stored format**: The license names the portal's own
   organization key. The Clerk id is replaceable without issuing anything again.
8. **Infrastructure is code**: Every cloud resource the portal needs is declared in
   Terraform under `infra/`. What a vendor cannot expose is a documented manual step.

## 4 Current mechanics

The `--license-key` flag, or `SYNNAX_LICENSE_KEY`, reaches `channel/verification`, which
parses the key, stores it under one fixed key in the KV, and hands the channel service a
cap check. Without a key the cap is 50 channels. Nothing binds the key to a machine,
signs it, or exposes its state over the API. The KV is Aspen's, which gossip replicates
to every node, so one fixed key holds one license for the whole cluster.

The Core already verifies EdDSA JWTs for user sessions with `golang-jwt/jwt/v5`
(`core/pkg/service/auth/token`). It also serves the web Console on its own listener,
which is where an operator without a desktop install activates.

## 5 Design

### 5.0 Editions and ownership

Every license belongs to an organization, and the portal owns the organization record. A
user who signs up gets a personal organization with that user as its only member, and no
Clerk organization. An enterprise customer gets a team organization backed by a Clerk
organization, which supplies membership, invitations, and the `admin` and `member`
roles. Staff create the team in the Clerk dashboard when a deal closes and invite the
customer's admin. Customers never create teams, and a personal account shows no
organization concept.

Clerk bills each monthly retained organization past the first 100. Desktop makes every
free user an organization owner, so personal organizations are not Clerk organizations.

Desktop is the only free edition. The standalone Core is enterprise and needs a license
before it does anything except report its fingerprint and accept a license (§5.3). The
keyless 50-channel tier is removed.

### 5.1 The license token

A license is a JWS compact token: a JWT signed with ML-DSA-44 from FIPS 204, which a
quantum computer cannot forge. The signature is 2,420 bytes, so a token runs to about
3,700 characters. `golang-jwt` has no ML-DSA method, so the Core builds and checks the
three parts itself with Go's `crypto/mldsa`. The header carries `alg: ML-DSA-44` and
`kid`. The Core embeds a small set of public keys by `kid`, the anchors. Rotation adds a
key to the set in one release and moves signing to it. A compromised key leaves the set
in the next release, and the licenses signed under it are issued again.

The claims are defined once, in `schemas/synnax/license.oracle`, which generates the Go,
TypeScript, and Python `License` types. The portal's signer uses the TypeScript type.

- **`jti`**: License UUID. The portal's key for the license.
- **`iat`**: Issued-at, in seconds since the epoch.
- **`exp`** (optional): Expiry. Absent on a perpetual license.
- **`claims_version`**: The claim set version, `1`.
- **`organization`**: The portal's organization UUID, never a Clerk id.
- **`edition`**: `d` for desktop or `e` for enterprise.
- **`fingerprints`**: The hashes the license binds to (§5.2). Empty on a floating
  license.
- **`fingerprint_scheme`**: The fingerprint scheme, `1`.
- **`machines`**: How many machines can activate under the license.
- **`channels`**: The channel cap per Core. Zero means no cap.
- **`max_version`** (optional): The maximum Core minor version, such as `"0.62"`.
  Required on a perpetual license. On a subscription it is the fallback after `exp`.
- **`required`**: The claims the license cannot be honored without.

The three terms follow from `exp` and `max_version`:

| Term                       | `exp`  | `max_version` | Behavior                                           |
| -------------------------- | ------ | ------------- | -------------------------------------------------- |
| Subscription               | set    | absent        | Any version until expiry                           |
| Perpetual                  | absent | set           | Any time, up to that minor version                 |
| Subscription with fallback | set    | set           | Any version until expiry, then up to `max_version` |

The claim set grows without a version change. A new claim is optional, so an older Core
ignores it. A claim that restricts, such as a feature list, also goes in `required`: a
Core that does not understand a required claim refuses the token instead of ignoring the
restriction. `claims_version` changes only for a change that an older Core must never
read as version `1`. Verification also refuses a token that has neither `exp` nor
`max_version`, or whose `max_version` does not parse.

`Sign(priv, kid, license)` and `Verify(anchors, token)` live in the Core's `license`
package, and only tests call `Sign`. Production signing happens in AWS KMS under an
`ML_DSA_44` key, so the private key never exists in plaintext. The portal calls KMS
`Sign` with `ML_DSA_SHAKE_256` and `MessageType: RAW` over the signing input it builds.

### 5.2 Fingerprint

Scheme 1 lists the network interfaces, drops loopback, point-to-point, and interfaces
without a hardware address, and hashes each address to lowercase hex with Argon2id: one
pass, 4 MiB, and a fixed salt. A hardware address has few possible values, so a fast
hash gives the address back in seconds. Argon2id makes each guess cost milliseconds. The
license carries the set seen at issuance. A token applies when that set shares a hash
with the machine's current set, FlexNet's "any listed host id" rule, so a new USB
adapter or a swapped Wi-Fi card does not break a license. A machine with no hashes, such
as a container started with `--network none`, cannot activate.

The Core prints the fingerprint at start and returns it from `license.retrieve`. Docker
assigns a random address per container start, so the Docker docs pin one with
`--mac-address`. Each node of a cluster activates against the same license, and the
portal counts activations against `machines`. A cold standby takes a seat.

The scheme is a deterrent, not a wall. The control that holds is the activation ledger
(§5.7). Because the token records its scheme, a new scheme does not invalidate issued
licenses.

### 5.3 Core license service and enforcement

`core/pkg/service/license` holds the machine's fingerprint, the anchors, the stored
licenses, and a clock mark. The anchors are package constants. `ServiceConfig.Anchors`
overrides them for tests only: `start` never sets it, and no flag or environment
variable maps to it.

The KV holds one entry per activated license, keyed by `jti` under a fixed prefix. Aspen
replicates the KV, so every node sees every activation and picks the entry whose
fingerprints match its own hardware. Activating every node through one node therefore
works. The service deletes the entry the old key format wrote.

When it opens, the service verifies each stored token, evaluates its term, and records
one of three states:

- **`missing`**: No stored token verifies and matches this machine. The Core starts, but
  only the connectivity check, login, and the license operations work. Every other
  request fails with `license.ErrMissing`. The start log prints the fingerprint.
- **`expired`**: The best token no longer covers this Core: `exp` plus the grace window
  has passed and `max_version` does not cover this version, or `max_version` alone does
  not. Requests fail with `license.ErrExpired`, whose message gives the reason.
- **`ok`**: Everything works. Within 7 days of expiry, inside the grace window, or past
  `exp` but covered by `max_version`, the state carries a warning that the log repeats
  hourly and the Console shows. The channel cap applies to external channels through the
  channel service's `ChannelLimit` check. A zero cap disables it.

The state holds for the life of the process, by principle 4, with two exceptions:
activation moves it to `ok`, and a clock that catches up makes the service reload.

The clock mark is the latest time the service has seen. It moves forward, never back, at
open and on every hourly check. A clock more than 24 hours behind the mark cannot prove
that a license is inside its term, so a license with `exp` counts as expired, or falls
back to `max_version` when it has one. A license with only `max_version` ignores the
clock. While the clock is behind, the service checks it every minute and reloads the
stored licenses once it catches up. A device that boots without a clock and starts the
Core before NTP syncs therefore unlocks itself within a minute of the sync.

The gate lives in one place. `BindTo` in `core/pkg/api/layer.go` keeps three endpoint
rosters. Login and the connectivity check skip the token check. `license.retrieve` and
`license.activate` need a token but skip the license. Every other endpoint also carries
`license.Middleware`, which answers `ErrMissing` or `ErrExpired` without calling the
handler. The middleware never reads the request target. License errors register with
freighter under the `sy.license` family, so clients decode them by type.

The connectivity check carries the state as one word, `ok`, `missing`, or `expired`,
without authentication. Every client learns it on the round trip it already makes, and
the Docker health check keeps working on an unlicensed container. The fingerprint and
the license stay behind the authenticated retrieve.

### 5.4 Core API and start flags

Two authenticated operations live under `core/pkg/api/license`:

- **`license.retrieve`**: Returns the state, any warning, the fingerprint, and the
  decoded license when there is one.
- **`license.activate`**: Verifies a token against the anchors and the fingerprint,
  stores it, and moves the Core to `ok` without a restart. It refuses a token that is
  invalid, bound to another machine, or expired. A machine-bound token is safe to email,
  because it is useless anywhere else.

Access goes through the existing RBAC enforcer on one object, `builtin:license`. The
built-in roles already grant the `builtin` type: the Owner holds every action and the
Engineer holds retrieve. Owners therefore activate and Engineers read, with no new
resource type.

`--license-key` and `SYNNAX_LICENSE_KEY` take a token, and `--license-file` reads one
from a path, for systemd units and the Windows service. Both activate at start, which is
how a provisioned server and CI run, and a token the Core refuses stops the start. A
running Core activates through the Console. There is no `synnax license` command:
scripts call the retrieve endpoint with `curl`.

An unlicensed Core starts its embedded Driver without waiting for it, because the Core
refuses the Driver's rack until a license applies. The Driver retries on its own.

### 5.5 Clients and Console

The TypeScript and Python clients gain a `license` module with `retrieve` and
`activate`, and typed license errors. The C++ client gains the `sy.license` errors, and
the Driver retries them the way it retries an unreachable Core instead of exiting.

The TypeScript connection lifecycle (RFC 0049) reads the state from the connectivity
check and gains an `unlicensed` error reason. While it holds, the client refuses unary
calls other than the check, login, and the two license operations with the typed error,
instead of spending its retry budget on each call. The check loop keeps polling, so a
Console left open notices a Core activated by a start flag. A Core from before licensing
sends no state and reads as `ok`.

While the Core is unlicensed, the Console shows an activation screen in place of the
workspace, behind login: the fingerprint with a copy button, a field and a file picker
for the token, and a link to the portal's activation page. On success the client checks
again at once, the state flips, and the screen goes away. When licensed, the Core badge
and the version modal show the edition, organization, term, machines, and channel cap,
with any warning. The same screen serves the web Console the Core hosts, which is how a
headless server activates.

In Desktop (RFC 0063), the gate sits outside the embedded Core guard, because an
unlicensed Core never settles, and it shows the sign-in screen of §5.8 instead.

### 5.6 Portal identity and organizations

Clerk provides sign-up, sign-in, sessions, and, for teams, membership, invitations, and
roles, through `@clerk/astro`. The portal renders every screen itself on Clerk's client
API (§5.10), and no Clerk widget appears. The portal's organization records follow Clerk
where they are read: resolving a session stores a record for each of the user's teams,
and issuing a license stores the record for the team staff chose. A Clerk webhook does
the same as a fast path, and nothing depends on its delivery.

Staff are members of the Synnax Labs team with the `owner` or `admin` role. Inside a
team, every member activates machines, releases seats, and downloads tokens. Only admins
manage members.

Neon Postgres holds the portal's tables through Drizzle:

- **`organization`**: `key`, `kind` (`personal` or `team`), `name`, `clerk_org_id` (team
  only), and `owner_user_id` (personal only).
- **`license`**: `key`, `organization`, `edition`, `term`, `nodes`, `channels`,
  `expires_at`, `max_version`, `label`, `issued_by`, and `revoked_at`. The portal builds
  the token from this record and never stores it.
- **`activation`**: `license`, `fingerprint`, `first_seen`, `last_seen`, `released_at`,
  the machine's name, and the hash of its Desktop renewal secret.
- **`event`**: An append-only audit log of issue, activate, renew, release, rename,
  transfer, and revoke.

Resend sends the mail Clerk does not: expiry warnings at 30, 7, and 1 days, and
revocation notices. A daily Vercel Cron job runs the expiry sweep.

`infra/portal/` declares in Terraform the KMS signing key, the IAM identity the Vercel
runtime signs with, the portal's Vercel environment and domain, and the CI secret, with
state in HCP Terraform. Neon and Clerk come through the Vercel Marketplace, which
injects their variables. Their dashboard steps, and Resend, which has no provider, are
documented in `infra/README.md`.

### 5.7 Portal licenses and the activation ledger

A license opens to its activation ledger: the machines that hold seats, by name, when
each last received a token, and a release action that frees the seat. A released machine
can activate again, which is how a hardware change is handled.

Issuing a token is one endpoint, `POST /api/licenses/:key/activate`, which takes a
fingerprint and a machine name. The name is required, because activation is the only
moment anyone knows which box a set of hashes belongs to.
`POST /api/activations/:key/name` renames the machine later. The endpoint checks that
the caller is a member of the owning organization, that the license is neither revoked
nor expired, and that a seat is free or this fingerprint already holds one. It then
records the activation and the event and returns the signed token. It is rate limited
per caller and per organization.

Staff issue every enterprise license, trials included: the organization, machines,
channel cap, label, and term. A subscription takes an expiry and an optional fallback
version, and a perpetual license takes a maximum version. A trial is a short
subscription. Nothing is self-serve, and nothing takes payment.

### 5.8 Desktop sign-in

Desktop follows RFC 8252 with a custom scheme of its own, `synnax-desktop://`,
registered in `tauri.desktop.conf.json`. The Console keeps `synnax://`, so the two apps
never claim each other's links.

The app reads the fingerprint from its embedded Core, creates a one-time `state` value,
and opens `portal.synnaxlabs.com/desktop/sign-in?state=&fp=&name=&v=` in the system
browser, where `name` is the hostname and `v` the app version. After the user signs in,
the page calls `POST /api/desktop/link`, which issues a desktop license to the user's
personal organization, records the activation, creates an opaque renewal secret, and
returns the token and the secret. The page opens
`synnax-desktop://activate?state=&token=&secret=`, with an Open Synnax Desktop button
for a browser that blocks the navigation. The app refuses a link whose `state` it did
not create, activates the token on the embedded Core, and keeps the secret in its
session. No code exchange exists: the token binds to the host, so a captured link
licenses nothing else.

Each machine holds its own desktop license: a subscription for one machine, with no
channel cap and no maximum version. A machine that signs in again replaces its earlier
link: the earlier activation with the same host hash is released and its license
revoked. On launch and every six hours while online, the app renews a license that nears
expiry: `POST /api/desktop/renew`, with the secret as a bearer token, extends the expiry
and returns a fresh token that the app activates. Unlinking the machine in the portal
revokes the license, so the next renewal is refused and the license lapses at expiry
plus the grace window. Expiry notices skip the desktop edition.

In Desktop, the gate shows a sign-in screen in place of the enterprise activation
screen: one line and a Sign in button, a waiting state while the browser is open, an
offline state with Try again, and Use a license file, which opens the paste-and-file
screen. Its messages use plain words ("Sign in to continue") per RFC 0063 §5.5. The app
has no sign-out. Switching accounts is Unlink in the portal or Erase all data.

### 5.9 Development, CI, and hosted Cores

Every build enforces, so development uses real licenses from the Synnax Labs
organization:

- **Engineer machines**: Each engineer activates once with a one-year internal license.
- **CI**: Runners have random hardware, so staff issue a 90-day floating license and
  store it as the GitHub secret `SYNNAX_LICENSE_TOKEN`, which the workflows pass as
  `SYNNAX_LICENSE_KEY`. The job that runs a released Core keeps the old key secret. The
  floating token is the one license that is a secret.
- **Before the portal deploys**: The portal's signing module runs as a local script
  against the KMS key, so staff can sign the engineer and CI tokens first.
- **Hosted Cores**: The demo Core behind the docs live plot, and every other Core Synnax
  runs, hold machine-bound internal licenses.
- **Go tests**: `core/pkg/service/mock` opens a service layer with a throwaway key pair
  and a signed license, so only the license package's own tests deal with licensing.

### 5.10 Portal interface

The portal is its own shell on its own host. The docs header carries one Sign in button
to it.

The shell puts the scope first. The top bar holds the logo, the scope, a Docs link, and
an avatar menu with Settings and Sign out. A team member gets a scope switcher, and a
personal user sees only their name. A tab bar holds the scope's sections, and every
scope opens on Overview:

- **Personal**: Overview, with Download Synnax Desktop and Sign in from the app, the
  recent machines, and one Enterprise panel with Talk to us. Devices, with each linked
  machine's license status, Rename, and Unlink.
- **Team**: Overview, with usable licenses, seats in use, and the next expiry. Licenses,
  and Members, where admins invite, remove, and change roles through Clerk.
- **Staff**: Admin, in every scope: every license, searchable and filtered by status,
  with Expiring as the renewal queue and Issue license as a dialog.

A license reads Active, Expiring (within 30 days of expiry, never for Desktop), Expired,
or Revoked. Its page shows the facts, the machines with Download token, Rename, and
Release, the released machines, and the activity from the event log. Staff also get
Edit, which changes the terms in place so seats and history survive a renewal, Floating
token, and Revoke. Activate a machine is a dialog that takes a name and the fingerprint
the Console copied, and downloads the token. `/licenses/activate?license=` is the page
the Console links.

Routes sit at the root of the host: `/`, `/devices`, `/licenses`, `/licenses/<key>`,
`/licenses/activate`, `/members`, `/settings`, `/admin`, and `/api/...`. The scope is
the `?org=` query parameter, so a license URL survives an organization rename.

Each page is an Astro page that loads its data on the server and renders one React
island with it. Actions are Pluto `Modal` dialogs that post JSON to the `/api/...`
routes, show a route error inline, and reload the page on success. Sign-in, sign-up,
password reset, and the OAuth callback are custom pages on Clerk's client API: email and
password with an email code, Google, and Microsoft. The theme follows the operating
system, and below the mobile breakpoint the tab bar scrolls and tables become cards.

## 6 What this RFC does not cover

- Support. The docs feedback form stays on Formspree.
- Building Synnax Desktop itself, which RFC 0063 covers. This RFC defines the license
  Desktop runs under.
- Payments and self-serve purchase.
- The privacy policy, which changes alongside the portal, outside this RFC.
- Integration test runner infrastructure.

## 7 Implementation phases

The work ships as a stack of five pull requests into `main`.

- **Phase 1: Core.** The license schema and its generated types, the `license` service
  with the anchors, the fingerprint, the clock mark, and the term checks, the API
  operations and the gate, the start flags, the channel limit, the `service/mock`
  fixture, and CI on the new secret. Deletes `channel/verification` and the 50-channel
  tier.
- **Phase 2: Clients.** The TypeScript and Python `license` modules, the typed errors in
  every client, the `unlicensed` connection reason, and the Driver retry.
- **Phase 3: Console.** The activation screen and guard, the Core badge, and the version
  modal.
- **Phase 4: Portal.** `site/portal/` and its Vercel project, `infra/portal/`, Clerk,
  Neon, and Resend, the four tables, KMS signing, the activation endpoint, the staff
  area with all three terms, the expiry cron, and every page of §5.10. The Console's
  activation link moves to the portal. Before it deploys, production Clerk needs the
  name attribute, organizations, and the Microsoft connection, and Neon needs the
  Drizzle migration, which neither the build nor the deploy runs.
- **Phase 5: Desktop sign-in.** The `synnax-desktop` scheme, the sign-in page and its
  link route, the renew route, the Desktop sign-in screen and deep link handler, the
  account slice, and the renewal loop. Neon needs the migration that adds the two
  activation columns.

### 7.0 Compatibility

The old key format is dropped without a grace window. Before the release that carries
Phase 1, staff issue licenses through the staff area: a subscription license for every
subscription holder, a perpetual license for the existing perpetual customer with the
maximum version their contract sets, and internal licenses for the demo Core, the
engineer machines, and CI. The release notes state that the standalone Core needs a
license and link the activation page.

A Core given an old-format key at start refuses it and does not start, so a customer
replaces the key with a token when they upgrade. The Core deletes the old key's stored
entry. Clients from before this release see a generic error from an unlicensed Core, and
new clients decode it.

## 8 Resolved decisions

1. **Organizations own every license**: A personal organization per user avoids an owner
   that is sometimes a user and sometimes an organization, which would fork every query
   and permission check.
2. **One signed offline token**: Online validation makes a running Core depend on our
   uptime and rules out air-gapped sites. Keygen would replace the signer but not the
   portal, the verifier, or the identity model.
3. **A standard JWT signed with ML-DSA**: `kid` gives key rotation, and the three-part
   frame is a published format any language can build. A forged token buys nothing that
   patching public source does not, by principle 3, but a post-quantum signature settles
   the question in security reviews. The trade is real: a token grows from about 500
   characters to about 3,700.
4. **Claims evolve additively**: New claims are optional, restrictive ones also go in
   `required`, and `claims_version` changes only for a break. An older Core keeps
   working on a newer token and never honors less than the token demands.
5. **No download wall**: The Tauri updater, `pip`, and `docker pull` cannot be gated,
   and sign-in and activation give the counts a wall would. The trade is real: nothing
   stops a direct GitHub link.
6. **The standalone Core requires a license and starts without one**: Desktop is the
   free edition. Refusing to start would block activation through the Core's own web
   Console.
7. **Every build enforces**: A key injected only at release would make any clone an
   unlimited Core. The cost is one activation per engineer machine and a rotating CI
   secret.
8. **A MAC-derived fingerprint set with an activation ledger**: Ignition's shape, a
   hardware fingerprint plus a server-side count with reactivate. Matching by
   intersection, FlexNet's rule, survives adapter changes. Keygen's per-boot fingerprint
   needs an online heartbeat.
9. **Staff issue every license and create every team**: There is no self-serve trial, so
   a team a customer made would stay empty until a deal closes. The trade is real:
   onboarding is a staff step.
10. **Hard cutover from the old key**: A grace window would keep the forgeable parser
    alive for a release, and the holder list is small and known.
11. **Private key in AWS KMS**: An environment secret is one leak away from unlimited
    licenses until a release ships new anchors. The trade is real: a second cloud
    account and a network call per issuance.
12. **Clerk backs only team organizations**: Clerk bills per retained organization, and
    Desktop makes every free user one. Clerk ids stay out of the token, so the vendor
    stays replaceable.
13. **Time is checked at start, never against a serving Core**: Grafana keeps running on
    an expired license with a banner, and a stop mid-run costs a test operator the test.
    The trade is real: a process can outlive its license until it restarts.
14. **A clock that was behind recovers without a restart**: Devices without a
    battery-backed clock boot in the past and sync minutes later. Checking every minute
    while behind unlocks them and keeps the rollback check whole.
15. **Perpetual licenses are first-class, capped by minor version**: An existing deal is
    perpetual. The repository ships one shared minor version, so patch releases stay
    covered and no build timestamp is involved.
16. **License state on the connectivity check**: The Console's synchronizers hit gated
    endpoints as soon as the change stream opens, so the connection must know first, and
    the check is the call it already makes. The trade is real: one word of state is
    readable without credentials.
17. **The gate is a transport roster**: The request target is a path on HTTP and a
    method on gRPC. The roster already exempts login and the connectivity check, and one
    list shows what is gated.
18. **Access through the `builtin` object**: The built-in roles already grant `builtin`,
    so Owners activate and Engineers read with no new resource type and no ontology
    migration.
19. **Plain `license` names in the Core**: The repository is public, so hidden
    identifiers only cost readers. GitLab, Elastic, and CockroachDB ship readable
    enforcement code, and principle 3 sets the bar at a source edit.
20. **No CLI**: A `synnax license` command needs a Go client that speaks TLS, and none
    exists. Start flags, the Console, and `curl` cover every case.
21. **The Driver retries the gate error**: Its loops retry only an unreachable Core, so
    an unlicensed Core would stop the Driver it starts.
22. **The portal is its own package, deployment, and shell**: Served from the docs,
    every account change shipped with the docs, and the docs carried Clerk, Neon, and
    KMS settings. Vercel, Linear, and GitHub all give the signed-in surface its own
    shell. The trade is real: two Vercel projects and a second layout.
23. **Sign-in screens are ours, on Clerk's client API**: Clerk's components take an
    appearance object, not a design. The trade is real: reset, verification, second
    factors, and OAuth callbacks are our pages to maintain.
24. **Members act on licenses, admins act on the team**: The engineer at the test stand
    needs a token without asking a manager. The trade is real: any member can release
    another member's machine, and the event log is the recourse.
25. **A custom scheme and no code exchange for Desktop**: A loopback redirect needs an
    HTTP listener in the shell, and some managed browsers block it. The token binds to
    the host, so a captured link licenses nothing. The trade is real: `tauri dev` on
    macOS activates through the file fallback.
26. **One renewing license per Desktop machine**: A long term without renewal would make
    Unlink do nothing for a year. One license per machine reuses issue, activate,
    revoke, and the ledger unchanged. The trade is real: the renewal secret sits on
    disk.
27. **Desktop is unlimited and has no sign-out**: A channel cap would not sell the
    enterprise edition (RFC 0063 §8) and would be the first wall an evaluator hits. The
    Core has no operation to drop a token, so a sign-out could only stop renewal, and
    Unlink and Erase all data already cover it.
