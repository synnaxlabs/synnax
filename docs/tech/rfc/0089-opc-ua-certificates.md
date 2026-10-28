# 89 OPC UA certificates

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-27
- **Related**:
  [RFC 0045 - Serving Core on multiple listeners with per-listener certificates](0045-core-multi-listener-per-listener-certs.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0081 - Typed devices](0081-typed-devices.md)

## 0 Summary

Each Driver issues, keeps, and renews its own OPC UA client certificate. Its private key
never leaves the Driver's computer. Each device pins its server's certificate: the
connect dialog asks the user to trust it, and a saved device without a pin trusts the
first certificate it sees. After that, the Driver rejects any other certificate from
that server. A switch turns validation off as a troubleshooting tool. The native file
pickers go away.

## 1 Motivation

- **The Driver reads the paths on a different computer**: The dialog picks a path on the
  Console's computer (`console/src/platform/runtime/files.ts:152`), and the Driver opens
  it on its own (`driver/opcua/connection/connection.cpp:63`). With a remote rack, the
  file is not there. The browser Console cannot fill the fields at all.
- **The Driver trusts every server**: `configure_encryption` passes
  `trust_list_size = 0` (`connection.cpp:179`), so open62541 keeps its accept-all check
  (`plugins/ua_config_default.c:1154`). The server certificate field has no effect, and
  the docs say so
  (`site/docs/src/pages/reference/hardware/opc-ua/connect-server.mdx:129`). The OPC
  Foundation calls this "no authentication": a man in the middle can read the username
  and password.
- **Security needs files that users make by hand**: With no client certificate, every
  secure endpoint fails as `BadIdentityTokenRejected` (`connection_test.cpp:285`).
- **Two policies broke silently**: Basic256 and Basic128Rsa15 worked in v0.48, which
  shipped open62541 v1.4.3. PR #1695 moved to v1.4.14, which builds them only with
  `UA_INCLUDE_INSECURE_POLICIES` (`ua_config_default.c:1206`). The Console still offers
  them (`SelectSecurityPolicy.tsx:38-43`), and they fail as "No suitable endpoint".
- **Browse stores the password**: The browse reply copies the full connection into the
  task status (`driver/opcua/scan_task.cpp:204`), which the Core keeps in its status
  table (`core/pkg/service/status/service.go:104`). Any Viewer can read it.
- **Failures lose their cause**: `test_connection` reports only the description text
  (`scan_task.cpp:231`). A file that does not load reaches the user as an identity token
  error.

## 2 Principles

1. **The private key stays on the Driver's computer**: OPC UA Part 4 §6.1.2 says so.
   Only public certificates go to the Core.
2. **A server certificate never changes silently**: After a device pins a certificate,
   the Driver rejects every other certificate until a user trusts it.
3. **Certificates move as contents**: A path has meaning only on the computer that reads
   it.

## 3 Design

### 3.0 The Driver certificate

The first time a command or task needs it, the OPC UA factory loads the Driver's
certificate. If there is none, it makes one with `UA_CreateCertificate`
(`plugins/include/open62541/plugin/create_certificate.h:38`):

- **Key**: RSA 2048 with SHA-256, which every policy accepts.
- **Validity**: 3 years, the default in Ignition and Kepware.
- **Identity**: SAN `urn:synnax:driver:<uuid>` and the host name. The URI stays the same
  across renewals and regeneration. open62541 gives every certificate serial number 1,
  so each certificate gets the CN `Synnax Driver <uuid>` with a new UUID.
- **Files**: Part 12 Annex F layout in the state file's directory
  (`driver/rack/persist.cpp:28`): `opcua/pki/own/certs/` and `opcua/pki/own/private/`.
  The directory is 0700 and each key is 0600. On Windows, they inherit the profile
  directory's ACL.

All devices on a rack use this certificate, as in Ignition, Kepware, UaExpert, and OPC
Publisher. A device that moves to a different rack needs the server to trust that rack's
Driver too.

**Renewal**: 60 days before expiry, the Driver issues the next certificate and keeps
using the current one. Once a day, it connects with the next certificate to each server
that a secured device uses. A server that does not trust it yet puts it in its rejected
list, where its admin can approve it. The Driver switches when every server accepts the
next certificate, or at expiry.

Two new scan task commands join `browse` and `test_connection`
(`driver/opcua/scan_task.h:55`). `get_certificate` replies with the current and next
certificates as PEM, with SHA-1 thumbprints and expiry dates. `regenerate_certificate`
replaces both with a new pair, for a key that is no longer safe.

### 3.1 Files on the Driver's computer

`client_certificate` and `client_private_key` keep their meaning: paths on the Driver's
computer. Empty, the default, now selects the Driver certificate. These are the `auto`
and `file` sources of RFC 0045 §4.1, for sites that issue their own certificates.
Renewal does not apply to files. The Driver reports a file that it cannot read, or a key
that does not match its certificate, as its own error. A certificate with no SAN URI
connects with a fallback URI, which a server that checks the URI rejects. Keys with a
password stay unsupported.

### 3.2 Server trust

`server_certificate` holds the pinned server certificate as PEM. After
`UA_ClientConfig_setDefaultEncryption`, the Driver replaces `verifyCertificate` with its
own check: the DER bytes must equal the pin, and the certificate must be valid at the
Driver's time. The open62541 trust list check is not used, because it fails a CA-issued
server certificate as `BadCertificateChainIncomplete`. The check keeps the certificate
that the server presented, so the Driver knows which side refused.

- **Connect dialog**: `test_connection` never pins, and with no pin it rejects every
  certificate. Its failure reply carries the status code name and the side that refused.
  If the Driver refused, the reply also carries the server certificate with its subject,
  thumbprint, and expiry.
- **Saved device with no pin**: A connection from a task or the health check accepts the
  first certificate, then only that one for the life of the connection, so an automatic
  reconnect cannot reach another server. Only the health check writes the certificate to
  the device, as the scanners write devices (`driver/common/scan_task.h:162`). This is
  OpenSSH's `accept-new` rule. If a Console save overwrites the pin at that moment, the
  next health check pins again.
- **`certificate_validation_disabled`**: When true, the Driver accepts any certificate
  and leaves the pin unchanged.

Read tasks retrieve the device each time they start. Write tasks retrieve it when they
start and once when an open connection breaks. A Trust click or an edited endpoint then
takes effect on the next connection without a new task configuration.

The pool key (`driver/opcua/connection/pool.cpp:21`) adds the file paths, the Driver
certificate in use, and the validation switch. It leaves out the pin, so the connection
that trusted the first certificate serves the device after the health check pins it. The
pool reuses an idle connection only if its server presented the pinned certificate, and
leaves other idle connections open for callers that trust them. The circuit breaker
counts a certificate rejection like any failure, so a task that writes fast cannot open
a secure channel per write. While the breaker is open, it repeats the last rejection
instead of its pause message, so the status keeps its cause. A new pin resets the count.

### 3.3 The connect dialog

Text fields replace `FS.InputPath` for the two files. It and `Runtime.pickPath` have no
other users, so both are deleted. Basic256 and Basic128Rsa15 keep their "(deprecated)"
labels and show a caution line: "Use only if the server supports nothing newer."

| Field                       | Shows                                                      | Actions              |
| --------------------------- | ---------------------------------------------------------- | -------------------- |
| Client certificate          | Driver certificate, thumbprint, expiry; or two path fields | Download, Regenerate |
| Server certificate          | "Not trusted", or the pinned thumbprint                    | Trust                |
| Validate server certificate | A switch, on by default, with a troubleshooting caution    |                      |

Trust saves the rejected certificate to the form and connects again. A changed server
certificate fails the same way and offers Trust for the new one. Download saves DER
through `Runtime.useDownload` (`console/src/platform/runtime/useDownload.ts:42`), so it
works in the browser. For a pinned certificate, the Console computes the SHA-1
thumbprint over the DER bytes itself: Web Crypto exists only in a secure context, and a
browser Console can use plain HTTP.

### 3.4 Statuses

Connect replies and the device health check use these. A device shows the first that
applies.

| Condition                       | Variant | Message                                    |
| ------------------------------- | ------- | ------------------------------------------ |
| The Driver rejects the server   | Error   | Server certificate not trusted             |
| The server rejects the Driver   | Error   | Server rejected the Driver certificate     |
| The server rejects the files    | Error   | Server rejected the client certificate     |
| `BadCertificateTimeInvalid`     | Error   | Server certificate expired                 |
| A file cannot be read           | Error   | Failed to read client certificate          |
| The Driver certificate fails    | Error   | Failed to load the Driver certificate      |
| Validation is off               | Warning | Connected without certificate validation   |
| The next certificate is waiting | Warning | Server does not trust the next certificate |

Each description says what to do: compare the thumbprint, approve the Driver on the
server before a date, or renew the server's certificate. The browse reply carries only
`channels`, so `device::Properties::to_json` and `Config::to_json` are deleted.

### 3.5 Migration

Console property schema v2 (`console/src/feature/opcua/device/types/`) adds
`certificateValidationDisabled` and clears a `serverCertificate` that is not PEM. Such a
value is a path from an earlier release, which had no effect. The Driver treats a
non-PEM value as no pin, because a device that no Console opened still holds it. RFC
0081 §4.1 encodes the result in `opcua.Server`, and its Core migration replaces v2. The
Python client's `server_cert` takes PEM contents
(`client/py/synnax/opcua/types.py:163`).

After the upgrade, the health check pins each secured device within seconds, and its
tasks keep running.

### 3.6 Testing

The C++ mock server gets encryption, with certificates that `UA_CreateCertificate` makes
when the tests run. Tests cover the pin on first connect, a changed certificate, a
server that rejects the Driver, an expired certificate, files mode, the validation
switch, every security policy, and the renewal switch over with an injected clock. Pool
tests cover reuse after the pin, an idle connection that a caller does not trust, and
the breaker on repeated rejections. The integration tests `opcua_read.py` and
`opcua_write.py` pass the simulator's certificate as PEM, and one case uses the Driver
certificate.

## 4 Implementation phases

One pull request, after this RFC merges, changes the Driver, the Console, the Python
client, the docs page, and the integration tests together. The pin on first connect and
the trust check must ship in one release, or secured devices stop on upgrade.

| Phase | Scope                                                       | Touches                                                                                                                       | Depends on |
| ----- | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Pin OPC UA server certificates and add a Driver certificate | `driver/opcua`, `driver/rack`, `console/src/feature/opcua`, `client/py/synnax/opcua`, `site/docs`, `integration/tests/driver` | -          |

## 5 Resolved decisions

1. **A Driver identity, not one per device**: One certificate per installation is the
   norm in Ignition, Kepware, UaExpert, Prosys, and OPC Publisher. The trade is real: a
   device that moves to a new rack needs that rack's Driver trusted on the server.
2. **Files on the Driver's computer, not uploads**: An upload sends the private key
   through the Core, and any Viewer can stream the task command channel
   (`core/pkg/api/framer/framer.go:300`). HighByte, Grafana, and Azure IoT Operations
   take uploads. Telegraf, Node-RED, and PLC4X take paths. The trade is real: a browser
   user with a site certificate must copy the files to the Driver's computer.
3. **Trust on the device, not in a Driver trust store**: A pin on the device scopes
   trust to one endpoint. A Driver-wide list would accept any trusted certificate from
   any server, because open62541 v1.4 skips the host name and URI checks. Trust also
   stays after a Driver reinstall. The trade is real: two devices for one server each
   pin.
4. **Pin the server's certificate, not a CA**: Every tool checked pins as its main path:
   the Ignition quarantine, Kepware's Trusted Servers, UaExpert, Prosys, OPC Publisher,
   and Telegraf. CA trust here would first need host name, URI, and revocation checks.
   The trade is real: a renewed server certificate needs a new Trust.
5. **Pin on first connect, not fail closed**: Secured devices keep running after the
   upgrade. A marker for pre-upgrade devices only would need the first Core migration of
   OPC UA properties, so the rule covers every saved device. The trade is real: a device
   made from code trusts its first server without a check by a person.
6. **A validation switch, as a troubleshooting tool**: Ignition has one, with a caution.
   The warning status keeps a device with the switch on from looking healthy. The trade
   is real: a user can leave it on.
7. **Restore the deprecated policies, not remove them**: v0.48 shipped them, and an old
   server that supports only them would otherwise have no security. Ignition, Kepware,
   and UaExpert still offer them. The trade is real: SHA-1 security is available again.
8. **Overlap renewal, not manual renewal or a switch at expiry**: A self-signed
   certificate always needs a new approval on each server. The overlap gives 60 days and
   a list of the servers that still need it. The trade is real: the Driver holds two
   identities, and each server logs one rejected attempt a day until it approves.
9. **DER download only**: Every OPC UA server handles DER, the format on the wire, and
   Kepware and Ignition import it. The docs show the PEM conversion.
10. **Commands, not rack status**: Rack status details hold only `rack`
    (`schemas/synnax/rack.oracle:28`). An OPC UA field there would put integration data
    on the rack.

## 6 What this RFC does not cover

- **Credentials at rest**: The username and password stay in plain text, as in every
  device (RFC 0079 §8, decision 9).
- **Certificate import and private key passwords**: Site certificates come from files on
  the Driver's computer (§3.1).
- **Trust in a CA**: Each device pins its server's certificate (§3.2).
- **GDS certificate management**: OPC UA Part 12 push and pull need a GDS client, which
  open62541 v1.4 does not have.
