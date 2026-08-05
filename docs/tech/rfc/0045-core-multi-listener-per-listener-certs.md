# 45 Serving Core on multiple listeners with per-listener certificates

- **Author**: Emiliano Bonilla
- **Date**: 2026-07-13
- **Related**: [RFC 0042 - Core Structure Refactor](./0042-core-structure-refactor.md),
  [SY-4456 - Serve Core on Multiple Ports with Unique Certs](https://linear.app/synnax/issue/SY-4456)

## 0 Summary

A Synnax Core node opens exactly one TCP socket and fronts it with exactly one TLS
certificate. Every kind of traffic (hardware drivers, Console users, and node-to-node
cluster gossip) arrives on that port and is proven by that certificate. This RFC lets a
node serve multiple listeners, each bound to its own address and presenting its own
certificate.

The design comes from an enterprise customer: drivers should reach a node at a corporate
address like `core01.example.com:9090` with the customer's production PKI certificate,
while Console users reach the same node at `node01.<tailnet>.ts.net:9091` over Tailscale
with a Tailscale-issued certificate. Routing users through Tailscale lets the customer
enforce per-user network ACLs instead of a broad subnet-wide rule.

It generalizes that narrow request into a sustainable model:

1. A **listener** is a first-class concept: one bound socket, one TLS identity, its own
   protocol multiplexer. A node runs one or more.
2. Every listener serves the **full** API surface. Listeners differ only by address and
   certificate, not by which services they expose.
3. Each listener's certificate comes from an independently-chosen **source**: `file`,
   `auto`, or `tailscale` in v1, freely mixed across listeners.
4. Configuration reuses the existing `listen` key with a **polymorphic** shape: a scalar
   address (today's behavior) or a list of listener objects. Backwards compatibility is
   the scalar case, not a special rule.
5. `insecure` remains a global server mode. Client-certificate mTLS is deferred, but the
   per-listener security type is shaped to accept it without restructuring.

---

## 1 Motivation

### 1.0 The request

Paraphrased from an enterprise customer's IT team:

> The ability to serve the Core on multiple ports with a different cert per port.
> Hardware drivers connect to the Core on a corporate address and port with the
> corporate PKI cert. Console users connect on a separate port at a `ts.net` address,
> which forces them to go through Tailscale and allows per-user ACLs rather than a broad
> subnet-wide ACL.

"Different cert per port" bundles two goals:

1. **Distinct TLS identity per network path.** The driver and user paths present
   certificates from different trust chains (a corporate PKI cert versus a Tailscale
   Let's Encrypt cert for a `ts.net` name).
2. **Network-path segregation for access control.** The user path lands on the Tailscale
   interface, so Tailscale can gate _which_ users may reach it at the network layer.
   This is the goal the customer actually cares about, and it is why the ports must be
   genuine separate sockets on separate interfaces, not one socket serving two names.

### 1.1 Why the current architecture cannot do this

A node creates one listener and one certificate:

- `core/pkg/server/server.go` calls `net.Listen("tcp", s.ListenAddress.PortString())`
  once, then fans that socket into logical branches (gRPC, HTTP, redirect) with
  `cockroachdb/cmux`.
- `core/pkg/security/secure.go` builds one `*tls.Config` whose `GetCertificate` callback
  returns a single loaded node certificate and ignores the requested name.
- The listen address is a single value (`--listen`, default `localhost:9090`), and
  certificate configuration is a separate, global flag group (`--certs-dir`,
  `--node-cert`, `--node-key`) with no association to any address.

"Where I listen" and "what certificate I present" are unrelated, singular config. There
is no seam to introduce a second address with a second certificate.

### 1.2 Precedent

CockroachDB, which uses the same `cockroachdb/cmux` multiplexer Synnax vendors, splits
its traffic across separate ports (`--listen-addr` for RPC and inter-node join,
`--sql-addr` for client SQL, `--http-addr` for the admin UI). Its reasons match this
request: customers want their own TLS PKI for client traffic distinct from inter-node
traffic, and separate ports enable defense in depth by letting operators firewall the
inter-node port to peers only. CockroachDB deprecated its combined RPC+SQL port because
one shared port "makes the TLS security story difficult." Synnax is in that
combined-port position today.

---

## 2 Goals and non-goals

### 2.0 Goals

1. A node can serve N listeners, each with an independent bind address and certificate.
2. Certificate sources `file`, `auto`, and `tailscale` are supported and freely mixed.
3. Existing single-listener deployments are unaffected.
4. The architecture leaves additive seams for later work (client mTLS, an `acme` source,
   per-listener service exposure, per-listener plaintext).

### 2.1 Non-goals (v1)

1. **Per-listener service exposure.** Every listener serves the full API. Choosing which
   services a listener exposes (the CockroachDB RPC/SQL/HTTP split) is deferred.
2. **Client-certificate mTLS.** The server proves its identity per listener; clients
   keep authenticating with the existing token mechanism. The type design reserves a
   seam for it (Section 4.6).
3. **The `acme` certificate source** (direct Let's Encrypt for a public DNS name,
   without Tailscale).
4. **Per-listener insecure/plaintext.** `insecure` stays a whole-server mode.
5. **Bridging Tailscale identity into Synnax authorization.** Tailscale gates network
   reachability; Synnax RBAC gates operations. The layers stay orthogonal.

---

## 3 Current architecture

A single socket is multiplexed into branches. In secure mode the root `cmux` peels
plaintext HTTP/1 (for the HTTP-to-HTTPS redirect) off the front, wraps the remainder in
a `tls.NewListener` using the one `*tls.Config`, and runs a second `cmux` inside the
TLS-terminated stream to separate gRPC (HTTP/2) from HTTPS/1 API and Console traffic.
Inter-node Aspen and distribution transports register as gRPC services on that branch,
so cluster gossip shares the client port.

```
              net.Listen(":9090")            one *tls.Config, one cert
                     |                        (GetCertificate ignores SNI)
              root cmux
             /            \
   HTTP/1 (redirect)     TLS listener
                              |
                         secure cmux
                        /           \
                   gRPC (h2)      HTTPS/1
              API + gossip      API + console
```

The existing abstraction is `Branch` (`core/pkg/server/branch.go`): each branch receives
a `net.Listener` derived from the shared socket. Nothing above the branch owns a socket
or a certificate. That is the concept this RFC introduces.

---

## 4 Design

### 4.0 The listener

A **listener** sits one level above `Branch` and owns:

- one bind **address** (`host:port`), where `host` may be `0.0.0.0`, a specific
  interface IP, or the node's Tailscale IP (binding to an interface is itself an access
  control),
- one **certificate source** (Section 4.1) backing its own `*tls.Config`,
- its own `cmux` tree, reusing the existing secure/insecure multiplexing,
- optionally a **name** (for logs) and an **advertise** marker (Section 4.4).

`Server.start` changes from "listen once, build one `cmux` tree" to "for each listener:
listen, build its own `cmux` tree." Branches are instantiated per listener. Since every
listener serves the full surface (Section 2.1), the branch set is identical; only the
address and `*tls.Config` differ.

Per-listener certificates need no special mechanism: each listener's
`tls.Config.GetCertificate` points at its own source. SNI selection still works within a
listener, but the driver and user paths are separated by distinct sockets on distinct
interfaces, not by SNI.

### 4.1 Certificate sources

A certificate source answers the standard Go TLS callback. The interface matches
`crypto/tls`'s `GetCertificate` exactly, so a source plugs straight into a listener's
`*tls.Config`:

```go
type Source interface {
    GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}
```

v1 implementations:

1. **`file`.** Loads a certificate and key from PEM files, as today's node certificate
   does. It **hot-reloads**: because `GetCertificate` runs per handshake, the source
   re-reads on change and swaps the certificate live, so rotation needs no restart or
   cluster bounce.
2. **`auto`.** Self-signs from Synnax's built-in CA, as `--auto-cert` does today. Each
   `auto` listener derives its SANs from its own address. For development and
   self-contained on-prem deployments.
3. **`tailscale`.** Wraps `tailscale.com/client/tailscale.LocalClient.GetCertificate`,
   which provisions and renews the `ts.net` certificate via Let's Encrypt (DNS-01). The
   Tailscale method _is_ the callback, so there is no glue and no certificate paths in
   config. Requires a running `tailscaled` with HTTPS enabled on the tailnet.

Sources are chosen per listener and freely mixed; the motivating deployment runs `file`
on the driver listener and `tailscale` on the Console listener. `auto` and `tailscale`
manage their own renewal, so `file` is the only source needing explicit hot-reload.

#### 4.1.0 The source space, and why `tailscale` is first-class

`Source` is an open extension point, so the built-in set is a choice, not a limit.
Nearly every enterprise certificate mechanism reduces to one of four shapes:

- **static** (`file`): Any agent that writes a PEM to disk: cert-manager, a Vault agent,
  a `tailscale cert` cron, a cloud secret-store sync. With hot-reload, `file` absorbs
  this whole class.
- **self-signed** (`auto`): Synnax's own CA.
- **an authority the node talks to**: Public or private ACME, Vault PKI, SPIFFE SVIDs,
  EST/SCEP/CMP; a native source buys automatic renewal inside Core.
- **a hardware-held key** (`pkcs11`/HSM/TPM): The one shape that cannot collapse to
  `file`, because the key must never exist as a file.

So only a hardware-key source _requires_ being native; everything else, `tailscale`
included, can be delegated to `file` plus an external renewer. First-classing a source
is a demand-and-UX call, not a capability gap.

`tailscale` earns its slot despite being file-collapsible (`tailscale cert` writes PEMs
a `file` source would serve): it is the blessed path for the motivating deployment and
materially better. `LocalClient.GetCertificate` is one line, needs no filesystem or
write permissions, and lets `tailscaled` own DNS-01 provisioning and renewal. `acme`,
`vault`, and `spiffe` are the natural next additions; cloud secret stores and legacy
enrollment protocols stay `file`-delegated until asked for.

### 4.2 Security package: boundaries and responsibilities

Today one `secureProvider` conflates three responsibilities:

1. **Owning the node certificate** (`secureProvider.tls`, loaded once via
   `cert.Loader`).
2. **Certificate selection** (`getCert`, the callback, which ignores the `ClientHello`).
3. **Assembling the TLS config** (`TLS()`, fixing cipher suites, `MinVersion`,
   `NextProtos`, the CA pool, and `ClientAuth`, then wiring the callback in).

One `Provider` produces one `*tls.Config` for the whole node. Multi-listener breaks
this: responsibility 2 must vary per listener while the policy in 1 and 3 stays
node-wide. The refactor splits the three along that seam.

**`cert.Source` isolates certificate selection.** It is responsibility 2, lifted out of
`secureProvider` and named (Section 4.1); `getCert` becomes the `file` source.
Implementations live in `security/cert`, with `tailscale` in its own subpackage
(`security/cert/tailscale`) so its dependency never enters the base package. A factory
maps the `source` string to a constructor, so adding a source is a registration, not a
server edit.

**TLS policy stays node-wide.** Cipher suites, `MinVersion`, `NextProtos`, the CA pool,
and the default `ClientAuth` are node security policy, identical on every listener. A
listener's `*tls.Config` is assembled as _policy + this listener's source_ by a small
helper.

**`Provider` narrows.** It stops owning the cert and becomes the policy holder and the
factory that turns a source into a `*tls.Config`. The secure/insecure switch
(`newSecureProvider` / `newInsecureProvider`) stays node-wide (Section 4.5).

**`ProviderConfig` splits by scope.** The node-wide fields (`Insecure`, `KeySize`,
CA/policy config) stay; the per-certificate paths move into the `file` source's config,
one per listener. In the scalar case the existing `--certs-dir` / `--node-cert` /
`--node-key` flags populate the default listener's `file` source, so nothing observable
changes.

**Outbound node identity is not per-listener.** `Source` governs the _inbound_
certificate a listener presents. The node's own identity when dialing peers
(`getClientCertificate`, `NodePrivate`) is one node-wide identity and stays on
`Provider`, so cluster identity is never tied to whichever listener a client used.

| Concern                                           | Today                                 | Target                                    | Scope        |
| ------------------------------------------------- | ------------------------------------- | ----------------------------------------- | ------------ |
| Cert selection callback                           | `secureProvider.getCert`              | `cert.Source` (`file`/`auto`/`tailscale`) | per listener |
| Cert loading from disk                            | `cert.Loader` in `secureProvider`     | the `file` source                         | per listener |
| Ciphers / versions / CA pool / default ClientAuth | `TLS()` literal                       | node TLS policy in `security`             | node-wide    |
| `*tls.Config` assembly                            | `Provider.TLS()`                      | `policy + source` helper                  | per listener |
| Secure vs insecure                                | `NewProvider` switch                  | unchanged                                 | node-wide    |
| Outbound / client identity                        | `getClientCertificate`, `NodePrivate` | unchanged on `Provider`                   | node-wide    |

`security/cert` stays ignorant of listeners and the server, and the server layer
consumes sources plus policy without knowing how a certificate is obtained.

### 4.3 Configuration surface

`listen` becomes **polymorphic**: either a scalar address or a list of listener objects.

Scalar form, identical to today:

```yaml
listen: localhost:9090
```

List form (the motivating deployment: drivers on a corporate address, Console users over
Tailscale):

```yaml
listen:
  - address: core01.example.com:9090
    cert:
      source: file
      cert: /usr/local/synnax/certs/driver.crt
      key: /usr/local/synnax/certs/driver.key
    advertise: true
  - address: node01.example-tailnet.ts.net:9091
    cert:
      source: tailscale
    name: console

peers: [core02.example.com:9090, core03.example.com:9090]
```

```bash
synnax start --config /etc/synnax/synnax.yaml
```

Drivers reach `:9090` with the production certificate; Console users reach `:9091` over
Tailscale with the auto-provisioned `ts.net` certificate; nodes gossip via the
advertised `core01.example.com:9090` address; Tailscale enforces per-user ACLs on the
Console path.

Design rules:

1. **One axis of polymorphism.** `listen` may be a scalar or a list; list items are
   always objects, never bare address strings. This bounds the parsing surface but needs
   a hand-written `UnmarshalYAML`/`UnmarshalJSON` accepting `string | []Listener`.
2. **The config file is the home for multi-listener.** A flag cannot express a list of
   rich objects. `--listen` stays a scalar-string flag building the single default
   listener; a `listen` list in config takes over.
3. **Certificate flags are scalar-only.** `--certs-dir`, `--node-cert`, `--node-key`,
   and `--auto-cert` configure the single default listener. Combining them with a
   `listen` list is a **startup error**, not a silent precedence rule, since a silent
   winner could load the wrong certificate. In the list form every certificate lives in
   a per-listener `cert` block.

Mapping from today's flags:

| Today (global)                               | New (per-listener)                             |
| -------------------------------------------- | ---------------------------------------------- |
| `--listen ADDR`                              | scalar `listen`, or one `listen[].address`     |
| `--node-cert` / `--node-key` / `--certs-dir` | `listen[].cert.{cert,key}` with `source: file` |
| `--auto-cert`                                | `listen[].cert.source: auto`                   |
| (new)                                        | `listen[].cert.source: tailscale`              |
| `--peers`                                    | unchanged (cluster peers)                      |
| (implicit: the sole listener)                | `listen[].advertise: true`                     |

### 4.4 Cluster advertise address

Gossip is served on every listener (Section 2.1), but peers must dial one agreed
address. Exactly one listener is **advertised**:

1. **Scalar `listen`** or **a list with no `advertise: true`**: the sole (or first)
   listener.
2. **A list with one `advertise: true`**: that listener.
3. **A list with more than one `advertise: true`**: startup error.

Defaulting to the first keeps the simple case terse; erroring on multiple prevents a
misconfiguration that could split the cluster by advertising an unreachable listener.

### 4.5 Insecure mode

`insecure` stays a **global** switch. It disables encryption, authentication, and
authorization together, a whole-node development mode rather than a per-socket TLS
toggle; making it per-listener would either change that meaning or create a confusing
hybrid. So:

- `--insecure` set: the node runs open; certificate sources are ignored.
- `--insecure` unset: every listener must resolve a certificate source, or startup
  fails.

Per-listener plaintext (e.g. one listener behind a terminating proxy) is a future field.

### 4.6 mTLS seam (deferred)

Client-certificate mTLS is out of scope for v1, but two properties keep it additive:

1. Each listener owns its `*tls.Config`, and requiring client certificates is a
   per-config setting (`ClientAuth`, `ClientCAs`). A future
   `listen[].cert.clientAuth: require` with a per-listener client CA is a field
   addition, not a restructuring.
2. The per-connection identity plumbing already recovers client certificates at the
   service layer (`freighter/go/grpc/tls.go` `MuxCredentials` lifts the `*tls.Conn` out
   of the muxed connection). It becomes per-listener for free once listeners own their
   config.

The eventual shape: the driver listener requires machine certificates while the Console
listener relies on Tailscale plus Synnax login.

---

## 5 Validation rules

Enforced at startup, failing fast with a clear message:

1. At most one listener may set `advertise: true`.
2. In secure mode, every listener must resolve a certificate source.
3. A `file` source must specify both `cert` and `key`.
4. An `auto` or `tailscale` source must not specify `cert`/`key`; the paths would be
   ignored, so reject them to avoid confusion.
5. Listener addresses must be unique.
6. A `listen` list must be non-empty and must not be combined with any global
   certificate flag (Section 4.3).

---

## 6 Backwards compatibility

The scalar `listen` case is the compatibility path, not a special branch. A node with no
`listen` list (scalar `--listen` plus the existing certificate flags) builds one
listener and behaves as today; all flags keep their meaning and defaults; the
`--decoded` whole-config blob carries the polymorphic `listen` like any other field. No
migration.

---

## 7 Implementation plan

Staged so the first phase is a pure refactor.

**Phase 1 - Listener abstraction (no new behavior).** Introduce the listener above
`Branch`; refactor `Server.start` to loop over a listener set, each with its own
`net.Listen`, `cmux` tree, and `*tls.Config`. Extract the `cert.Source` interface,
backed by `file` and `auto` sources factored out of `secureProvider`. `--listen` builds
one default listener; single-listener deployments are unchanged.

**Phase 2 - Multi-listener configuration.** Add the polymorphic `listen` parsing,
per-listener `cert` blocks, the `advertise` marker and its resolution, and startup
validation (Section 5). Route the advertised address into the cluster join config.

**Phase 3 - Tailscale source and `file` hot-reload.** Add the `tailscale` source and
`file` hot-reload (Section 4.1). The `file` source loads once in Phases 1 and 2,
matching today; live swapping is a deliberate behavior change and lands only here.
Completes the motivating requirement.

Later, out of scope: `acme` source, per-listener `clientAuth`, per-listener service
exposure, per-listener plaintext.

---

## 8 Risks and gotchas

1. **Cluster join misconfiguration.** Advertising a listener peers cannot reach (e.g.
   the Tailscale one) prevents the cluster forming. Mitigated by the advertise rules.
2. **Privileged bind for port 443.** Low ports need `cap_net_bind_service` or root;
   document per platform.
3. **Tailscale prerequisites.** The `tailscale` source needs `tailscaled` with HTTPS
   enabled; first-request DNS-01 provisioning has latency, so consider warming on boot.
4. **`auto` SANs.** Each `auto` listener's certificate must carry SANs for its own
   address, not a global host list.
5. **Polymorphic parsing.** The scalar-or-list `listen` needs a hand-written unmarshaler
   with clear validation errors, bounded by the single polymorphic axis (Section 4.3).

---

## 9 Future work

1. **Per-listener service exposure.** Let a listener choose which services it exposes
   (client API, Console assets, gossip), mirroring the CockroachDB split; keeps gossip
   off client-facing listeners.
2. **Client mTLS.** Per-listener `clientAuth` with a per-listener client CA (Section
   4.6).
3. **`acme` source.** Direct Let's Encrypt for public DNS names without Tailscale.
4. **Per-listener plaintext.** A listener behind a terminating proxy while the node
   otherwise runs secure.
