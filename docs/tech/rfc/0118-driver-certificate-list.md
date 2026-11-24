# 118 The Driver certificate in the certificate list

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-22
- **Related**: [RFC 0089 - OPC UA certificates](0089-opc-ua-certificates.md),
  [RFC 0101 - Secrets and certificates](0101-secrets-and-certificates.md)

## 0 Summary

The Driver certificate is the only certificate that the Core does not list. Each Driver
now saves a file certificate in the Core for each certificate in its folder, so the
Driver certificate shows in Settings with its expiry status. The private key stays in
the Driver's folder, and no record names it.

## 1 Motivation

The Settings list shows every stored and file certificate
(`console/src/feature/secret/Settings.tsx:175`), and the Core reports the expiry of its
own certificates (RFC 0101 §3.8). The Driver certificate has only its own paths: a rack
menu modal and the `get_certificate` and `regenerate_certificate` scan task commands
(RFC 0089 §3.0). An admin who checks which certificates expire must open each rack. When
renewal stalls because a server does not trust the next certificate, only the status of
each device says so. RFC 0101 §6 leaves the Driver certificate out with no decision.

## 2 Principles

1. **The private key stays where it is made**: The Driver makes its key, and no other
   computer needs it. OPC UA Part 4 §6.1.2 says the key should be where only the
   application can reach it.
2. **The shapes that exist describe it**: The Driver certificate is a certificate in
   files on the computer that uses it, which is what a file certificate is.

## 3 Design

### 3.0 The record

For each certificate in its folder, the OPC UA scan task saves a file certificate on its
rack:

| Field              | Value                                                                          |
| ------------------ | ------------------------------------------------------------------------------ |
| `name`             | "<rack> OPC UA Driver certificate", or "<rack> next OPC UA Driver certificate" |
| `certificate_path` | The certificate file in `<state_dir>/opcua/pki/own/certs`                      |
| `private_key_file` | Unset                                                                          |
| `details`          | Read by the Driver from the file                                               |
| `rack`             | The rack of the Driver                                                         |

The Core writes its expiry status as for any certificate with details
(`core/pkg/service/certificate/expiry.go`), and deletes it with the rack
(`schemas/synnax/certificate.oracle`, `@ref on_delete cascade`). An empty
`client_certificate` still selects the Driver certificate (RFC 0089 §3.1), so no device
needs the record. A device that pins a record as its server certificate blocks its
delete (`core/pkg/service/certificate/reference.go:50`). The scan task then reports the
error until a user changes the device.

### 3.1 The private key

The key stays in `<state_dir>/opcua/pki/own/private`, outside the secrets folder. No
secret or certificate names it, so no user can make the Driver read it for another use.

### 3.2 Keeping the records current

The Driver matches records to its certificates by `certificate_path`, which it makes
itself, so the paths need no normal form. A record on its rack whose file has the name
that the Driver gives its certificates, "Synnax Driver [<thumbprint>].der", is a Driver
record, in any folder. On each scan, under the scan task lock, the Driver compares its
certificates and its rack name with those of its last sync that succeeded. On a change,
it retrieves the file certificates of its rack and keeps one record for each
certificate. It renames a record when its certificate becomes the current one, and
deletes every other Driver record, including a second record of one file. A sync that
fails runs again at the next scan.

So a switch that `Store::current()` makes when a read or write task connects shows at
the next scan. A record that a user deletes comes back at the next change or start. A
Driver whose state directory moves deletes the records of the old folder.

### 3.3 Permissions

A rack may update and delete only the file certificates on it. The write binder lets a
user change a draft panel that uses that user with no other access (`Owned` in
`core/pkg/api/bind/service.go`). A file certificate uses its rack in the same way, so
the certificate table joins the owned tables, and the Driver role does not change. A
rack still gets an access error when it updates or deletes a stored certificate or a
file certificate on another rack, or moves a file certificate to another rack. The
Driver role still grants create on every certificate
(`core/pkg/service/access/rbac/builtin/builtin.go:183`). The gRPC transport gains
`certificate/delete`, which is a no-op today (`core/pkg/transport/grpc/grpc.go:267`),
and the C++ client gains delete and a retrieve that filters by rack.

### 3.4 The client certificate field

A certificate without a key cannot be a client certificate. The client certificate field
of the connect dialog lists only certificates with a key, and its create dialog requires
a key. This also hides pinned server certificates, which the field lists today
(`console/src/platform/certificate/Field.tsx:43`).

## 4 Implementation phases

| Phase | Scope                                                     | Touches                                                                                  | Depends on |
| ----- | --------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------- |
| 1     | Let a rack delete its own file certificates               | `core/pkg/api/bind`, `core/pkg/api/certificate`, `core/pkg/transport/grpc`, `client/cpp` | -          |
| 2     | Offer only certificates with a key as client certificates | `console/src/platform/certificate`, `console/src/feature/opcua`                          | -          |
| 3     | List the Driver certificate in the Core                   | `driver/opcua`, `site/docs`                                                              | 1, 2       |

## 5 Resolved decisions

1. **The key on the Driver, not in the Core**: A stored key reaches anyone who copies
   the Core's data folder with its secrets key, and any rack whose devices use the
   certificate. A key that someone takes still works after their access ends. The OPC UA
   GDS prefers keys that the application makes, and Vault PKI does not store the keys it
   issues. Ignition, Kepware, Litmus Edge, and AWS SiteWise keep the client key on the
   computer that connects. Only Azure IoT Operations holds it centrally, for keys that a
   user uploads. The trade is real: a Driver that loses its folder needs a new trust on
   each server.
2. **A file certificate, not a new source**: A source says where a certificate lives,
   and the Driver certificate lives in files on its rack. HighByte also lists its OPC UA
   client certificate with its other certificates. The trade is real: the Core holds no
   PEM, so Settings cannot download it, as for every file certificate. The connect
   dialog still downloads it from the Driver.
3. **The key outside the secrets folder**: A stored certificate whose key is a file
   secret would also keep the key on the Driver. But a file secret can name any file in
   the secrets folder, so a user could make the key the value of an HTTP token and send
   it to a server that the user picks. The trade is real: the Driver keeps its key in a
   second folder.
4. **Match by file name, not by a stored mark**: The Driver names its own certificate
   files, so no field or fixed key marks its records. The trade is real: a file
   certificate that a user points at a Driver certificate file looks like a Driver
   record, and the Driver renames or deletes it.

## 6 What this RFC does not cover

- Download and Regenerate in Settings. The connect dialog keeps both through the scan
  task commands (RFC 0089 §3.0).
- The renewal flow. The Driver still tries the next certificate on each server and
  switches when every server accepts it (RFC 0089 decision 8).
