# 101 Secrets and certificates

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-03
- **Related**: [RFC 0073 - Authentication](0073-authentication.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0089 - OPC UA certificates](0089-opc-ua-certificates.md)

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
  properties (`core/pkg/api/device/device.go:185`) and can log in to the equipment
  directly.
- **A connection test sends the password to every Viewer**: Connect dialogs put the
  connection in a `test_connection` command
  (`console/src/feature/opcua/device/useConnectModal.tsx:106-110`), and streaming
  `sy_task_cmd` needs only framer retrieve (`core/pkg/api/framer/framer.go:446-450`).
- **The Driver writes passwords back**: When the OPC UA health check pins a server
  certificate, the scanner writes the whole device again
  (`driver/common/scan_task.h:560-573`).
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

## 3 Design

### 3.0 Secrets

```
Secret struct {
    key     uuid = create
    name    string
    source  Source = stored     // stored | file
    path    string = ""         // file: a path on the computer that uses it
    value   bytes               // stored: ciphertext
    updated telem.TimeStamp
}
```

The schema is `schemas/synnax/secret.oracle`, the package `core/pkg/service/secret`, and
the ontology type `secret`. `value` carries `@go wire omit`, `@ts omit`, and `@py omit`,
as the session hash does (`schemas/synnax/auth.oracle:33-38`), so no endpoint or signal
can carry it. Secrets have no exporter. The Console shows `updated` as "Set on <date>".

| Endpoint          | Caller               | Does                                   |
| ----------------- | -------------------- | -------------------------------------- |
| `secret/create`   | Can create secrets   | Saves a name and a value or path       |
| `secret/retrieve` | Can retrieve secrets | Returns records without values         |
| `secret/set`      | Can update secrets   | Renames, or replaces the value or path |
| `secret/delete`   | Can delete secrets   | Refuses while anything uses the secret |
| `secret/grant`    | Can create devices   | Returns a grant for a rack (§3.3)      |
| `secret/resolve`  | A rack only          | Returns values (§3.3)                  |

Owner and Engineer get every action on `secret` and `certificate`. Operator and Viewer
retrieve them. The Driver role also retrieves and creates certificates, for pins (§3.6).

### 3.1 Certificates

```
Certificate struct {
    key              uuid = create
    name             string
    source           Source = stored
    pem              string = ""      // stored: the chain, leaf first
    private_key      secret.Key = ""  // stored: optional
    certificate_path string = ""      // file
    private_key_path string = ""      // file: optional
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
leaf and saves it as a secret in the same transaction. For a file certificate, the
details come from the file browser (§3.5). Certificates are public: anyone who can
retrieve one can download it as PEM or DER. The endpoints match the secret endpoints,
without grant and resolve.

### 3.2 Encryption key

The Core encrypts each value with AES-256-GCM under one 32-byte key, with the secret's
key as associated data so a value cannot move to another record. At first start, the
Core writes a random key to `<data>/secrets.key` with mode 0600, beside the embedded
Driver key (`core/pkg/driver/driver.go:58`). `--secrets-key-file` reads it from another
path, and `SYNNAX_SECRETS_KEY` passes it as base64. A memory-mode Core keeps a key in
memory. The Core stores a check value and refuses to start when the key cannot decrypt
it, so a wrong key fails once instead of in every task. Every node of a cluster needs
the same key.

### 3.3 Who reads a secret

Only a rack (RFC 0073 §4.8) can call `secret/resolve`. Every other subject, Owner
included, gets a permission error. The Core's integrations run the same check in
process, as the Core's rack. A rack reads a secret when:

1. A device or task on the rack uses it, directly or through a certificate.
2. The request carries a grant for that rack that names it.

A grant covers commands that run before a device is saved: a connect dialog's test and
browse. The Console calls `secret/grant` with a rack and items, and the Core checks that
the caller can create devices. The grant names the rack and the secrets, where a
certificate stands for its key. It expires in 60 s and is signed with HMAC-SHA-256 under
a key derived from the store key. The Console sends it with the command. Only that rack
can use it, so it can travel on `sy_task_cmd`. Without grants, an Operator, who can
write any command (`core/pkg/service/access/rbac/builtin/builtin.go:171-179`), could
send a saved password to a server of their choice.

A user who can edit a device can still point it at their own server, as in Grafana and
Ignition. Write-only keeps secrets from people who can only view, and out of statuses,
streams, and exports.

### 3.4 References

The Core finds what uses an item by reading device properties and task configs when it
needs to, with no stored index. Each integration's Go service gives a function from its
properties or config to the item keys in them. `core/pkg/service/layer.go` passes a map
from make to function into a finder that both services take in their config. OPC UA and
HTTP get small Go structs for their item fields in `core/pkg/service/opcua` and
`core/pkg/service/http`. The finder serves the rack rule, delete, and the Console's
"Used by" list. A Core holds hundreds of devices, so a scan costs little.

### 3.5 File items and the file integration

A file item names a path on the computer that uses it: the Driver's computer for OPC UA
and HTTP, the Core's for Kafka and MQTT. The reader opens the file at each connect, and
the value never enters the store.

A new integration, `file`, gives every rack one internal task, "File Browser", through
`InitialTasks` (`core/pkg/service/driver/factory.go:32`) in the Core and
`configure_initial_tasks` in the C++ Driver. Its one command, `list`, takes a path, or
nothing for the roots and home folder. The reply gives each entry's name, kind, size,
and modified time. For a certificate file it adds the subject, issuer, expiry, and
fingerprints, which the Console saves with a file certificate so expiry warnings work
for files too. For a key file it gives the name only. The whole disk is visible, as it
is to a path typed by hand. A site that does not want browsing leaves `file` out of the
Driver's integrations.

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

A Secrets toolbar lists secrets and certificates in two groups, with "Used by", Replace,
and Delete. Every connect form uses two shared fields:

- **Secret field**: Shows the name and "Set on <date>". It picks from the list, or adds
  a secret from a typed value.
- **Certificate field**: Shows the subject, issuer, expiry, and SHA-256 fingerprint,
  with Download. It picks from the list, or adds a certificate from a dropped file
  (`console/src/platform/fs/DropZone.tsx`), pasted PEM, or the file browser on the
  device's rack, with an optional key file.

A new item is named after its device and field, such as "Plant OPC UA password". The
dialog saves it before its first test, and Cancel deletes the items it made that nothing
uses. OPC UA's Trust button saves the offered certificate as an item.

### 3.8 Expiry

Every hour, the certificate service writes each certificate's status under
`certificate:<key>`, only on change. Within 30 days of `not_after` it is the warning
"Certificate expires soon", with the date in the description. After, it is the error
"Certificate expired". Each health check gives a device that uses the certificate the
same status, so RFC 0092 toasts it only when the device is in use.

### 3.9 Migration and clients

v0.60.1 shipped the OPC UA, HTTP, and PagerDuty fields. At first start, each
integration's Go service migrates its own records. Each password, token, API key, and
routing key becomes a secret named "<device name> <field>". Each OPC UA path pair
becomes a file certificate, and each PEM pin a stored certificate. Kafka and MQTT have
not shipped, so their fields change with no migration. The Core, the Driver, and the
Console change in one release.

The TypeScript and Python clients get `secrets` and `certificates` from Oracle, and the
C++ client gets `resolve` for the Driver. A script makes a secret with
`client.secrets.create(name=..., value=...)` and puts its key on the device.

## 4 Implementation phases

Phase 2 is the store with no users, and Phase 3 its Console and first user. Kafka and
MQTT have not shipped, while OPC UA and HTTP need a migration that ships with the Driver
change, so they are separate phases.

| Phase | Scope                                                  | Touches                                                                                                                                                                                         | Depends on |
| ----- | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Add RFC 0101 for secrets and certificates              | `docs/tech/rfc`                                                                                                                                                                                 | -          |
| 2     | Add the secret and certificate services                | `schemas/synnax`, `core/pkg/service/secret`, `core/pkg/service/certificate`, `core/pkg/service/access`, `core/pkg/service/layer.go`, `core/pkg/api`, `core/cmd/start`, `client/ts`, `client/py` | 1          |
| 3     | Pick secrets in the Console and move PagerDuty to them | `console/src/feature/secret`, `console/src/feature/pagerduty`, `core/pkg/service/pagerduty`, `schemas/synnax`                                                                                   | 2          |
| 4     | Add the file integration and remote file browser       | `driver/file`, `driver/rack`, `core/pkg/service/file`, `core/pkg/service/layer.go`, `console/src/feature/file`                                                                                  | 3          |
| 5     | Move Kafka and MQTT to secrets and certificates        | `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `schemas/synnax`, `console/src/feature/kafka`, `console/src/feature/mqtt`, `client/py`                                                       | 3          |
| 6     | Move OPC UA and HTTP to secrets and certificates       | `driver/opcua`, `driver/http`, `client/cpp`, `core/pkg/service/opcua`, `core/pkg/service/http`, `console/src/feature/opcua`, `console/src/feature/http`, `client/py`, `integration/tests`       | 4          |

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
7. **A rack reads only what its rack uses**: This is narrower than the Driver role's
   device access (`core/pkg/service/access/rbac/builtin/builtin.go:117-135`), so a
   stolen test-stand computer reads only its own secrets.
8. **Grants, not save then test**: Grafana saves before it tests. Synnax keeps the test
   before save (RFC 0079 §4.2). The trade is real: a signed grant type.
9. **Uploaded private keys**: This replaces RFC 0089 principle 1 and decision 2, whose
   reason, commands that leak to every Viewer, is gone. Grafana, HighByte, and
   Ignition's MQTT modules take uploads. The trade is real: a key lives in the Core too,
   against OPC UA Part 4 §6.1.2.
10. **Paths as items, not fields**: A site whose keys never leave their computer still
    works. The trade is real: a file item works on one computer.
11. **The file browser is its own integration**: A site can turn it off. Its reply is a
    task status, as OPC UA browse replies are. The trade is real: a Viewer can see file
    names in a reply, and an Operator can list folders, as with any command.
12. **References found on demand**: The device stays the one source of truth. The trade
    is real: a scan per resolve and delete.
13. **Pins become items**: A pinned server certificate gets details and expiry warnings.
    The trade is real: one list entry per pinned server.

## 6 What this RFC does not cover

- Core listener certificates (RFC 0045) and cluster encryption (RFC 0009).
- The Driver's own OPC UA certificate, which stays in the Driver's directory.
- Credentials typed into HTTP task headers or query parameters. The device's API key
  auth sends a credential header.
- Permissions on task commands in general.

## 7 Open questions

- The toolbar's name.
- The expiry warning lead time, 30 days here.
