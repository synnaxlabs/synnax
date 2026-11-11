# 101 Secrets and certificates

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-03
- **Related**: [RFC 0073 - Authentication](0073-authentication.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0089 - OPC UA certificates](0089-opc-ua-certificates.md),
  [RFC 0100 - Gorp references](0100-gorp-references.md)

## 0 Summary

Two new Core services hold what devices use to log in and to secure their connections.
The secret service holds hidden values: passwords, tokens, API keys, and private keys.
After a value is saved, only the program that connects can read it: the Core for its own
integrations, and a Driver for the devices on its rack. The certificate service holds
public certificates, shows their details, and warns before they expire. A device holds
keys to these items, never values or paths. A connect form picks an item from one list,
or adds one by typing a value or dropping a file. A new file integration lets the
Console browse the disk of the computer that uses a file.

## 1 Motivation

- **Every user can read every credential**: Kafka, MQTT, OPC UA, HTTP, and PagerDuty
  keep passwords and tokens in plain text (`schemas/synnax/kafka.oracle:20-46`,
  `core/pkg/service/mqtt/properties.go:65-89`,
  `driver/opcua/connection/connection.h:34-92`,
  `console/src/feature/http/device/types.ts:21-50`,
  `schemas/synnax/pagerduty.oracle:46`). A Viewer retrieves every device with its
  properties (`core/pkg/api/device/device.go:171`) and can log in to the equipment
  directly.
- **A connection test sends the password to every Viewer**: Connect dialogs put the
  connection in a `test_connection` command
  (`console/src/feature/opcua/device/useConnectModal.tsx:106-110`), and streaming
  `sy_task_cmd` needs only framer retrieve (`core/pkg/api/framer/framer.go:307-311`).
- **The Driver writes passwords back**: When the OPC UA health check pins a server
  certificate, the scanner writes the whole device again
  (`driver/common/scan_task.h:565-578`).
- **Paths work on one computer**: MQTT reads certificate files on the Core's computer
  (`core/pkg/service/mqtt/properties.go:207-235`), and OPC UA on the Driver's
  (`driver/opcua/certificate/certificate.cpp:242`). With a remote Core or the browser
  Console, nobody can fill them without a shell.
- **Certificates are opaque**: The Console shows only an OPC UA pin's thumbprint
  (`console/src/feature/opcua/device/Certificate.tsx:56`). Nothing shows a saved
  certificate's subject, issuer, or expiry, or warns before it expires. HTTP takes no
  certificate at all, so a self-signed server needs verification off
  (`driver/http/processor/processor.cpp:107`).

## 2 Principles

1. **A device holds keys, never values**: Devices, tasks, commands, statuses, and
   exports carry item keys only, so no path has to hide a field.
2. **Only the connecting program reads a secret**: No person reads a value back, Owner
   included.
3. **A rack reads only what its rack uses**: A Driver reads a secret only when a device
   or task on its rack uses it, or a grant names it (§3.3).
4. **Certificates move as contents**: A path stays possible only as an item that names a
   file on the computer that uses it.
5. **No user picks the file behind a secret**: A file secret names a file in a folder
   that an admin sets on its rack (§3.5).

## 3 Design

### 3.0 Secrets

```
Secret struct {
    key       uuid = create
    name      string
    source    Source = stored   // stored | file
    file_name string = ""       // file: a file in the rack's secrets folder
    rack      rack.Key = 0      // file: the rack that reads the file
    value     bytes             // stored: ciphertext
    updated   telem.TimeStamp
}
```

The schema is `schemas/synnax/secret.oracle`, the package `core/pkg/service/secret`, and
the resource type `secret`. `value` carries `@go wire omit`, `@ts omit`, and `@py omit`,
as the session hash does (`schemas/synnax/auth.oracle:33-38`), so no endpoint or signal
can carry it. Secrets have no exporter. The Console shows `updated` as "Set on <date>".

| Endpoint              | Caller                                        | Does                                       |
| --------------------- | --------------------------------------------- | ------------------------------------------ |
| `secret/set`          | Create for a new secret, update for a replace | Saves a value or file name, or replaces it |
| `secret/retrieve`     | Can retrieve secrets                          | Returns records without values             |
| `secret/dispatch`     | Can update secrets                            | Applies edits as actions, such as a rename |
| `secret/delete`       | Can delete secrets                            | Refuses while anything uses the secret     |
| `secret/grant/create` | Can create devices and retrieve the secrets   | Stores a one-time grant (§3.3)             |
| `secret/resolve`      | A rack only                                   | Returns stored values; refuses file values |

The write binder (`core/pkg/api/bind/write.go`) makes the create, update, and delete
checks on each record that a write changes. Owner and Engineer get every action on
`secret` and `certificate`. Operator and Viewer retrieve them. The Driver role also
retrieves secrets, and retrieves and creates certificates, for pins (§3.6).

### 3.1 Certificates

```
Certificate struct {
    key              uuid = create
    name             string
    source           Source = stored
    pem              string = ""      // stored: the chain, leaf first
    private_key      secret.Key = ""  // stored: optional
    certificate_path string = ""      // file
    private_key_file string = ""      // file: optional, in the secrets folder
    rack             rack.Key = 0     // file: the rack that reads the paths
    subject          string
    issuer           string
    not_before       telem.TimeStamp
    not_after        telem.TimeStamp
    sha1             string
    sha256           string
}
```

The package is `core/pkg/service/certificate`. For a stored certificate, the Core parses
`pem` with `crypto/x509` on every write and fills the details, so no client parses a
certificate. A write may carry a private key as PEM: the Core checks that it matches the
leaf and saves it as a stored secret in the same transaction, so the caller must also be
able to write that secret. The Core never takes a `private_key` from the caller, so the
key of a stored certificate is always a stored secret, and a replace cannot point it at
a file secret of another rack. A `pem` that holds a private key is refused, so a key
never lands in the public record. For a file certificate, the details come from the file
browser and from the reader (§3.5). Certificates are public: anyone who can retrieve one
can download it as PEM or DER. The endpoints match the secret endpoints, without grants
and resolve, and add `certificate/refresh` (§3.5).

### 3.2 Encryption key

The Core encrypts each value with AES-256-GCM under a random data key, with the secret's
key as associated data so a value cannot move to another record. The Core stores the
data key encrypted under the secrets key. A new secrets key, or one that a key service
such as Vault holds later, needs only the data key encrypted again, not every value.

At first start, the Core writes a random secrets key to `<data>/secrets.key` with mode
0600, beside the embedded Driver key (`core/pkg/driver/driver.go:59`).
`--secrets-key-file` reads it from another path, and `SYNNAX_SECRETS_KEY` passes it as
base64. A memory-mode Core keeps a key in memory. The Core refuses to start when the
secrets key cannot decrypt the data key, so a wrong key fails once instead of in every
task. Every node of a cluster needs the same key, so a node that joins starts with the
first node's key through `--secrets-key-file` or `SYNNAX_SECRETS_KEY`. A joining node
that makes its own key refuses to start.

### 3.3 Who reads a secret

Only a rack (RFC 0073 §4.8) can call `secret/resolve`. Every other subject, Owner
included, gets a permission error. The Core's integrations read their secrets in process
with no check, because they run in the Core. A rack reads a secret when:

1. A device or task on the rack uses it, directly or through a certificate.
2. The request carries a grant for that rack and the task of the command, and the grant
   names it.

A user who saves a device or task that names a secret must be able to retrieve that
secret, or the save fails. Otherwise a user could make a rack or a Core integration use
a secret the user cannot see. A secret reference is a use link, so the write binder
makes this check on every save (`core/pkg/api/bind/write.go:51`). Phase 7 adds the first
rule, with the first Driver integration.

A grant covers commands that run before a device is saved: a connect dialog's test and
browse. A test reads secrets only with a grant, even for a saved device. The Console
calls `secret/grant/create` with a subject, a task, the secrets, and the command to run,
and the Core checks that the caller can create devices and retrieve each secret. Only a
rack can be a subject for now, and the task must be on that rack. For a certificate, the
Console sends the secret in its `private_key`. The Core stores the grant with an expiry
one minute away and returns its key. The Console then sends that task a command that
carries only the key.

The Driver resolves the key with the task of the command. The Core refuses a grant for a
different task, deletes the grant, and returns the stored command with the values of its
secrets. The Driver runs the stored command, never the arguments on the stream. A grant
that no resolve uses ends at its expiry. The key travels on `sy_task_cmd`, where an
Operator can read and write any command
(`core/pkg/service/access/rbac/builtin/builtin.go:197-205`). A copied key runs only the
Console's own command, against the Console's server, even when the Driver drops or never
sees the Console's command. At worst, a copied key spends the grant first and the
Console's test fails. Without grants, an Operator could send a saved password to a
server of their choice.

A user who can edit a device can still point it at their own server, as in Grafana and
Ignition. Write-only keeps secrets from people who can only view, and out of statuses,
streams, and exports.

### 3.4 References

Each field that names an item is a Gorp reference (RFC 0100, on `main` in
`gorp/table_references.go`) with `@ref kind use` and `@ref on_delete restrict`, such as
PagerDuty's `routing_key` and a certificate's `private_key`. Gorp refuses a write that
names a missing item, and keeps a reverse index. A delete of an item in use fails with a
`gorp.RestrictError`, which `core/pkg/api/restrict.go` rewrites to name the task or
device. A task config lives in its own table, so the API maps it to its task with
`config.TaskOf` (`core/pkg/service/task/config/config.go`). The Console's "Used by" list
and the rack rule (§3.3) read the index through `resource.Dependents`. A certificate is
a union on its source, and Oracle reads `@ref` inside a union variant (#4920).

### 3.5 File items and the file integration

A file item names the rack that reads it: a Driver's rack for OPC UA and HTTP, or the
Core's own rack (`core/pkg/service/driver/driver.go:92`) for Kafka, MQTT, and PagerDuty.
A file certificate names the path of its certificate and the name of its key file in the
rack's secrets folder. A file secret names only a file in that folder. An admin sets the
folder with `secrets_dir` in the Driver's config (`driver/rack/file.cpp`), or with
`--secrets-dir` on the Core. With no folder set, a file secret fails at connect. The
Core refuses a file name that is `.` or `..`, or that holds `/`, `\`, or `:` (a Windows
drive), so a file secret cannot reach outside its folder. The same rule holds for the
key file of a certificate. A secret's bytes go to a server that the user picks, such as
a Kafka broker, so a path would let the user send any file that the reader can open, the
secrets key included. TLS sends no private key, but the reader signs with it, so a key
path would let the user connect as any system whose key the reader can open. A
certificate is public, so its path can point anywhere. Each integration's Core service
refuses a device or task that names a file item of another rack, directly or through a
certificate. A replace keeps the source of an item, so a stored secret never becomes a
file. A replace of an item in use keeps its rack, or fails as a delete does. So an item
has one file and one set of details. The reader opens the file at each connect, and the
value never enters the store.

A new integration, `file`, gives every rack one internal task, "File Browser", through
`InitialTasks` (`core/pkg/service/driver/factory.go:32`) in the Core and
`configure_initial_tasks` in the C++ Driver. Its one command, `list`, takes a path, or
nothing for the roots, the home folder, and the secrets folder when one is set. The
reply gives each entry's name, kind, size, and modified time. For a certificate file it
adds the subject, issuer, expiry, and fingerprints, which the Console saves with a file
certificate so expiry warnings work for files too. After a connect reads every file that
a certificate names, the reader compares the SHA-256 fingerprint of the certificate file
with the saved one. When they differ, it saves the new details. A Core reader saves them
in process. A rack calls `certificate/refresh`, which changes only the details. Like
`secret/resolve`, it skips the write binder: the Core checks that the caller is the rack
that the certificate names. So a replaced file gets new details at its next connect, and
a typed path gets its first details there, with or without the file browser. For a key
file it gives the name only. The whole disk is visible, as it is to a path typed by
hand. A site that does not want browsing leaves `file` out of the Driver's integrations.

### 3.6 Integrations

| Integration | Field                                      | Today        | After                             |
| ----------- | ------------------------------------------ | ------------ | --------------------------------- |
| Kafka       | `sasl.password`                            | Plain text   | Secret                            |
| Kafka       | `ca_certificate`                           | PEM          | Certificate                       |
| Kafka       | `client_certificate`                       | None         | Certificate                       |
| MQTT        | `password`                                 | Plain text   | Secret                            |
| MQTT        | `ca_file`                                  | Core path    | `ca_certificate`, certificate     |
| MQTT        | `cert_file`, `key_file`                    | Core paths   | `client_certificate`, certificate |
| OPC UA      | `password`                                 | Plain text   | Secret                            |
| OPC UA      | `client_certificate`, `client_private_key` | Driver paths | `client_certificate`, certificate |
| OPC UA      | `server_certificate`                       | PEM          | Certificate                       |
| HTTP        | `auth.token`, `auth.key`, `auth.password`  | Plain text   | Secret                            |
| HTTP        | `ca_certificate`, `client_certificate`     | None         | Certificate                       |
| PagerDuty   | `routing_key`                              | Plain text   | Secret                            |

An empty OPC UA `client_certificate` still selects the Driver certificate (RFC 0089
§3.0). Go builds TLS from contents with `tls.X509KeyPair`, OPC UA loads contents into
open62541, and HTTP sets `CURLOPT_CAINFO_BLOB`, `CURLOPT_SSLCERT_BLOB`, and
`CURLOPT_SSLKEY_BLOB` (curl 8.23, `MODULE.bazel:16`). An OPC UA pin on first connect
(RFC 0089 §3.2) saves the server's certificate as an item named after the device, and
the device write carries only its key.

### 3.7 Console

A Credentials toolbar lists secrets and certificates in two groups, with "Used by",
Replace, and Delete. Every connect form uses two shared fields:

- **Secret field**: Shows the name and "Set on <date>". It picks from the list, or adds
  a secret from a typed value.
- **Certificate field**: Shows the subject, issuer, expiry, and SHA-256 fingerprint,
  with Download. It picks from the list, or adds a certificate from a dropped file
  (`console/src/platform/fs/DropZone.tsx`), pasted PEM, or the file browser on the
  device's rack, with an optional key file.

Both fields list only the file items of the device's rack. A new item is named after its
device and field, such as "Plant OPC UA password". The dialog saves it before its first
test, and Cancel deletes the items it made that nothing uses. OPC UA's Trust button
saves the offered certificate as an item.

### 3.8 Expiry

Every hour, the certificate service writes each certificate's status under
`certificate:<key>`, only on change. It is the success "Certificate valid" until 30 days
before `not_after`. Then it is the warning "Certificate expires soon", with the date in
the description. After `not_after`, it is the error "Certificate expired". A file
certificate with no details, such as a typed path, gets no status. Each health check
gives a device that uses the certificate the same status, so RFC 0092 toasts it only
when the device is in use.

The Core's CA (RFC 0045) and the certificate that each listener presents, from any
source, get the same statuses, so a site learns before its Core stops taking
connections. `synnax cert ca` makes a CA that is valid for one year
(`core/pkg/security/cert/x509.go:29`). These certificates stay out of the store, because
the Core needs them before it opens its database.

### 3.9 Migration and clients

v0.60.1 shipped the OPC UA, HTTP, and PagerDuty fields. At first start, each
integration's Go service migrates its own records. Each password, token, API key, and
routing key becomes a secret named "<device name> <field>". Each OPC UA path pair
becomes a file certificate whose key file is `<certificate key>.pem`, and each PEM pin a
stored certificate. The certificate key is unique, so two old keys with the same file
name, such as the `own/private/private.pem` of two OPC UA PKI folders, get two names. No
other record keeps the old path, so the migration writes the warning "Private key file
renamed" under `certificate:<key>`, with the old path and the new name in the
description. Until an admin copies the key into the secrets folder under the new name,
connect fails with an error that names the folder and the file. The reader saves details
only after a connect that reads every file (§3.5), and the expiry check writes no status
for a certificate with no details. So the warning stays until the certificate first gets
details, and the first expiry status after that replaces it. Details come from the
reader once the key is in place, or earlier from a replace that picks the certificate
file in the file browser. Kafka and MQTT have not shipped, so their fields change with
no migration. The Core, the Driver, and the Console change in one release.

The TypeScript and Python clients get `secrets` and `certificates` from Oracle, and the
C++ client gets `resolve` for the Driver. A script makes a secret with
`client.secrets.set(name=..., value=...)` and puts its key on the device.

## 4 Implementation phases

Phases 2 and 3 are the stores with no users, and Phase 4 their Console and first user.
Kafka and MQTT have not shipped, while OPC UA and HTTP need a migration that ships with
the Driver change, so they are separate phases.

| Phase | Scope                                                  | Touches                                                                                                                                                                                                                                          | Depends on |
| ----- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- |
| 1     | Add RFC 0101 for secrets and certificates              | `docs/tech/rfc`                                                                                                                                                                                                                                  | -          |
| 2     | Add the secret service                                 | `schemas/synnax`, `core/pkg/service/secret`, `core/pkg/service/auth`, `core/pkg/service/access`, `core/pkg/service/layer.go`, `core/pkg/api`, `core/cmd/start`, `client/ts`, `client/py`                                                         | 1          |
| 3     | Add the certificate service                            | `schemas/synnax`, `core/pkg/service/certificate`, `core/pkg/service/access`, `core/pkg/service/layer.go`, `core/pkg/api`, `client/ts`, `client/py`                                                                                               | 2          |
| 4     | Pick secrets in the Console and move PagerDuty to them | `console/src/feature/secret`, `console/src/feature/pagerduty`, `core/pkg/service/pagerduty`, `schemas/synnax`                                                                                                                                    | 3          |
| 5     | Add the file integration and remote file browser       | `driver/file`, `driver/rack`, `core/cmd/start`, `core/pkg/service/file`, `core/pkg/service/secret`, `core/pkg/service/certificate`, `core/pkg/service/pagerduty`, `schemas/synnax`, `core/pkg/service/layer.go`, `console/src/feature/file`      | 4          |
| 6     | Move Kafka and MQTT to secrets and certificates        | `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `core/pkg/service/certificate`, `schemas/synnax`, `console/src/feature/kafka`, `console/src/feature/mqtt`, `client/py`                                                                        | 5          |
| 7     | Move OPC UA and HTTP to secrets and certificates       | `driver/opcua`, `driver/http`, `client/cpp`, `core/pkg/service/secret`, `core/pkg/api/certificate`, `core/pkg/service/opcua`, `core/pkg/service/http`, `console/src/feature/opcua`, `console/src/feature/http`, `client/py`, `integration/tests` | 5          |
| 8     | Report the expiry of the Core's own certificates       | `core/pkg/service/certificate`, `core/pkg/security/cert`, `core/pkg/service/layer.go`                                                                                                                                                            | 3          |

## 5 Resolved decisions

1. **Separate records, not hidden fields**: A credential shares its device's read rule.
   Hiding a field must happen on every path, and a client that writes it back erases it.
2. **One shared list**: As in Ignition 8.3 and HighByte, many devices use one item, and
   a password changes in one place. The trade is real: delete must check users.
3. **Nobody reads a secret back**: As in Grafana, Ignition, Node-RED, and GitHub. The
   trade is real: a forgotten password must come from the equipment's admin.
4. **One value per secret, not a login**: A username is not secret and stays on the
   device. The trade is real: a shared username changes on each device.
5. **Two services, with the key as a secret**: Certificates are public and parsed;
   secrets are hidden and opaque. The certificate points to its key, so a device picks
   one item.
6. **The Core holds the key, beside its data**: Kafka, MQTT, and PagerDuty run in the
   Core and must decrypt anyway, and the Core mints rack credentials, so sealing to each
   rack would protect nothing more. The default needs no setup, as in n8n and Grafana.
   The trade is real: a copy of the data folder with its key reads every secret.
7. **A rack reads what its devices and tasks use**: A rack needs no grant for its own
   saved work. The trade is real: the Driver role can save any device and retrieve any
   secret (`core/pkg/service/access/rbac/builtin/builtin.go:136-155`), so a stolen rack
   key can name any secret in a device on its rack and then read it. SY-5597 limits each
   rack to its own devices and tasks.
8. **One-time grants, not save then test**: Grafana saves before it tests. Synnax keeps
   the test before save (RFC 0079 §4.2). A grant works once, as an AWS KMS grant retires
   when its work is done, and holds its command, so a copied key can only rerun the
   Console's command. A hash of the connection settings would also bind it, but the
   Console and the Driver would have to hash the same fields the same way in two
   languages. A Console that deletes the grant when its command gets no reply leaves the
   key open while it waits. A grant names a subject, not a session or a connection, so
   other subjects can use grants later. Unlike the sessions of RFC 0073 §4.4, a grant
   has a clock, because no person ends a grant that no test uses. The trade is real: a
   stored write for each test, and an Operator who copies a key can spend it first and
   fail the Console's test.
9. **Uploaded private keys**: This replaces RFC 0089 principle 1 and decision 2, whose
   reason, commands that leak to every Viewer, is gone. Grafana, HighByte, and
   Ignition's MQTT modules take uploads. The trade is real: a key lives in the Core too,
   against OPC UA Part 4 §6.1.2.
10. **Paths as items, not fields**: A site whose keys never leave their computer still
    works. The trade is real: a file item works on one rack, so two racks that hold the
    same file need two items.
11. **The file browser is its own integration**: A site can turn it off. Its reply is a
    task status, as OPC UA browse replies are. The trade is real: a Viewer can see file
    names in a reply, and an Operator can list folders, as with any command.
12. **References in Gorp, not found on demand**: Gorp refuses a delete or a missing item
    with no code in each integration, and "Used by" reads an index instead of a scan.
    The trade is real: each field that names an item must be a typed schema field, and
    each write updates the index.
13. **Pins become items**: A pinned server certificate gets details and expiry warnings.
    The trade is real: one list entry per pinned server.
14. **Set, not create**: A write carries the whole item, so one `set` creates it or
    replaces it. A rename goes through dispatch, as edits to labels and ranges do. A
    replace needs only update access. The trade is real: a reused key replaces an item
    without a warning.
15. **File secrets read from a folder that an admin sets**: Kafka Connect let a user
    name any file and sent its bytes to the user's broker (CVE-2024-31141). Its fix,
    KIP-993, lists the folders that an admin allows. Docker, Kubernetes, and Telegraf
    read secrets by name from a folder that the operator sets, and Vault Agent and the
    Kubernetes CSI driver write secrets into such a folder. Files for certificates only,
    as in Node-RED, also closes the leak, but keeps every secret of a site in Synnax. A
    certificate's key file follows the same rule: the reader signs with the key, so a
    key path would let a user connect as any system whose key the reader can open. The
    trade is real: an admin must set the folder on each rack before a file secret works,
    and must copy each OPC UA key into it under its new name.

## 6 What this RFC does not cover

- Core listener certificates (RFC 0045), other than their expiry (§3.8), and cluster
  encryption (RFC 0009).
- External secret providers such as Vault. A `vault` source next to `stored` and `file`
  fits later, and so does a secrets key that a key service holds (§3.2).
- The Driver's own OPC UA certificate, which stays in the Driver's directory.
- Credentials typed into HTTP task headers or query parameters. The device's API key
  auth sends a credential header.
- Permissions on task commands in general.
