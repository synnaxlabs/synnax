# 73 Authentication

- **Author**: Patrick Dotson
- **Date**: 2026-10-01
- **Related**: [RFC 0009 - Encrypting cluster communications](0009-encryption.md),
  [RFC 0043 - Oracle support for struct unions](0043-oracle-struct-unions.md),
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md),
  [PR #3038 - SY-4952: Key credentials by user UUID and move the username onto them](https://github.com/synnaxlabs/synnax/pull/3038)

## 0 Summary

Synnax authenticates one thing in one way: a user, with a username and a password. The
Driver logs in as a person, the embedded Driver holds the root password in a file, a
rack has no identity of its own, and a token cannot be revoked.

This RFC separates three ideas that the code joins today. A subject is the user or rack
record that logs in. A credential is a stored record that points at one subject and
holds the data for one method: a password, an API key, or an account at an OpenID
Connect provider. A session is a stored record that a login creates. Every method
exchanges its proof for a session at one login call, and every request carries only the
opaque session token.

The authentication service never learns what a user or a rack is, and it reaches each
method through one small interface. A new subject type or a new method therefore adds no
code to the service. Access control and audit are out of scope: authentication hands
them one subject ID, as it does today.

## 1 Motivation

- **The Driver is not a person**: The Driver logs in with a user's username and password
  (`driver/cmd/login.cpp:47-48`) and saves the password to a file that every user on the
  machine can write (`driver/rack/persist.cpp:17-22`). The embedded Driver receives the
  root credentials of the Core (`core/cmd/start/start.go:375`).
- **A rack has no identity**: The Driver stores a rack key and claims it at start
  (`driver/rack/remote.cpp:28-50`). Any user who can write racks can claim any rack.
- **One method**: The service checks a bcrypt hash and nothing else
  (`core/pkg/service/auth/service.go:90-113`). Customers ask for single sign-on, API
  keys, and smart cards.
- **No revocation**: A token is a signed JWT that the Core does not store
  (`core/pkg/service/auth/token/token.go:106-131`). The token of a deleted user works
  for up to 24 hours (`core/pkg/service/layer.go:431-435`). No logout endpoint exists.
- **Clients keep the password**: The TypeScript and C++ clients hold the plaintext
  password for their whole life and log in again when the token expires
  (`client/ts/src/auth/auth.ts:118-171`, `client/cpp/auth/auth.h:94-120`). The Console
  saves it to disk (`console/src/session/core/slice.ts:17-22`).
- **The username is the primary key**: A rename is a delete and a create
  (`core/pkg/service/auth/writer.go:67-73`), and the username lives in two tables.

## 2 Vocabulary

- **Subject**: The record that logs in, named by an `ontology.ID`: a type and a key. A
  user and a rack are subjects.
- **Method**: A way to prove identity: `password`, `api_key`, or `oidc`.
- **Credential**: A stored record that binds one subject to the data of one method.
- **Identifier**: The public value that finds a credential, unique within its method: a
  username, an API key ID, or an issuer and account at a provider.
- **Proof**: What a client presents at login: a username and password, an API key, or
  the code that a provider returned.
- **Authenticator**: The code that checks the proof of one method.
- **Session**: A stored record that a login creates. It names the subject and the
  credential.
- **Session token**: The opaque secret that a client sends on each request to name its
  session.
- **Provider**: An external OpenID Connect identity provider, such as Microsoft Entra,
  Google, or an on-site Keycloak.

NIST SP 800-63-4 calls a password or key an authenticator and calls the binding record a
credential. This RFC uses "credential" for the stored record and "authenticator" for the
code, which matches Kubernetes and Grafana.

## 3 Background

Lampson's "gold standard" names three mechanisms around every guarded operation
(Computer Security in the Real World, 2000):

- **Authentication**: Who made this request?
- **Authorization**: Is that principal permitted to do this operation on this object?
- **Audit**: A record of what the guard decided.

The three share one value: the principal. Authentication produces it, authorization
decides on it, and audit records it. NIST SP 800-53 AU-3 requires each audit record to
carry the identity of the subject, so the three must agree on one identifier. In Synnax
that identifier is the subject ID, and the seam already exists: the middleware sets it
(`core/pkg/api/auth/middleware.go:40`) and about 30 API files pass it to access control
in an `access.Request` (`core/pkg/service/access/enforcer.go:14-18`).

Other points from the standards that shape this design:

- **People and machines are different problems**: NIST SP 800-63-4 covers natural
  persons only. SP 800-53 covers devices in IA-3 and services in IA-9. Session timeouts
  exist to prove that a person is still present.
- **Operational technology is an exception**: NIST SP 800-82r3 says session termination
  can be tailored where "immediate operator response is required", and that automatic
  logoff "can be detrimental to the operation of the OT system".
- **Defense sites want smart cards**: DoDI 8520.03 prefers the certificate on a Common
  Access Card. A smart card login is TLS client authentication plus a map from the
  certificate to an account, and identity providers such as Keycloak and Microsoft Entra
  do both.
- **A random key needs only a fast hash**: A password needs a slow hash because a person
  chose it. A key with 128 bits or more of entropy is safe under SHA-256.

## 4 Prior art

- **One interface per method, one principal out**: Kubernetes runs a set of
  authenticators that each return `user.Info`. Grafana runs `authn.Client`
  implementations that each return `authn.Identity`. We align.
- **The credential is a separate record**: Ory Kratos stores a type, identifiers, and a
  private config for each credential and requires the identifier to be unique within its
  type. Vault stores an alias for each method. We align with Kratos. Kubernetes and
  CockroachDB derive the identity from the credential itself, which makes it hard to
  revoke one credential.
- **Every method ends in one internal token**: Vault, Teleport, Microsoft Entra, and the
  Docker Registry exchange every proof for one short-lived credential. GitHub, Grafana,
  Stripe, and InfluxDB accept an API key directly on each request. We align with the
  first group, because one artifact on the request path gives one check and one place
  for revocation.
- **Stored sessions revoke at once**: Vault service tokens, Grafana sessions, and the
  CockroachDB Console keep the session in the database. Keycloak moved its sessions from
  memory to the database. A signed token with no stored state, as in Entra, revokes only
  when it expires.
- **Live connections**: OpenSSH, PostgreSQL, etcd, and the defaults of Kafka and
  Teleport check at open only. NATS and EMQX close at expiry. OPC UA keeps a session
  alive while the client is active and separates the inactivity timeout from the token
  lifetime. We align with OPC UA, the one control protocol in the set.
- **Machines enroll, then hold their own key**: A Kubernetes kubelet, a Teleport agent,
  and a Tailscale node each exchange a one-time token for a standing credential. We use
  the session of the person who installs the Driver in place of a join token.
- **First login creates the account**: CockroachDB and Vault create the internal record
  at the first external login. Grafana documents that a link by email lets an external
  login take over a local user. We create and never link.
- **The first account**: CockroachDB `root` always exists and host material always opens
  it. Grafana, Keycloak, and Vault apply their settings one time and recover with a tool
  on the host. We align with CockroachDB, because RFC 0063 depends on it.

## 5 Principles

1. **The service knows subjects only as IDs**: The authentication service stores and
   returns an `ontology.ID`. It imports no package that owns a subject.
2. **A new subject type fits with one line**: A future service-account type that the
   Python client logs in as, or a node, adds one entry at the wiring site.
3. **A new method fits with one authenticator**: It adds one union variant and one
   implementation of the interface.
4. **One artifact on the request path**: Every request carries a session token and
   nothing else.
5. **A deliberate act can interrupt a test, and a timer cannot**: Revocation closes a
   live stream. No clock closes the stream of a live client.
6. **Secrets are stored hashed and shown one time**: The Core stores no value that a
   client can log in with.

## 6 Design

### 6.0 Subjects

A subject is an existing record: a user or a rack. No identity record sits between the
subject and its credentials. `access.Request.Subject` is already an `ontology.ID`, and
`role.Writer.AssignRole` already takes one
(`core/pkg/service/access/rbac/role/writer.go:78`), so a rack subject reaches access
control with no change to that seam.

The service receives a map from subject type to the methods that type accepts:

| Subject | `password` | `api_key` | `oidc` |
| ------- | ---------- | --------- | ------ |
| User    | Yes        | Yes       | Yes    |
| Rack    | No         | Yes       | No     |

The map is built in `core/pkg/service/layer.go` and injected. A credential for a pair
that is not in the map is a validation error.

### 6.1 Credentials

One Gorp table holds every credential. The method data is an Oracle union, after `Tab`
in `schemas/synnax/versions/panel/v0.oracle`, so Oracle generates the `Method` enum and
one typed shape for each method.

```
Data union on method {
    password {
        hash bytes
    }
    api_key {
        hash bytes
    }
    oidc {}
}

Credential struct {
    key        Key          // UUID, @key
    subject    ontology.ID
    identifier string
    name       string
    created    timestamp
    last_used  timestamp
    data       Data
}
```

- **Subject**: A credential points at exactly one subject. A subject holds zero or more
  credentials.
- **Identifier**: The username for `password`, the key ID for `api_key`, and the issuer
  and account for `oidc`. It is unique within its method. A second credential with the
  same method and identifier is a uniqueness error.
- **Indexes**: Two lookup indexes, declared in Go with `gorp.NewLookupIndex`
  (`x/go/gorp/index.go:144`): one on the pair of method and identifier, and one on the
  subject. Oracle `@index` covers one top-level field, so it cannot express the pair.
- **Output**: The union is Go-only, as `SecureCredentials` is today. Clients receive a
  credential with no `data`.
- **Create is an upsert**: A create with an existing key replaces the record, which is
  the house rule. A password change is a create with the same key.

The username leaves the user record and becomes the identifier of the password
credential. A rename changes one field of one record. PR #3038 moves the username to a
password record that is keyed by the user. This table replaces that shape.

### 6.2 Methods

- **`password`**: The proof is a username and a password. The data is the bcrypt hash,
  as today.
- **`api_key`**: The Core makes the key from a random key ID and a random secret, and
  returns it one time. The identifier is the key ID. The data is the SHA-256 hash of the
  secret. A key has no scope of its own: it acts as its subject.
- **`oidc`**: The proof is the authorization code from a provider. The authenticator
  exchanges the code at the provider, validates the ID token against the keys of the
  provider, and uses the issuer and the `sub` claim as the identifier. The data is
  empty, because the Core holds no secret for the account. The Core must reach the
  provider over the network, so an air-gapped site uses an on-site provider. A site with
  smart cards puts the card check in its provider.

A provider is a start setting of the Core: a name, an issuer URL, and a client ID. The
login flow has one step before the proof exists. The client asks the Core for the
authorization URL of a provider, opens the system browser, and receives the code on a
loopback redirect, with PKCE (RFC 8252). Only the Console runs this flow. A script uses
an API key.

When a provider account has no credential, the Core creates a user and an `oidc`
credential, and gives the user the built-in Viewer role
(`core/pkg/service/access/rbac/builtin/builtin.go:176`). A provider setting turns this
off. The Core never links a provider account to an existing user by email or name. A
logged-in user adds a link from their own session.

### 6.3 The interface

```go
// Login checks a proof and starts a session.
func (s *Service) Login(ctx context.Context, proof Proof) (Session, string, error)

// Authenticate returns the session for a token.
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error)

// Logout ends sessions.
func (s *Service) Logout(ctx context.Context, keys ...SessionKey) error

func (w Writer) Create(
    ctx context.Context,
    subject ontology.ID,
    in Input,
) (Credential, error)
func (w Writer) Delete(ctx context.Context, keys ...Key) error
func (w Writer) DeleteFor(ctx context.Context, subjects ...ontology.ID) error
```

Each method implements one interface:

```go
type Authenticator interface {
    // Authenticate checks the proof and returns the credential it matches.
    Authenticate(ctx context.Context, proof Proof, find Finder) (Credential, error)
    // Create builds the identifier and stored data for a new credential.
    Create(ctx context.Context, in Input) (string, Data, error)
}
```

- **`Proof` and `Input`** are unions on `Method`, as `Data` is.
- **`Finder`** returns the credential for an identifier within the method. The password
  authenticator finds by username and then compares the hash. The OpenID Connect
  authenticator validates the code first and then finds by issuer and account.
- **Dispatch**: `ServiceConfig` takes a `map[Method]Authenticator`, built at the wiring
  site. A proof for a method that is not in the map is a validation error, because the
  method comes from the client.
- **Failure**: An unknown identifier and a wrong secret return the same
  `ErrInvalidCredentials`, so a caller cannot test which usernames exist.

`Login` finds the authenticator, calls it, writes `last_used`, creates the session, and
returns the token. Everything in this section is new. It replaces
`Service.Authenticate(ctx, tx, creds)` (`core/pkg/service/auth/service.go:90`), the
`Writer` methods `Register`, `UpdateUsername`, `ChangePassword`, and `Deactivate`
(`core/pkg/service/auth/writer.go:33-102`), and the `token` package.

### 6.4 Sessions

```
Session struct {
    key         Key          // UUID, @key
    token_hash  bytes
    subject     ontology.ID
    credential  credential.Key
    created     timestamp
    last_active timestamp
}
```

A session is a record in a Gorp table. The table sits on Aspen, so every node in a
multi-node cluster reads it. The token is a random value, and the Core stores its
SHA-256 hash under a lookup index. The signing key and the JWT go away.

On each request the middleware hashes the token, reads the session from the in-memory
index, and puts the subject and the session key on the context. `GetSubject`
(`core/pkg/api/auth/middleware.go:85`) keeps its signature.

A session ends in exactly two ways:

1. **Revocation**: A delete of the record. Logout deletes one session. An administrator
   deletes any session. A delete of a credential deletes its sessions, and a delete of a
   subject deletes its credentials. A change to the data of a credential deletes its
   sessions, except the one that made the change.
2. **Idle timeout**: A session with no activity for the timeout is dead. A request with
   its token fails, and a sweep on each node deletes the record.

Activity is any authenticated request and any open stream. A node records activity in
memory and writes `last_active` at most one time in each quarter of the timeout, so a
request does not cause a write. A client library with nothing else to send calls a renew
endpoint at half the timeout. A live client therefore never idles out. A client that
crashes or loses its network stops, and its session ends.

A session has no absolute lifetime.

**Revocation latency**: On the Core that receives the delete, revocation is immediate.
On another node it takes effect when Aspen delivers the delete, which is the gossip
interval. A node that is cut off from its peers acts on its local copy, so a revocation
reaches it when it rejoins. With no absolute lifetime, a stolen session token works
until someone revokes it or its client goes idle.

### 6.5 Streams

A stream authenticates when it opens. The middleware then watches the session through
`Table.Observe` (`x/go/gorp/observe.go:53`). When the session ends, the Core cancels the
context of the stream, and the stream closes with an authentication error.

A test operator can hold control authority through a writer stream for hours. By
principle 5, an administrator who deletes a stolen credential stops that stream at once,
and no timer does. An open stream counts as activity, so a failed renew call cannot end
it.

A safe rotation of a rack API key adds the new key, moves the Driver to it, and then
deletes the old key. A delete of the old key first closes the streams of the Driver.

### 6.6 Accepted methods

A Core start setting lists the methods that the Core accepts. The default is every
method. The wiring site builds the authenticator map from the setting, so a method that
is off has no authenticator.

- A method that is off is refused at login and at credential creation.
- Its stored credentials stay, and work again when the method is back on.
- An endpoint with no token check returns the accepted methods and the provider names.
  The Console login page reads it.
- `api_key` can be off. This is a bad idea today, because racks have no other method. A
  Core with `api_key` off and the embedded Driver on does not start. A future method for
  Drivers removes this limit.
- The root user has no exception. With `password` off, the root user cannot log in with
  a password.

### 6.7 The root user

The root user is the one user whose password credential the Core start settings own.

1. It is an ordinary user record with `root_user` set.
2. At each start the Core makes its password credential match the settings, as
   `reconcileRootUser` does today (`core/pkg/service/user/root.go:50`).
3. The API refuses to change or delete that credential.
4. It exists for first access and for recovery. A person with access to the host can
   always set its password.
5. It holds the Owner role and cannot be deleted. Those rules belong to access control
   and do not change.
6. In every other way it is a normal user. It can hold an API key or a provider link.

It stops being a shared machine account. The embedded Driver no longer receives its
password (§6.8). Synnax Desktop still makes a root password for each launch (RFC 0063).

A site that wants passwords off starts in two steps: start with `password` on, log in as
root, give the Owner role to a provider user, then restart with `password` off. A site
that breaks its provider settings recovers the same way.

### 6.8 Rack enrollment

A Driver logs in as its rack with an API key. The session names the rack, so the Driver
stops storing a rack key.

1. **`synnax-driver login`** asks for a host, a username, and a password, as today. It
   uses them one time: it logs in as that person, creates the rack, creates an API key
   for the rack, and saves the key. It never saves the password.
2. **The Console** creates a rack and its key, and shows the key one time. A person or a
   tool such as Terraform gives the key to the Driver through its configuration.
3. **The embedded Driver** receives a new API key for the embedded rack at each start of
   the Core. The Core deletes the key of the previous start.

Access control already decides who can create a rack, so the person's session does the
work of a join token. A Driver whose key is lost or deleted needs a person to run the
login command again.

### 6.9 User and rack schemas

```
User struct {
    key        Key     // UUID, @key
    first_name string
    last_name  string
    root_user  bool = false
}
```

The `username` field goes away. A user with only a provider link has no username. The
ontology name of a user is the full name, and the Console shows the username from the
password credential where one exists. A first login through a provider fills the name
from the `given_name` and `family_name` claims.

```
Rack struct {
    key          Key
    name         string
    embedded     bool = false
    status       Status?
    integrations string[] = []
}
```

The rack record does not change. Its credentials live in the credential table and point
at it. `DeleteFor` runs in the delete path of `user.Writer` and `rack.Writer`.

### 6.10 API and clients

| Endpoint                   | Token check | Purpose                                  |
| -------------------------- | ----------- | ---------------------------------------- |
| `auth/methods`             | No          | Accepted methods and provider names      |
| `auth/oidc/authorize`      | No          | Authorization URL for a provider         |
| `auth/session/create`      | No          | Login with a proof                       |
| `auth/session/renew`       | Yes         | Activity with no other effect            |
| `auth/session/retrieve`    | Yes         | Sessions of a subject                    |
| `auth/session/delete`      | Yes         | Logout and revocation                    |
| `auth/credential/create`   | Yes         | New credential, key returned one time    |
| `auth/credential/retrieve` | Yes         | Credentials of a subject, with no `data` |
| `auth/credential/delete`   | Yes         | Delete credentials                       |

The login response holds the token, the subject ID, and the Core information that it
holds today. The token travels as `Authorization: Bearer`, as today.

- **Client libraries**: Each client takes one proof when it is built, logs in at the
  first request, and sends the token. When the Core reports that the session ended, a
  client that holds an API key or a password logs in again. The `Refresh-Token` header
  goes away.
- **Console**: It stores the session token and stops storing the password. A dead
  session sends the user to the login page through the `error(auth)` path of RFC 0049.
  Logout deletes the session. User and rack pages gain a credentials list.
- **Driver**: The `username` and `password` fields of `synnax::Config` are replaced by
  an `api_key` field in the Driver configuration.

### 6.11 Seams

- **Authorization**: The middleware sets the subject ID. Access control reads it as it
  does today. A rack needs a role to act, so Phase 7 adds a built-in Driver role and
  assigns it when a rack is created.
- **Audit**: The session holds the subject, the credential, and so the method. The
  middleware puts the session key on the context for a future audit record. The service
  logs each login attempt with its method, identifier, result, and source address, and
  never logs a secret. The credential and session tables are observable.

### 6.12 Worked examples

**A service account.** A later RFC adds a `service_account` resource. Its PR adds one
entry to the subject map, with `api_key`, and calls `DeleteFor` in its delete path. The
Python client is built with an API key, and the session names the service account. The
authentication service does not change.

**A certificate.** A smart card with no provider, or a node that authenticates to a
peer, uses a `certificate` method. The identifier is the fingerprint of the certificate,
and the data is empty. The Core builds the proof from the TLS state that Freighter
already carries (`freighter/go/context.go:63-68`). The PR adds one union variant and one
authenticator. It also needs the Core to request client certificates, which RFC 0045
§4.6 deferred.

## 7 What this RFC does not cover

- **Authorization**: Roles, policies, and the scope of an API key.
- **The audit log**: Where audit records go and who reads them.
- **Node authentication**: Peer calls between Core nodes carry no authentication today.
  §6.12 shows the fit.
- **Certificates and smart cards with no provider**: Blocked on RFC 0045 §4.6.
- **A second factor**: One-time codes and passkeys.
- **SAML and LDAP**: A provider that speaks OpenID Connect bridges both.
- **A limit on failed logins**: Lockout and rate limits.
- **Proof of possession**: Request signatures and DPoP.

## 8 Implementation phases

Each phase is one pull request into `main`.

- **Phase 1: Credential table.** The `Credential` and `Data` schemas, the two indexes,
  the `Authenticator` interface, the password authenticator, and the `Writer`. A
  migration moves each `SecureCredentials` row and its username into a credential. The
  login endpoint calls the new path.
- **Phase 2: Sessions.** The `Session` table, `Login`, `Authenticate`, `Logout`, the
  idle sweep, and the middleware. Deletes the `token` package and `TokenPrivate`.
- **Phase 3: Streams.** A stream closes when its session ends. Open streams count as
  activity.
- **Phase 4: Session API and clients.** The session endpoints, and login with a proof,
  renewal, and logout in the TypeScript, Python, and C++ clients.
- **Phase 5: Console sessions.** The Console stores the token, not the password, and
  logs out.
- **Phase 6: API keys.** The `api_key` authenticator, the credential endpoints, and the
  client bindings.
- **Phase 7: Racks.** The subject map entry, the built-in Driver role, the one-time
  enrollment in `synnax-driver login`, and the key for the embedded Driver. Deletes the
  root password and the CA key path from `driver.Config`
  (`core/cmd/start/start.go:375-379`).
- **Phase 8: Console credentials.** The credentials list on user and rack pages.
- **Phase 9: Accepted methods.** The start setting, `auth/methods`, and the Console
  login page that reads it.
- **Phase 10: OpenID Connect.** Provider settings, the authenticator, the authorize
  endpoint, the Console browser flow, and the first-login user.
- **Phase 11: Username.** Deletes `username` from the user record and from the clients.
- **Phase 12: Cleanup.** Deletes the old login endpoint.

### 8.0 Compatibility

- **Stored data**: Phase 1 migrates every credential. No user sets a new password.
- **Tokens**: Every JWT stops working at the upgrade that carries Phase 2. Clients log
  in again by themselves, because they hold a password.
- **Old clients**: `auth/login` keeps its username and password request for one stable
  release and returns a session token. An old client treats the token as opaque, so it
  works. Phase 12 deletes the endpoint.
- **Old Drivers**: A Driver that holds a saved username and password runs the one-time
  enrollment of §6.8 at its first start after the upgrade and deletes the password from
  its file.

## 9 Resolved decisions

1. **Authentication only**: Roles, policies, and the audit log each need their own RFC.
   This RFC defines the seams.
2. **The subject is the record itself**: Vault and Keycloak put an entity record between
   the resource and its credentials. That pays off when one person merges many external
   accounts, which Synnax does not need.
3. **No service-account type now**: The design must accept one later (§6.12).
4. **Exchange only**: An API key directly on each request is one `curl` line, and GitHub
   and Grafana work that way. The trade is real: a plain HTTP caller makes a login call
   first. The request check is one function that returns a session, so a direct key can
   be added there later.
5. **Stored sessions, opaque token**: A signed token needs no lookup and no replication.
   It cannot be revoked, gives a stream nothing to watch, and needs a signing key that
   every node shares. The trade is real: each login is a replicated write.
6. **On disk, not in memory**: Sessions in memory end at each restart, and one node does
   not see the sessions of another.
7. **No absolute lifetime**: Rejected even as an option that is off by default. A timer
   must not end a live test, and NIST SP 800-82r3 allows the exception. The trade is
   real: the 12 hour and 24 hour session limits of NIST SP 800-63B are not met, and a
   stolen token lives until it is revoked.
8. **One credential table with a typed union**: A table for each method gives typed
   columns but adds a table and management code for each method. Opaque bytes for the
   method data lose the types. The union keeps both.
9. **No join token for racks**: The session of the person who installs the Driver does
   the same work through the normal API. The trade is real: a Driver with a lost key
   cannot recover by itself.
10. **First login creates a Viewer**: A user with no role would be safer, but that user
    sees an empty Console and needs an administrator before any use. The trade is real:
    every person in the directory of the provider can read the data of the Core, unless
    the provider setting turns creation off.
11. **API keys can be turned off**: A rule that forbids it would block a site whose
    security rules ban standing keys. The Core refuses to start in the one state that
    cannot work.
12. **No root exception to accepted methods**: A password that still works for one
    account fails a site that requires smart cards. Host access is the last resort in
    both designs.
13. **Settings own the root credential for its whole life**: Grafana and Keycloak apply
    the setting one time and recover with a host tool. RFC 0063 needs a new root
    password at each launch. The trade is real: the root password stays in a flag or an
    environment variable.

## 10 Open questions

1. **Idle timeout**: The default value and its setting name. A long default, such as 7
   days, lets a Console user close a laptop for a weekend.
2. **Token and key format**: A prefix for secret scanners, as in `ghp_` and `glsa_`, and
   the lengths of the key ID and the secret.
3. **Default root password**: Whether the `synnax` and `seldon` defaults stay
   (`core/cmd/start/flags.go:73-74`).
4. **Provider settings**: Start settings as written, or stored records that the Console
   edits with no restart.
5. **Driver role**: The policies of the built-in role that a rack receives.
6. **Per-node signing**: Whether a JWT from one node fails on another today. The code
   suggests it, and no test confirms it. Sessions remove the question.
