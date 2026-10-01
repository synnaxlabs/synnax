# 73 Authentication

- **Author**: Patrick Dotson
- **Date**: 2026-10-01
- **Related**:
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md),
  [PR #3038 - SY-4952: Key credentials by user UUID and move the username onto them](https://github.com/synnaxlabs/synnax/pull/3038)

## 0 Summary

Synnax can authenticate one thing in one way: a user, with a username and password. So
the Driver logs in as a person, a rack has no identity of its own, and a token cannot be
revoked.

This RFC splits authentication into three records:

- A **subject** is whatever logs in: a user or a rack.
- A **credential** ties one subject to one method: a password, an API key, or an account
  at an OpenID Connect provider.
- A **session** is what a login creates. Every request carries its token.

The authentication service only sees subjects as IDs, and it reaches each method through
one small interface. Adding a subject type or a method later does not change the
service. Authorization and audit are out of scope. Authentication hands them a subject
ID, as it does today.

## 1 Motivation

- **The Driver is not a person**: It logs in with a user's password and saves that
  password to a world-writable file (`driver/rack/persist.cpp:17-22`). The embedded
  Driver gets the root password of the Core (`core/cmd/start/start.go:375`).
- **A rack has no identity**: The Driver stores a rack key and claims it at start
  (`driver/rack/remote.cpp:28-50`). Any user who can write racks can claim any rack.
- **One method**: The service checks a bcrypt hash and nothing else
  (`core/pkg/service/auth/service.go:90-113`). Customers ask for single sign-on, API
  keys, and smart cards.
- **No revocation**: The token is a signed JWT that the Core does not store. A deleted
  user's token works for up to 24 hours, and there is no logout.
- **Clients keep the password**: Every client holds the plaintext password and logs in
  again when its token expires. The Console saves it to disk
  (`console/src/session/core/slice.ts:17-22`).

## 2 Background and prior art

Security literature splits access into three jobs: authentication (who is asking),
authorization (may they do this), and audit (a record of what was decided). All three
share one value, the identity of the caller. Synnax already has that seam. The
middleware puts a subject ID on each request (`core/pkg/api/auth/middleware.go:40`), and
about 30 API files pass it to access control.

How other systems answer the questions in this RFC:

- **One interface per method**: Kubernetes and Grafana each run a set of authenticators
  that all return the same identity type. We do the same.
- **Credentials as their own records**: Ory Kratos stores a type, an identifier, and
  private data for each credential, and requires the identifier to be unique within its
  type. We follow that shape.
- **Exchange or direct**: Vault, Teleport, and Microsoft Entra exchange every proof for
  one internal token. GitHub, Grafana, and Stripe also accept an API key directly on
  each request. We exchange.
- **Stored or signed sessions**: Vault and Grafana store sessions and can revoke them at
  once. A signed token, as in Entra, lives until it expires.
- **Live connections**: OpenSSH, PostgreSQL, and Kafka check at open and never again.
  NATS closes the connection when the token expires. OPC UA keeps the session alive
  while the client is active. We follow OPC UA, the one control protocol in the group.
  NIST SP 800-82r3 warns that automatic logoff "can be detrimental to the operation of
  the OT system".
- **Machine enrollment**: Kubernetes, Teleport, and Tailscale hand a machine a one-time
  join token, which it trades for its own credential.
- **Smart cards**: A card login is a TLS client certificate plus a mapping from the
  certificate to an account. Keycloak and Entra both do this and then act as a normal
  OpenID Connect provider.

## 3 Principles

1. **The service knows subjects only as IDs**: It never imports the user or rack
   packages.
2. **New subject types and methods plug in**: A future service account, or a certificate
   method, must not change the service.
3. **One thing on the request path**: Every request carries a session token and nothing
   else.
4. **A person can interrupt a test, and a timer cannot**: Revocation closes a live
   stream. No clock does.
5. **The Core stores nothing a client could log in with**: Secrets are hashed and shown
   once.

## 4 Design

### 4.0 Subjects

A subject is an existing record, named by its `ontology.ID`. There is no separate
identity record between a subject and its credentials. Access control already takes an
`ontology.ID` as its subject, so a rack reaches it with no change.

Each subject type accepts a fixed set of methods. The set is a map built in
`core/pkg/service/layer.go` and passed to the service:

| Subject | `password` | `api_key` | `oidc` |
| ------- | ---------- | --------- | ------ |
| User    | Yes        | Yes       | Yes    |
| Rack    | No         | Yes       | No     |

### 4.1 Credentials

One table holds every credential. The method-specific data is an Oracle union, the same
pattern as `Tab` in the panel schema, so Oracle generates the `Method` enum and a typed
shape for each method.

```
Data union on method {
    password { hash bytes }
    api_key  { hash bytes }
    oidc     {}
}

Credential struct {
    key        Key          // UUID
    subject    ontology.ID
    identifier string
    name       string
    created    timestamp
    last_used  timestamp
    data       Data
}
```

- A credential belongs to exactly one subject. A subject can have any number.
- The `identifier` is the public value used to find a credential. It is unique within a
  method.
- The table has two lookup indexes: one on method plus identifier, and one on subject.
- `data` never leaves the Core. Clients see every other field.
- Creating a credential with an existing key replaces it. That is how a password
  changes.

| Method     | Proof at login         | Identifier         | Stored data           |
| ---------- | ---------------------- | ------------------ | --------------------- |
| `password` | Username and password  | Username           | bcrypt hash           |
| `api_key`  | The key                | Key ID             | SHA-256 of the secret |
| `oidc`     | Code from the provider | Issuer and account | Nothing               |

An API key is random, so a fast hash is safe. The Core returns the key once, when it is
created. A key has no permissions of its own. It acts as its subject.

This table replaces `SecureCredentials` and the shape in PR #3038. The username moves
from the user record to the identifier of the password credential, so a rename edits one
field.

### 4.2 OpenID Connect

A provider is a Core start setting: a name, an issuer URL, and a client ID. Only the
Console logs in this way. It asks the Core for the provider's login URL, opens the
system browser, and receives a code on a loopback redirect. That code is the proof. The
Core exchanges it with the provider and validates the result, so the Core must be able
to reach the provider. An air-gapped site runs its own provider. A site with smart cards
puts the card check in its provider.

Three rules cover how provider accounts become users:

1. The first login creates a user with the built-in Viewer role. A provider setting can
   turn this off.
2. The Core never links a provider account to an existing user because an email or name
   matches. Grafana documents this as an account takeover risk.
3. A logged-in user can link their provider account from their own session.

### 4.3 The interface

```go
func (s *Service) Login(ctx context.Context, proof Proof) (Session, string, error)
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error)
func (s *Service) Logout(ctx context.Context, keys ...SessionKey) error

func (w Writer) Create(
    ctx context.Context,
    subject ontology.ID,
    in Input,
) (Credential, error)
func (w Writer) Delete(ctx context.Context, keys ...Key) error
func (w Writer) DeleteFor(ctx context.Context, subjects ...ontology.ID) error
```

`Login` turns a proof into a session and its token. The middleware calls `Authenticate`
on every request. The `Writer` manages credentials the same way for every subject and
method.

Each method implements one interface:

```go
type Authenticator interface {
    // Authenticate checks the proof and returns the credential it matches.
    Authenticate(ctx context.Context, proof Proof, find Finder) (Credential, error)
    // Create builds the identifier and stored data for a new credential.
    Create(ctx context.Context, in Input) (string, Data, error)
}
```

`Proof` and `Input` are unions on `Method`, like `Data`. `Finder` looks up a credential
by identifier. The service takes a `map[Method]Authenticator` in its config. A proof for
a method that is not in the map is a validation error. An unknown identifier and a wrong
secret return the same error, so a caller cannot probe for usernames.

This replaces today's `Service.Authenticate`, the four `Writer` methods, and the `token`
package.

### 4.4 Sessions

```
Session struct {
    key         Key          // UUID
    token_hash  bytes
    subject     ontology.ID
    credential  credential.Key
    created     timestamp
    last_active timestamp
}
```

A session is a Gorp record, so it survives a restart and every node in a cluster sees
it. The token is a random value, and the Core stores only its hash. On each request the
middleware hashes the token and finds the session through an in-memory index. JWTs and
the signing key go away.

A session ends in two ways:

1. **Revocation**: Someone deletes it. Logout deletes one session, and an administrator
   can delete any. Deleting a credential deletes its sessions. Deleting a subject
   deletes its credentials. Changing a password ends that credential's other sessions.
2. **Idle timeout**: No activity for the timeout period. Any request or open stream
   counts as activity, and a client with nothing to send calls a renew endpoint. So a
   running client never times out, and a crashed one does.

There is no absolute lifetime.

Recording activity does not cost a write per request. Each node tracks it in memory and
updates `last_active` a few times per timeout period.

**Revocation is not instant everywhere.** It is immediate on the node that receives the
delete. Other nodes see it when Aspen replicates the delete. A node cut off from the
cluster keeps honoring the session until it rejoins. And with no absolute lifetime, a
stolen token works until someone revokes it or its client stops.

### 4.5 Streams

A stream authenticates when it opens and then watches its session. When the session
ends, the Core closes the stream with an authentication error.

An operator can hold control authority through a writer stream for hours. An
administrator who deletes a stolen credential stops that stream at once. Nothing else
does: an open stream counts as activity, so its session cannot time out.

To rotate a rack's API key without dropping the Driver, add the new key, move the Driver
to it, and then delete the old one.

### 4.6 Accepted methods

A Core start setting lists the methods the Core accepts. The default is all of them.

- A method that is off is refused at login and at credential creation. Its credentials
  stay in the table and work again if the method is turned back on.
- An unauthenticated endpoint reports the accepted methods, so the Console knows what to
  show on its login page.
- `api_key` can be turned off, but racks have no other way to log in today. A Core with
  `api_key` off and the embedded Driver on refuses to start.
- The root user gets no exception.

### 4.7 The root user

The root user is the one user whose password is owned by the Core start settings.

- It is a normal user record with `root_user` set. It can hold an API key or a provider
  link like anyone else.
- At every start, the Core resets its password to match the settings, as it does today
  (`core/pkg/service/user/root.go:50`). The API refuses to change or delete that
  credential.
- It exists for first access and for recovery. Anyone who can restart the Core can get
  back in.
- It keeps the Owner role and cannot be deleted. Those are access control rules, and
  they do not change.

It stops being a shared machine account, because the embedded Driver no longer uses its
password. Synnax Desktop still generates a root password per launch (RFC 0063).

To run with passwords off: start with `password` on, log in as root, give Owner to a
provider user, and restart with `password` off. Recovery from a broken provider setup
works the same way.

### 4.8 Racks and the Driver

A Driver logs in as its rack with an API key. The session tells it which rack it is, so
it no longer stores a rack key.

- **`synnax-driver login`** still asks for a host, username, and password. It uses them
  once to log in as that person, create the rack, and create an API key for it. It saves
  only the key.
- **The Console** can create a rack and its key and show the key once. This is the path
  for Terraform and other automated installs.
- **The embedded Driver** gets a fresh key for the embedded rack each time the Core
  starts. The Core deletes the previous one.

A rack needs permissions, so racks get a built-in Driver role when they are created.
This is the only access control change in this RFC.

### 4.9 User and rack schemas

```
User struct {
    key        Key
    first_name string
    last_name  string
    root_user  bool = false
}
```

`username` is gone. A user who only logs in through a provider has none. The Console
shows the full name, and the username from the password credential where one exists.

The rack schema does not change. A rack's credentials live in the credential table and
point at it.

### 4.10 API and clients

| Endpoint                   | Token | Purpose                          |
| -------------------------- | ----- | -------------------------------- |
| `auth/methods`             | No    | Accepted methods and providers   |
| `auth/oidc/authorize`      | No    | Login URL for a provider         |
| `auth/session/create`      | No    | Log in with a proof              |
| `auth/session/renew`       | Yes   | Report activity                  |
| `auth/session/retrieve`    | Yes   | List sessions                    |
| `auth/session/delete`      | Yes   | Log out or revoke                |
| `auth/credential/create`   | Yes   | Create or replace a credential   |
| `auth/credential/retrieve` | Yes   | List credentials, without `data` |
| `auth/credential/delete`   | Yes   | Delete credentials               |

- **Client libraries** take one proof at construction and log in on the first request.
  When a session ends, a client holding a password or API key logs in again by itself.
- **The Console** stores the session token instead of the password. A dead session sends
  the user to the login page. User and rack pages gain a credentials list.
- **The Driver** config replaces `username` and `password` with `api_key`.

### 4.11 Does it extend?

**A service account.** A later RFC adds a `service_account` resource. That PR adds one
row to the subject map with `api_key`. The Python client is constructed with a key, and
its session names the service account. The authentication service does not change.

**A certificate.** A smart card without a provider, or a Core node calling a peer, would
use a `certificate` method. The identifier is the certificate fingerprint, and the Core
builds the proof from the TLS state Freighter already carries. That PR adds one union
variant and one authenticator. It also needs the Core to request client certificates,
which RFC 0045 §4.6 deferred.

## 5 What this RFC does not cover

- Roles, policies, and scoped API keys.
- The audit log. Sessions record the subject and credential behind every request, so a
  later audit design has what it needs.
- Authentication between Core nodes. Peer calls are unauthenticated today.
- Smart cards without a provider, SAML, and LDAP.
- Second factors, login rate limits, and lockout.

## 6 Implementation phases

Each phase is one pull request into `main`.

- **Phase 1: Credentials.** The credential table, the `Authenticator` interface, the
  password authenticator, and the migration from `SecureCredentials`.
- **Phase 2: Sessions.** The session table, `Login`, `Authenticate`, `Logout`, the idle
  sweep, and the middleware. Deletes the `token` package.
- **Phase 3: Streams.** Streams close when their session ends and count as activity.
- **Phase 4: Clients.** Session endpoints, and login, renewal, and logout in the
  TypeScript, Python, and C++ clients.
- **Phase 5: Console sessions.** The Console stores the token and logs out.
- **Phase 6: API keys.** The authenticator, the credential endpoints, and client
  bindings.
- **Phase 7: Racks.** The Driver role, enrollment in `synnax-driver login`, and the
  embedded Driver key. Removes the root password from the Driver config.
- **Phase 8: Console credentials.** The credentials list on user and rack pages.
- **Phase 9: Accepted methods.** The start setting, `auth/methods`, and the login page.
- **Phase 10: OpenID Connect.** Provider settings, the authenticator, the Console login
  flow, and first-login users.
- **Phase 11: Username.** Removes `username` from the user record and the clients.
- **Phase 12: Cleanup.** Removes the old login endpoint.

### 6.0 Compatibility

- Phase 1 migrates every stored credential. Nobody resets a password.
- Every existing token stops working at Phase 2. Clients hold a password, so they log in
  again without help.
- The old `auth/login` endpoint keeps working for one stable release, so older clients
  can still log in.
- A Driver with a saved username and password enrolls itself on its first start after
  the upgrade and removes the password from its file.

## 7 Resolved decisions

1. **Exchange only, no API key on each request**: A direct key would make a plain `curl`
   call one line instead of two. We gave that up for a single check on the request path
   and one place for revocation. A direct key can be added later at that same check.
2. **Stored sessions, not signed tokens**: A signed token needs no lookup and no
   replication. It also cannot be revoked, gives a stream nothing to watch, and needs a
   signing key shared by every node. The cost is one replicated write per login.
3. **No absolute session lifetime**: A hard limit could end a live test. The cost is
   that Synnax does not meet the 12 and 24 hour session limits in NIST SP 800-63B, and a
   stolen token lives until it is revoked.
4. **No join token for racks**: The installer's own session does the same job through
   the normal API. The cost is that a Driver that loses its key cannot recover without a
   person.
5. **First provider login gets Viewer, not nothing**: A user with no role opens an empty
   Console. The cost is that everyone in the provider's directory can read the Core's
   data unless creation is turned off.
6. **The root password stays in the start settings**: Grafana and Keycloak apply the
   setting once and recover with a host tool. Synnax Desktop needs a new root password
   on every launch, so the settings stay in charge.

## 8 Open questions

1. **Idle timeout**: The default. Something long, like 7 days, lets a Console user close
   a laptop for a weekend.
2. **Token and key format**: A prefix that secret scanners can recognize, and the
   lengths.
3. **Default root password**: Whether `synnax` and `seldon` stay as defaults.
4. **Provider settings**: Start settings as written, or stored records the Console can
   edit without a restart.
5. **Driver role**: Which policies the built-in role carries.
