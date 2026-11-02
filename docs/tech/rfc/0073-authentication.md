# 73 Authentication

- **Author**: Patrick Dotson
- **Date**: 2026-10-01
- **Related**:
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0049 - Client connection lifecycle](0049-client-connection-lifecycle.md),
  [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md)

## 0 Summary

Synnax authenticates one thing in one way: a user, with a username and password. So the
Driver logs in as a person, a rack has no identity, and a token cannot be revoked.

This RFC splits authentication into three records:

- A **subject** is whatever logs in: a user or a rack.
- A **credential** ties one subject to one method: a password, an API key, or a link to
  an OpenID Connect account. Each method keeps its own table.
- A **session** is what a login creates. Every request carries its token.

The service sees subjects only as IDs and reaches each method through one small
interface, so a new subject type or method does not change it. Authorization and audit
are out of scope: they receive a subject ID, as today.

## 1 Motivation

- **The Driver is not a person**: It logs in with a user's password and saves it to a
  world-writable file (`driver/rack/persist.cpp:17-22`). The embedded Driver gets the
  root password (`core/cmd/start/start.go:375`).
- **A rack has no identity**: The Driver stores a rack key and claims it at start
  (`driver/rack/remote.cpp:28-50`). Any user who can write racks can claim any rack.
- **One method**: The service checks a bcrypt hash and nothing else
  (`core/pkg/service/auth/service.go:90-113`). Customers ask for single sign-on, API
  keys, and smart cards.
- **No revocation**: The token is a signed JWT the Core does not store. A deleted user's
  token works for up to 24 hours, and there is no logout.
- **Clients keep the password**: Every client holds it, and the Console saves it to disk
  (`console/src/session/core/slice.ts:17-22`).

## 2 Prior art

[Authentication, authorization, and auditing](https://bwlampson.site/69-SecurityRealIEEE/69-SecurityRealIEEE.htm)
share one value: the identity of the caller. Synnax has that seam already. The
middleware puts a subject ID on each request (`core/pkg/api/auth/middleware.go:40`), and
about 30 API files pass it to access control.

- **One interface per method**:
  [Kubernetes](https://github.com/kubernetes/apiserver/blob/master/pkg/authentication/authenticator/interfaces.go)
  and
  [Grafana](https://github.com/grafana/grafana/blob/main/pkg/services/authn/authn.go)
  run a set of authenticators that all return one identity type.
- **Credentials as their own records**:
  [Ory Kratos](https://www.ory.com/docs/kratos/concepts/credentials) gives an identity
  one or more credentials, unique within their type.
  [Vault](https://developer.hashicorp.com/vault/docs/concepts/identity) keeps one alias
  per auth method.
- **Exchange or direct**:
  [Vault](https://developer.hashicorp.com/vault/docs/concepts/tokens) and
  [Teleport](https://goteleport.com/docs/reference/architecture/authentication/) trade
  every proof for one internal credential.
  [GitHub](https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api),
  [Grafana](https://grafana.com/docs/grafana/latest/administration/service-accounts/),
  and [Stripe](https://docs.stripe.com/api/authentication) accept a key on each request.
  We exchange.
- **Stored or signed sessions**: A Vault token is stored and revoked at once. A signed
  Entra token stays
  [valid for an hour](https://learn.microsoft.com/en-us/entra/identity/conditional-access/concept-continuous-access-evaluation)
  whatever happens to the account. We store.
- **Live connections**: [SSH](https://www.rfc-editor.org/rfc/rfc4252) and
  [Kafka](https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/config/internals/BrokerSecurityConfigs.java)
  check a connection when it opens and never again.
  [NATS](https://github.com/nats-io/nats-server/blob/main/server/client.go) closes it
  when the token expires.
  [OPC UA](https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.2) ends a
  session only when the client stops sending. We follow OPC UA, and
  [NIST SP 800-82r3](https://nvlpubs.nist.gov/nistpubs/SP/NIST.SP.800-82r3.pdf) lets a
  site relax session lock and termination where "immediate operator response is required
  in emergency situations".
- **Machine enrollment**:
  [Kubernetes](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-tls-bootstrapping/),
  [Teleport](https://goteleport.com/docs/reference/deployment/join-methods/), and
  [Tailscale](https://tailscale.com/kb/1085/auth-keys) give a machine a short-lived
  enrollment token to trade for its own credential. We use the installer's session.
- **Smart cards**: A card login is a TLS client certificate plus a map from certificate
  to account. [Keycloak](https://www.keycloak.org/docs/latest/server_admin/index.html)
  and
  [Entra](https://learn.microsoft.com/en-us/entra/identity/authentication/concept-certificate-based-authentication-technical-deep-dive)
  do this and then act as an OpenID Connect provider. We rely on them.

## 3 Principles

1. **The service knows subjects only as IDs**: It never imports the user or rack
   packages.
2. **New subject types and methods plug in**: Neither changes the service.
3. **Every request carries a session token, and nothing else**: Passwords, API keys, and
   provider codes appear only at login.
4. **A person can interrupt a test, and a timer cannot**: Revocation closes a live
   stream. No clock does.
5. **Passwords are never saved to disk**: The one exception is the root password in the
   Core start settings (§4.11).

## 4 Design

### 4.0 Subjects

A subject is an existing record, named by its `ontology.ID`, which access control
already takes as its subject. Each subject type accepts a fixed set of methods, a map
built in `core/pkg/service/layer.go` and passed to the service:

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
    created  timestamp
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

A subject holds one password, any number of API keys, and one link per provider. A hash
never leaves the Core. Passwords and API keys record `created` so that a later age
policy, such as password rotation or key expiry, applies to credentials that already
exist. A link has no lifecycle of its own, so it carries no date.

**Passwords.** The username lives here, so a rename edits one field. New passwords use
Argon2id, the
[first choice of OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
Each hash carries a tag for its algorithm, so existing bcrypt hashes keep working: at
login the Core checks with the tagged algorithm, then stores the password again with
Argon2id.

A `fips` start setting, off by default, restricts passwords to PBKDF2, the only password
hash
[NIST approves](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-132.pdf).
With it on, the Core refuses to check a bcrypt or Argon2id hash, so on an existing Core
each older password fails until an administrator resets it. The root password is
unaffected, because the Core hashes it from the settings at every start. This is the
first piece of a FIPS mode for the whole Core.

**API keys.** A key is `syk_`, the record key, and 32 random bytes. The Core finds the
record by its key and compares the hash. The key is shown once, never expires, and acts
as its subject with no permissions of its own. A subject creates its own keys. A key for
a rack takes the permission to create racks, which Engineer holds. A key for a user
takes a permission only Owner holds, because the holder can then act as that user. The
Console shows whether a key is in use by listing its live sessions.

**OIDC links.** `provider` is the `name` of an entry in `oidc-providers`, a string
because providers are site configuration. `account` is the `sub` claim, the permanent
opaque ID the provider gives the person.

### 4.2 OpenID Connect

A provider is an entry in the Core config file: a name, an issuer URL, a client ID, and
a client secret when the provider requires one. A change needs a restart, and every node
needs the same entries.

Only the Console logs in this way, following
[RFC 8252](https://www.rfc-editor.org/rfc/rfc8252). It reads the provider's login
address from `auth/methods`, sends the browser there with a PKCE challenge, and receives
a code on a redirect: a loopback port for the desktop Console, its own page at the
Core's origin for the browser Console. The code and verifier are the proof. The Core
exchanges them with the provider, so it must reach it. An air-gapped site runs its own
provider. A site with smart cards puts the card check there.

The Core validates the ID token and reads two claims: `sub` becomes `account`, which the
[specification](https://openid.net/specs/openid-connect-core-1_0.html) says is never
reassigned, and `name` fills a new user's `name`. Nothing else is stored, and every
provider token is discarded.

1. The first login creates a user with the built-in Viewer role, so everyone in the
   provider's directory can read the Core's data. A provider setting turns this off.
2. The Core never links a provider account to an existing user because an email or name
   matches. Grafana
   [warns against it](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/),
   and it caused an
   [account takeover](https://grafana.com/blog/2023/06/22/grafana-security-release-for-cve-2023-3128/)
   there.
3. A logged-in user links their own provider account. With rule 1 off, this is the only
   way in.

### 4.3 The interface

```go
func (s *Service) Login(ctx context.Context, proof Proof) (Session, string, error)
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error)
func (s *Service) Logout(ctx context.Context, keys ...SessionKey) error
```

`Login` turns a proof into a session and its token. The middleware calls `Authenticate`
on every request. Neither names a subject type or a method. Each method implements:

```go
type Authenticator interface {
    // Authenticate checks the proof and returns the subject and the credential used.
    Authenticate(ctx context.Context, proof Proof) (Identity, error)
    // DeleteFor removes every credential of the given subjects within tx.
    DeleteFor(ctx context.Context, tx gorp.Tx, subjects ...ontology.ID) error
}
```

`Proof` is an Oracle union on `Method`, like `Tab` in the panel schema. `Identity` is a
subject and a credential key. The service takes a `map[Method]Authenticator` in its
config. A proof for a method not in the map is a validation error. An unknown username
and a wrong password return the same error.

Credential management is not in the interface, because the operations differ: a person
sets a password, creates a key, or links an account. Each method package has its own
writer. Deleting a user or rack calls `DeleteFor` on every authenticator.

An API key is not accepted directly on a request, so the middleware makes one check and
a stream watches one thing. The costs: a tool that can only send a fixed header, such as
a Grafana data source, cannot connect, and a script that never logs out leaves a session
until the idle timeout. A direct key can be added later as one more lookup, chosen by
the token prefix.

### 4.4 Sessions

```
Session struct {
    key         Key
    hash        bytes        // SHA-256 of the token's secret
    subject     ontology.ID
    credential  uuid
    last_active timestamp
}
```

A session is a Gorp record, so it survives a restart and every node sees it. It replaces
the JWT, which cannot be revoked, gives a stream nothing to watch, and needs a signing
key on every node. The cost is one replicated write per login. `credential` lets a
deleted credential end its sessions.

A token is `sys_`, the session key, and 32 random bytes, the same layout as an API key.
That exceeds the [64 bit minimum](https://pages.nist.gov/800-63-4/sp800-63b.html) for a
session secret and the
[128 bit strength](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-57pt1r5.pdf)
NIST requires from 2031. The FIPS mode governs the generator and the hash.

The middleware puts the subject, the credential, and the session key on each request.
Handlers use the first two, so a request with no session can exist later.

A session ends in two ways:

1. **Revocation**: Logout deletes one session, and an administrator can delete any.
   Deleting a credential deletes its sessions. Deleting a subject deletes its
   credentials. Changing a password ends that password's other sessions.
2. **Idle timeout**: No activity for 7 days, by default. Any request or open stream
   counts, and a client with nothing to send calls a renew endpoint. A node tracks
   activity in memory and writes `last_active` only when the stored value is older than
   a quarter of the timeout: at most four writes per session per period.

The timeout measures the client, not the person. An open Console holds streams, so it
never times out. When a client is gone and comes back, a Driver or script logs in again
with its key, and a Console user logs in by hand. Revocations and new sessions reach
other nodes when Aspen replicates them.

There is no absolute lifetime. So a session can live forever, a stolen token works while
someone uses it, a person disabled at the provider keeps access until their session is
revoked or goes idle, and the
[NIST SP 800-63B](https://pages.nist.gov/800-63-4/sp800-63b.html) limits of 24 hours at
AAL2 and 12 at AAL3 are not met. A laptop left open is protected by its operating system
lock, and offboarding must delete the Synnax user.

### 4.5 Streams

A stream opens with a session token, like any request, and the middleware checks it
once. The stream then watches its session and closes with an authentication error when
the session ends. Nothing else closes it.

### 4.6 Accepted methods

The `auth-methods` start setting lists the methods the Core accepts. Default: all.

- A method that is off is refused at login and at credential creation. Its credentials
  stay and work again when it is turned on.
- `api_key` can be off, but racks have no other method. A Core with `api_key` off and
  the embedded Driver on refuses to start.
- The root user gets no exception.

### 4.7 The root user

The root user is the one user whose password the Core start settings own.

- It is a normal user record with `root_user` set, and can hold an API key or a provider
  link like anyone else.
- At every start the Core resets its password to the settings, as today
  (`core/pkg/service/user/root.go:50`). The API refuses to change or delete it.
- It exists for first access and recovery: anyone who can restart the Core can get in.
  It keeps the Owner role and cannot be deleted.

To run with passwords off: start with `password` on, log in as root, give Owner to a
provider user, and restart with `password` off. Recovery from a broken provider setup
works the same way.

### 4.8 Racks and the Driver

A Driver logs in as its rack with an API key. The session tells it which rack it is, so
it no longer stores a rack key.

- **`synnax-driver login`** asks for a host, username, and password, as today. It uses
  them once to log in, create the rack, and create its key. It saves only the key.
- **The Console** can create a rack and its key, for automated installs.
- **The embedded Driver** has one standing key. The Core creates it with the embedded
  rack, saves it in an owner-only file in its data directory, and reuses it at every
  start, making a new one only when the saved key is missing or refused.

A Driver that loses its key needs a person to run the login command again.

The built-in Host role (`core/pkg/service/access/rbac/builtin/builtin.go:111`) is
renamed to Driver, and the Core assigns it to each rack at creation. The role loses rack
create and delete. Limiting it to its own rack needs a policy per rack, which is access
control work.

### 4.9 User and rack schemas

```
User struct {
    key       Key
    name      string
    root_user bool = false
}
```

`username` moves to the password record, and the user endpoint does not return it. The
endpoint still finds users by username. `name` replaces `first_name` and `last_name`.
The migration joins the two and falls back to the username when both are empty. The rack
schema does not change.

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

A client takes one proof when it is built. Giving both is a validation error.

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

Both make the same login call on the first request:

```json
{ "proof": { "method": "password", "username": "jane", "password": "..." } }
{ "proof": { "method": "api_key", "key": "syk_..." } }
```

The response holds the token and the subject. A client exposes the subject, and a user
record only when the subject is a user.

- **Client libraries** hold the proof in memory, log in again when a session ends, and
  delete their session when the client closes.
- **The desktop Console** keeps the token in the keychain and sends it in the
  `Authorization` header.
- **The browser Console** never sees the token. The Core sets it in a cookie that
  scripts cannot read, and the middleware accepts the header or the cookie, which also
  takes the token out of the WebSocket URL.
- **Both Consoles** go to the login page when a session is dead, and gain a credentials
  list on user and rack pages.

### 4.11 What each component saves

| Component        | Saves                                                                    | Where                                            |
| ---------------- | ------------------------------------------------------------------------ | ------------------------------------------------ |
| Core             | Hashes of passwords, keys, and tokens                                    | Database                                         |
| Core             | Root password, plain text                                                | Start flag, environment variable, or config file |
| Core             | Provider client secret, plain text, when required                        | Config file                                      |
| Core             | API key of the embedded Driver                                           | Owner-only file in the Core data directory       |
| Driver           | API key of its rack                                                      | Owner-only state file                            |
| Console, desktop | Session token                                                            | Operating system keychain                        |
| Console, browser | Session token                                                            | A cookie scripts cannot read                     |
| Python CLI       | API key                                                                  | Operating system keyring                         |
| Client libraries | Nothing                                                                  |                                                  |
| Synnax Desktop   | Nothing. The launch password goes to the Core in an environment variable |                                                  |

A root password given as a flag shows in the process list. The docs recommend the
environment variable or a config file only the Core's user can read.

### 4.12 Configuration changes

**Core**

| Setting                | Change                                             |
| ---------------------- | -------------------------------------------------- |
| `username`, `password` | Unchanged, with the `synnax` and `seldon` defaults |
| `auth-methods`         | New. Default: all                                  |
| `fips`                 | New. Default: false                                |
| `session-idle-timeout` | New. Default: 7 days                               |
| `oidc-providers`       | New. Config file only                              |

**Driver** (config file, `SYNNAX_DRIVER_` environment variables, and state file)

| Setting                                      | Change     |
| -------------------------------------------- | ---------- |
| `connection.username`, `connection.password` | Removed    |
| `connection.api_key`                         | New        |
| `remote_info.rack_key`                       | Removed    |
| State file permissions                       | Owner-only |

The config file the Core writes for the embedded Driver changes the same way.

**Client libraries**

| Library                            | Change                                                                             |
| ---------------------------------- | ---------------------------------------------------------------------------------- |
| TypeScript `synnaxParamsZ`         | Adds `apiKey`. `username` and `password` become optional                           |
| Python `Synnax(...)` and `Options` | Adds `api_key`                                                                     |
| Python `sy login`                  | Creates an API key and saves it to the keyring, in place of the password           |
| C++ `synnax::Config`               | Adds `api_key`. The `synnax` and `seldon` defaults apply only when no key is given |
| C interface `synnax_client_open`   | Gains an API key parameter, for LabVIEW                                            |

**Console**

| Setting                       | Change                                        |
| ----------------------------- | --------------------------------------------- |
| Saved Core record, `password` | Removed                                       |
| Session token                 | New. Keychain on desktop, cookie in a browser |

**Synnax Desktop**: No change.

### 4.13 Does it extend?

- **A service account**: One row in the subject map and a `DeleteFor` call on delete.
  The authentication service does not change.
- **A certificate**: A smart card without a provider, or a Core node calling a peer,
  adds one package with a table and an authenticator, and one `Proof` variant. It also
  needs the Core to request client certificates, which RFC 0045 §4.6 deferred.
- **Back-channel logout**: Keycloak can
  [call an endpoint](https://www.keycloak.org/docs/latest/server_admin/index.html) when
  a user is logged out or disabled. One endpoint validates the signed logout token and
  deletes the sessions of the matching link. Entra does not support this.

## 5 What this RFC does not cover

- Roles, policies, and scoped API keys.
- Mapping a provider's `groups` claim to roles.
- The audit log.
- Authentication between Core nodes.
- Smart cards without a provider, SAML, and LDAP.
- Second factors.
- A limit on failed logins.
- A Console screen lock after a person is inactive.
- FIPS mode beyond password hashing.

## 6 Implementation phases

Each phase is one pull request into `main`.

- **Phase 1: Passwords.** The `Authenticator` interface, the password package, the
  migration from `SecureCredentials`, tagged hashes with Argon2id, and `fips`.
- **Phase 2: Sessions.** The session table, `Login`, `Authenticate`, `Logout`, the idle
  sweep, and the middleware. Deletes the `token` package.
- **Phase 3: Streams.** Close when their session ends, and count as activity.
- **Phase 4: Clients.** Session endpoints, and login, renewal, and logout in the three
  client libraries.
- **Phase 5: Console sessions.** Keychain, cookie, no saved password, and logout.
- **Phase 6: API keys.** The authenticator, the key endpoints, and client bindings.
- **Phase 7: Racks.** The Driver role, enrollment in `synnax-driver login`, and the
  embedded Driver key.
- **Phase 8: Console credentials.** The credentials list on user and rack pages.
- **Phase 9: Accepted methods.** `auth-methods`, `auth/methods`, and the login page.
- **Phase 10: OpenID Connect.** Provider settings, the authenticator, the Console login
  flow, and first-login users.
- **Phase 11: User record.** Moves `username` off the user record and merges the name
  fields.
- **Phase 12: Cleanup.** Removes the old login endpoint.

### 6.0 Compatibility

- Phase 1 migrates every stored credential. Nobody resets a password.
- Every existing token stops working at Phase 2. Clients log in again without help.
- The old `auth/login` endpoint keeps working for one stable release.
- No Driver config needs a manual edit. At its first start after the upgrade, a Driver
  with a username and password logs in once, creates an API key for the rack it already
  has, and saves the key to its state file. A key in the state file wins over a username
  and password, so a password left in a config file or environment variable is ignored,
  with a warning.
- The Console deletes its saved password after the first login with it succeeds. The
  Python CLI does the same, and creates and saves an API key in that step.
