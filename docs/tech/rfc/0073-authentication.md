# 73 Authentication

- **Author**: Patrick Dotson
- **Date**: 2026-10-01
- **Related**:
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

Synnax can authenticate one thing in one way: a user, with a username and password. So
the Driver logs in as a person, a rack has no identity of its own, and a token cannot be
revoked.

This RFC splits authentication into three records:

- A **subject** is whatever logs in: a user or a rack.
- A **credential** ties one subject to one method: a password, an API key, or a link to
  an account at an OpenID Connect provider. Each method keeps its own table.
- A **session** is what a login creates. Every request carries its token.

The authentication service sees subjects only as IDs and reaches each method through one
small interface, so a new subject type or method does not change it. Authorization and
audit are out of scope. Authentication hands them a subject ID, as it does today.

## 1 Motivation

- **The Driver is not a person**: It logs in with a user's password and saves it to a
  world-writable file (`driver/rack/persist.cpp:17-22`). The embedded Driver gets the
  root password of the Core (`core/cmd/start/start.go:375`).
- **A rack has no identity**: The Driver stores a rack key and claims it at start
  (`driver/rack/remote.cpp:28-50`). Any user who can write racks can claim any rack.
- **One method**: The service checks a bcrypt hash and nothing else
  (`core/pkg/service/auth/service.go:90-113`). Customers ask for single sign-on, API
  keys, and smart cards.
- **No revocation**: The token is a signed JWT that the Core does not store. A deleted
  user's token works for up to 24 hours, and there is no logout.
- **Clients keep the password**: Every client holds it and logs in again when its token
  expires. The Console saves it to disk (`console/src/session/core/slice.ts:17-22`).

## 2 Background and prior art

Security literature splits access into three jobs:
[authentication, authorization, and auditing](https://bwlampson.site/69-SecurityRealIEEE/69-SecurityRealIEEE.htm).
All three share one value, the identity of the caller. Synnax already has that seam. The
middleware puts a subject ID on each request (`core/pkg/api/auth/middleware.go:40`), and
about 30 API files pass it to access control.

How other systems answer the questions in this RFC:

- **One interface per method**:
  [Kubernetes](https://github.com/kubernetes/apiserver/blob/master/pkg/authentication/authenticator/interfaces.go)
  and
  [Grafana](https://github.com/grafana/grafana/blob/main/pkg/services/authn/authn.go)
  each run a set of authenticators that all return the same identity type. We do the
  same.
- **Credentials as their own records**:
  [Ory Kratos](https://www.ory.com/docs/kratos/concepts/credentials) gives each identity
  one or more credentials, with identifiers that are unique within a credential type.
  [Vault](https://developer.hashicorp.com/vault/docs/concepts/identity) keeps one alias
  per auth method. We do the same, with one table per method.
- **Exchange or direct**:
  [Vault](https://developer.hashicorp.com/vault/docs/concepts/tokens) and
  [Teleport](https://goteleport.com/docs/reference/architecture/authentication/)
  exchange every proof for one internal credential.
  [GitHub](https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api),
  [Grafana](https://grafana.com/docs/grafana/latest/administration/service-accounts/),
  and [Stripe](https://docs.stripe.com/api/authentication) accept a key directly on each
  request. We exchange.
- **Stored or signed sessions**: A Vault token is stored and can be revoked at once. A
  Microsoft Entra access token is
  [valid for an hour by default](https://learn.microsoft.com/en-us/entra/identity/conditional-access/concept-continuous-access-evaluation)
  whatever happens to the account. We store.
- **Live connections**: [SSH](https://www.rfc-editor.org/rfc/rfc4252) and, by default,
  [Kafka](https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/config/internals/BrokerSecurityConfigs.java)
  check a connection when it opens and never again.
  [NATS](https://github.com/nats-io/nats-server/blob/main/server/client.go) closes a
  connection when its token expires.
  [OPC UA](https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.2) ends a
  session only when the client stops sending requests. We follow OPC UA, the one control
  protocol in the group.
  [NIST SP 800-82r3](https://nvlpubs.nist.gov/nistpubs/SP/NIST.SP.800-82r3.pdf) lets a
  site relax session lock and termination where "immediate operator response is required
  in emergency situations".
- **Machine enrollment**:
  [Kubernetes](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-tls-bootstrapping/),
  [Teleport](https://goteleport.com/docs/reference/deployment/join-methods/), and
  [Tailscale](https://tailscale.com/kb/1085/auth-keys) give a machine a short-lived
  enrollment token, which it trades for its own credential.
- **Smart cards**: A card login is a TLS client certificate plus a mapping from the
  certificate to an account.
  [Keycloak](https://www.keycloak.org/docs/latest/server_admin/index.html) and
  [Entra](https://learn.microsoft.com/en-us/entra/identity/authentication/concept-certificate-based-authentication-technical-deep-dive)
  both do this and then act as a normal OpenID Connect provider.

## 3 Principles

1. **The service knows subjects only as IDs**: It never imports the user or rack
   packages.
2. **New subject types and methods plug in**: A future service account, or a certificate
   method, must not change the service.
3. **Every request carries a session token, and nothing else**: The middleware makes one
   check. Passwords, API keys, and provider codes appear only at login.
4. **A person can interrupt a test, and a timer cannot**: Revocation closes a live
   stream. No clock does.
5. **Passwords are never saved to disk**: The one exception is the root password in the
   Core start settings. §4.11 lists what each component saves.

## 4 Design

### 4.0 Subjects

A subject is an existing record, named by its `ontology.ID`. No identity record sits
between a subject and its credentials. Access control already takes an `ontology.ID` as
its subject, so a rack reaches it with no change.

Each subject type accepts a fixed set of methods. The set is a map built in
`core/pkg/service/layer.go` and passed to the service:

| Subject | `password` | `api_key` | `oidc` |
| ------- | ---------- | --------- | ------ |
| User    | Yes        | Yes       | Yes    |
| Rack    | No         | Yes       | No     |

### 4.1 Credentials

Each method has its own table and package, because the three need different fields.

```
Password struct {
    key      Key
    subject  ontology.ID
    username string        // indexed, unique
    hash     bytes         // tagged with its algorithm
}

APIKey struct {
    key     Key
    subject ontology.ID
    hash    bytes        // SHA-256 of the secret
    name    string
    created timestamp
}

OIDCLink struct {
    key      Key
    subject  ontology.ID
    provider string
    account  string        // indexed with provider, unique
}
```

A credential belongs to exactly one subject. A subject can hold one password, any number
of API keys, and one link per provider. A hash never leaves the Core.

**Passwords.** The username lives on the password record, so a rename edits one field.
New passwords are hashed with Argon2id, the
[first choice of OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
Existing bcrypt hashes keep working. Each hash carries a tag for its algorithm and cost.
At login the Core checks with the tagged algorithm and stores the password again with
Argon2id when the tag names anything else.

A `fips` start setting, off by default, restricts passwords to PBKDF2, the only password
hash
[NIST approves](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-132.pdf).
With it on, the Core refuses to check a bcrypt or Argon2id hash at all. It is meant for
a new Core. On an existing Core, each older password fails until an administrator sets
it again. The root password is not affected, because the Core hashes it from the start
settings at every start. This setting is the first piece of a FIPS mode for the whole
Core.

**API keys.** A key is `syk_`, the record key, and 32 random bytes. The Core finds the
record by its key and compares the hash. A fast hash is safe because the secret is
random. The Core returns the key once, at creation.

- A key never expires. It works until someone deletes it.
- A key has no permissions of its own. It acts as its subject.
- A subject can create its own keys. A user with the permission, which the Owner role
  holds, can create a key for anyone, and can then act as that subject.
- `name` is a label such as "Terraform". To see whether a key is in use, the Console
  lists its live sessions.

**OIDC links.** `account` is the permanent ID the provider gives the person (§4.2). The
Core stores no secret for it.

### 4.2 OpenID Connect

A provider is an entry in the Core config file: a name, an issuer URL, a client ID, and
a client secret when the provider requires one. A change needs a restart, and every node
in a cluster needs the same entries.

Only the Console logs in this way, following
[RFC 8252](https://www.rfc-editor.org/rfc/rfc8252). It reads the provider's login
address from `auth/methods`, opens the system browser with a PKCE challenge, and
receives a code on a loopback redirect. The code and the PKCE verifier are the proof.
The Core exchanges them with the provider, so it must be able to reach the provider. An
air-gapped site runs its own. A site with smart cards puts the card check in its
provider.

The provider answers with a signed ID token. The Core checks its signature, issuer,
audience, expiry, and nonce, and reads two claims:

- `sub` becomes the link's `account`. The
  [specification](https://openid.net/specs/openid-connect-core-1_0.html) says it is
  never reassigned, and that an email must not be used as an identifier.
- `name` fills the `name` of a user created at first login.

The Core stores nothing else and discards every token the provider returns.

Three rules cover how provider accounts become users:

1. The first login creates a user with the built-in Viewer role. Everyone in the
   provider's directory can then read the Core's data. A provider setting turns this
   off.
2. The Core never links a provider account to an existing user because an email or name
   matches. Grafana
   [warns against it](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/),
   and it caused an
   [account takeover](https://grafana.com/blog/2023/06/22/grafana-security-release-for-cve-2023-3128/)
   there.
3. A logged-in user links their provider account from their own session. With rule 1
   off, this is the only way in: an administrator creates the user with a password, and
   the user links their own account.

### 4.3 The interface

```go
func (s *Service) Login(ctx context.Context, proof Proof) (Session, string, error)
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error)
func (s *Service) Logout(ctx context.Context, keys ...SessionKey) error
```

`Login` turns a proof into a session and its token. The middleware calls `Authenticate`
on every request. Neither names a subject type or a method.

Each method implements one interface:

```go
type Authenticator interface {
    // Authenticate checks the proof and returns the subject and the credential used.
    Authenticate(ctx context.Context, proof Proof) (Identity, error)
    // DeleteFor removes every credential of the given subjects.
    DeleteFor(ctx context.Context, subjects ...ontology.ID) error
}
```

`Proof` is an Oracle union on `Method`, the same pattern as `Tab` in the panel schema.
`Identity` is a subject, a method, and the key of the credential. The service takes a
`map[Method]Authenticator` in its config. A proof for a method that is not in the map is
a validation error. An unknown username and a wrong password return the same error.

Managing credentials is not in the shared interface, because the operations differ: a
person sets a password, creates a key, or links an account. Each method package has its
own writer. When a user or rack is deleted, its service calls `DeleteFor` on every
authenticator.

An API key is not accepted directly on a request. That keeps one check in the middleware
and one thing for a stream to watch. It has three costs:

- A tool that can only send a fixed header, such as a Grafana data source, cannot
  connect.
- A plain HTTP caller needs a login call and a retry.
- A short script that never logs out leaves a session behind until the idle timeout.

A direct key can be added later: one more lookup in the middleware, chosen by the token
prefix, and a stream that watches its credential.

### 4.4 Sessions

```
Session struct {
    key         Key
    hash        bytes        // SHA-256 of the token's secret
    subject     ontology.ID
    method      Method
    credential  uuid
    address     string
    created     timestamp
    last_active timestamp
}
```

A session is a Gorp record, so it survives a restart and every node in a cluster sees
it. It replaces the JWT, which cannot be revoked, gives a stream nothing to watch, and
needs a signing key shared by every node. The cost is one replicated write per login.

- `method` and `credential` let a deleted or changed credential end its sessions, and
  tell audit how the caller logged in.
- `address` is the client's network address at login. With `created`, it lets an
  administrator tell sessions apart before revoking one.
- `last_active` drives the idle timeout.

A session token is `sys_`, the session key, and 32 random bytes, the same layout as an
API key. That is above the
[64 bit minimum](https://pages.nist.gov/800-63-4/sp800-63b.html) for a session secret
and the
[128 bit strength](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-57pt1r5.pdf)
NIST requires from 2031. The FIPS mode governs the random generator and the hash.

The middleware puts the subject, the credential, and the session key on each request.
Handlers and audit rely on the first two. The session key is optional, so a request with
no session can exist later (§4.3).

A session ends in two ways:

1. **Revocation**: Someone deletes it. Logout deletes one session, and an administrator
   can delete any. Deleting a credential deletes its sessions, and deleting a subject
   deletes its credentials. Changing a password ends that password's other sessions.
2. **Idle timeout**: No activity for 7 days, by default. Any request or open stream
   counts, and a client with nothing to send calls a renew endpoint. Each node tracks
   activity in memory and writes `last_active` only when the stored value is older than
   a quarter of the timeout: at most four writes per session per timeout period.

The idle timeout measures the client, not the person. An open Console always holds
streams, so it never times out, even when nobody touches it. The timeout fires only when
the client is gone. A Driver or script then logs in again with its key. A Console keeps
no password, so its user logs in again by hand.

Revocation is immediate on the node that receives the delete. Other nodes see it when
Aspen replicates it, and a node cut off from the cluster sees it when it rejoins. A new
session has the same delay.

There is no absolute lifetime. That has four accepted consequences:

- A session can live forever. A laptop left open is protected by its operating system
  lock, not by Synnax.
- A stolen token works while someone uses it. Revocation is the only defense.
- A person disabled at the identity provider keeps access until their session goes idle
  or is revoked. Offboarding must include a delete of the Synnax user.
- The session limits in
  [NIST SP 800-63B](https://pages.nist.gov/800-63-4/sp800-63b.html), 24 hours at AAL2
  and 12 hours at AAL3, are not met.

### 4.5 Streams

A stream is opened with a session token, like any other request. The middleware checks
the token once, when the stream opens, and the messages on the stream carry nothing. The
stream then watches its session. When the session ends, the Core closes the stream with
an authentication error.

An operator can hold control authority through a writer stream for hours. An
administrator who deletes a stolen credential stops that stream at once. Nothing else
does.

### 4.6 Accepted methods

The `auth-methods` start setting lists the methods the Core accepts. The default is all
of them.

- A method that is off is refused at login and at credential creation. Its credentials
  stay in the table and work again if the method is turned back on.
- `api_key` can be turned off, but racks have no other way to log in. A Core with
  `api_key` off and the embedded Driver on refuses to start.
- The root user gets no exception.

### 4.7 The root user

The root user is the one user whose password is owned by the Core start settings.

- It is a normal user record with `root_user` set. It can hold an API key or a provider
  link like anyone else.
- At every start, the Core resets its password to match the settings, as it does today
  (`core/pkg/service/user/root.go:50`). The API refuses to change or delete it.
- It exists for first access and for recovery. Anyone who can restart the Core can get
  back in.
- It keeps the Owner role and cannot be deleted.

To run with passwords off: start with `password` on, log in as root, give Owner to a
provider user, and restart with `password` off. Recovery from a broken provider setup
works the same way.

### 4.8 Racks and the Driver

A Driver logs in as its rack with an API key. The session tells it which rack it is, so
it no longer stores a rack key.

- **`synnax-driver login`** still asks for a host, username, and password. It uses them
  once to log in as that person, create the rack, and create an API key for it. It saves
  only the key.
- **The Console** can create a rack and its key and show the key once, for automated
  installs.
- **The embedded Driver** has one standing key, like any other rack. The Core creates it
  with the embedded rack and saves it in the config file it already writes for the
  embedded Driver. Every later start reuses it. The Core makes a new key only when the
  saved one is missing or refused.

There is no join token. The installer's own session does that job through the normal
API. A standalone Driver that loses its key needs a person to run the login command
again.

A rack needs permissions. The built-in Host role exists for this
(`core/pkg/service/access/rbac/builtin/builtin.go:111`). It is renamed to Driver, and
the Core assigns it to each rack at creation.

### 4.9 User and rack schemas

```
User struct {
    key       Key
    name      string
    root_user bool = false
}
```

`username` moves to the password record. The user endpoint still returns it, filled from
the password table on read. One `name` field replaces `first_name` and `last_name`. A
migration joins the two and falls back to the username when both are empty.

The rack schema does not change.

### 4.10 API and clients

| Endpoint                | Token | Purpose                                                           |
| ----------------------- | ----- | ----------------------------------------------------------------- |
| `auth/methods`          | No    | Accepted methods, and each provider's login address and client ID |
| `auth/session/create`   | No    | Log in with a proof                                               |
| `auth/session/renew`    | Yes   | Report activity                                                   |
| `auth/session/retrieve` | Yes   | List sessions                                                     |
| `auth/session/delete`   | Yes   | Log out or revoke                                                 |
| `auth/password/set`     | Yes   | Set or change a password or username                              |
| `auth/key/create`       | Yes   | Create an API key, returned once                                  |
| `auth/key/retrieve`     | Yes   | List API keys                                                     |
| `auth/key/delete`       | Yes   | Delete API keys                                                   |
| `auth/oidc/link`        | Yes   | Link a provider account to the caller                             |
| `auth/oidc/retrieve`    | Yes   | List provider links                                               |
| `auth/oidc/unlink`      | Yes   | Remove a provider link                                            |

These replace `auth/login`, `auth/change-password`, and `user/change-username`. Changing
your own password needs the current one. Setting another user's needs permission.

A client takes one proof when it is built: a username and password, or an API key.
Giving both is a validation error.

```ts
new Synnax({ host: "core.example.com", port: 9090, username: "jane", password: "..." });
new Synnax({ host: "core.example.com", port: 9090, apiKey: "syk_..." });
```

```python
sy.Synnax(host="core.example.com", port=9090, username="jane", password="...")
sy.Synnax(host="core.example.com", port=9090, api_key="syk_...")
sy.Synnax()  # the key that `sy login` saved to the keyring
```

```cpp
synnax::Synnax(synnax::Config{.host = "core.example.com", .api_key = "syk_..."});
```

Both forms make the same login call on the first request, with a different proof:

```json
{ "proof": { "method": "password", "username": "jane", "password": "..." } }
{ "proof": { "method": "api_key", "key": "syk_..." } }
```

The response holds the session token and the subject. For a rack's key the subject is a
rack, so a client exposes the subject, and a user record only when the subject is a
user.

- **Client libraries** hold the password or key in memory and log in again when a
  session ends.
- **The desktop Console** keeps the session token in the keychain and sends it in the
  `Authorization` header.
- **The browser Console** never sees the token. The Core sets it in a cookie that
  scripts cannot read, and the middleware accepts the token from the header or the
  cookie. This also removes the token from the WebSocket URL.
- **Both Consoles** send the user to the login page when a session is dead, and gain a
  credentials list on user and rack pages.

### 4.11 What each component saves

No component saves a password, except the root password in the Core start settings.
Everything else is a hash, or a token or key that can be revoked on its own.

| Component        | Saves                                             | Where                                            | Protection                                                      |
| ---------------- | ------------------------------------------------- | ------------------------------------------------ | --------------------------------------------------------------- |
| Core             | Hashes of passwords, API keys, and session tokens | Database                                         | A hash cannot be used to log in                                 |
| Core             | Root password                                     | Start flag, environment variable, or config file | Plain text                                                      |
| Core             | Provider client secret, when required             | Config file                                      | Plain text                                                      |
| Core             | API key of the embedded Driver                    | Driver config file                               | Plain text, owner-only file                                     |
| Driver           | API key of its rack                               | Driver state file                                | Plain text, owner-only file                                     |
| Console, desktop | Session token                                     | Operating system keychain                        | Encrypted                                                       |
| Console, browser | Session token                                     | A cookie that scripts cannot read                | The page never holds it                                         |
| Python CLI       | API key                                           | Operating system keyring                         | Encrypted                                                       |
| Client libraries | Nothing                                           |                                                  |                                                                 |
| Synnax Desktop   | Nothing                                           |                                                  | The launch password goes to the Core in an environment variable |

A root password given as a start flag shows in the process list. The docs recommend the
environment variable or a config file that only the Core's user can read.

### 4.12 Configuration changes

Each setting is a start flag, a config file key, and an environment variable.

**Core**

| Setting                | Change                                             |
| ---------------------- | -------------------------------------------------- |
| `username`, `password` | Unchanged, with the `synnax` and `seldon` defaults |
| `auth-methods`         | New. Default: all                                  |
| `fips`                 | New. Default: false                                |
| `session-idle-timeout` | New. Default: 7 days                               |
| `oidc-providers`       | New. Config file only                              |

**Driver** (config file, `SYNNAX_DRIVER_` environment variables, and state file)

| Setting                                      | Change                                        |
| -------------------------------------------- | --------------------------------------------- |
| `connection.username`, `connection.password` | Removed                                       |
| `connection.api_key`                         | New                                           |
| `remote_info.rack_key`                       | Removed                                       |
| State file permissions                       | Owner-only, from readable and writable by all |

The config file that the Core writes for the embedded Driver changes in the same way.

**Client libraries**

| Library                            | Change                                                                             |
| ---------------------------------- | ---------------------------------------------------------------------------------- |
| TypeScript `synnaxParamsZ`         | Adds `apiKey`. `username` and `password` become optional                           |
| Python `Synnax(...)` and `Options` | Adds `api_key`                                                                     |
| Python `sy login`                  | Creates an API key and saves it to the keyring, in place of the password           |
| C++ `synnax::Config`               | Adds `api_key`. The `synnax` and `seldon` defaults apply only when no key is given |
| C interface `synnax_client_open`   | Gains an API key parameter, for LabVIEW                                            |

**Console**

| Setting                       | Change                                                    |
| ----------------------------- | --------------------------------------------------------- |
| Saved Core record, `password` | Removed                                                   |
| Session token                 | New. In the keychain on desktop, in a cookie in a browser |

**Synnax Desktop**: No change.

### 4.13 Does it extend?

**A service account.** A later RFC adds a `service_account` resource. That PR adds one
row to the subject map with `api_key`, and calls `DeleteFor` when an account is deleted.
The authentication service does not change.

**A certificate.** A smart card without a provider, or a Core node calling a peer, would
use a `certificate` method. That PR adds one package with a table and an authenticator,
and one variant to `Proof`. It also needs the Core to request client certificates, which
RFC 0045 §4.6 deferred.

**Back-channel logout.** Keycloak can
[call an endpoint](https://www.keycloak.org/docs/latest/server_admin/index.html) when a
user logs out or is disabled. That PR adds one endpoint that validates the provider's
signed logout token and deletes the sessions of the matching link. Entra does not
support this.

## 5 What this RFC does not cover

- Roles, policies, and scoped API keys.
- Mapping a provider's `groups` claim to roles.
- The audit log.
- Authentication between Core nodes.
- Smart cards without a provider, SAML, and LDAP.
- Second factors.
- A limit on failed logins. The first version has none.
- A Console screen lock after a person is inactive.
- FIPS mode beyond password hashing.

## 6 Implementation phases

Each phase is one pull request into `main`.

- **Phase 1: Passwords.** The `Authenticator` interface, the password package and its
  table, the migration from `SecureCredentials`, tagged hashes with Argon2id, and the
  `fips` setting.
- **Phase 2: Sessions.** The session table, `Login`, `Authenticate`, `Logout`, the idle
  sweep, and the middleware. Deletes the `token` package.
- **Phase 3: Streams.** Streams close when their session ends and count as activity.
- **Phase 4: Clients.** Session endpoints, and login, renewal, and logout in the
  TypeScript, Python, and C++ clients.
- **Phase 5: Console sessions.** The token in the keychain on desktop and in a cookie in
  a browser, with no saved password, and logout.
- **Phase 6: API keys.** The authenticator, the key endpoints, and client bindings.
- **Phase 7: Racks.** The Host role renamed to Driver and assigned to each rack,
  enrollment in `synnax-driver login`, and the embedded Driver key.
- **Phase 8: Console credentials.** The credentials list on user and rack pages.
- **Phase 9: Accepted methods.** The `auth-methods` setting, `auth/methods`, and the
  login page.
- **Phase 10: OpenID Connect.** Provider settings, the authenticator, the Console login
  flow, and first-login users.
- **Phase 11: User record.** Moves `username` off the user record and merges the two
  name fields into `name`.
- **Phase 12: Cleanup.** Removes the old login endpoint.

### 6.0 Compatibility

- Phase 1 migrates every stored credential. Nobody resets a password.
- Every existing token stops working at Phase 2. Clients log in again without help.
- The old `auth/login` endpoint keeps working for one stable release.
- A Driver with a saved username and password enrolls itself on its first start after
  the upgrade and removes the password from its file.
- The Console and the Python CLI delete the password they saved before the upgrade, once
  the first login with it succeeds.
