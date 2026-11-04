# Security review

You are on a fresh clone of `main` of the Synnax monorepo. Find the security
vulnerabilities in your area. Nobody watches this session, so do not ask questions.

## Keep the findings private

The repository is public. A vulnerability that is not fixed must not become public.

- Never push, commit, open a PR or an issue, comment, or post anywhere. Your last
  message is your only output. It goes to a private report.
- Never write a secret value. Give its location only.
- Never send code or findings to a network service.

## Method

1. Read `CLAUDE.md`, `docs/claude/architecture.md`, and the guides for your area.
2. Find each entry point in your area where untrusted data comes in: a network request,
   a file, an IPC message, a deep link, a device or server response, an event payload.
3. Follow the data to each sink: an access decision, a query, a file path, a shell
   command, a memory copy, a hardware write, a cloud API call. Follow it out of your
   area when it goes there.
4. Before you report a finding, try to disprove it. Look for a guard upstream and read
   the tests. If you cannot show how an attacker reaches the fault, leave it out.

Skip a path that does not exist. Skip generated and vendored code.

## What counts

A finding is a fault that an attacker can reach. It needs evidence:

- **Location**: `path:line` of the fault.
- **Reach**: the path from the entry point to the fault, as `path:line` steps, and what
  the attacker must have: network access, an account, a role, a malicious device or
  server, a pull request from a fork.
- **Impact**: what the attacker gets.
- **Severity**: one level from the table below.
- **Fix**: one sentence.

| Severity | Meaning                                                                                                                          |
| -------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Critical | An attacker with no account runs code, reads or changes any data, or commands hardware.                                          |
| High     | A user goes past their role, an attacker with no account crashes the Core or the Driver, or a fork pull request gets a secret.   |
| Medium   | The attack needs a precondition: a user action, a malicious device or server, or a position on the local network. Or data leaks. |
| Low      | The impact is small or the preconditions are hard.                                                                               |

These do not count: style, missing hardening with no path to a fault, a fault that needs
admin access to the Core or the host, and vulnerable dependency versions (a scanner
reports those). No speculation: "could be unsafe" is not a finding.

## Areas

### `authn`: authentication and authorization

Paths: `core/pkg/service/auth`, `core/pkg/service/access`, `core/pkg/service/user`,
`core/pkg/api/auth`, `core/pkg/api/access`, `core/pkg/api/user`,
`core/pkg/transport/grpc/auth`, `client/ts/src/auth`, `client/py/synnax/auth`,
`client/cpp/auth`, and `docs/tech/rfc/0073-authentication.md` for the design.

Look at password storage and comparison, token signing, validation, and expiry, sessions
and revocation, the access check on each API route, policy evaluation, and default
credentials.

### `api`: input validation on the API and transports

Paths: `core/pkg/api`, `core/pkg/transport`, `core/pkg/server`, `freighter/go`,
`freighter/ts`, `freighter/py`, `freighter/cpp`.

Look at request decoding (size limits, depth), the validation in each handler, queries
built from input, WebSocket upgrades and origins, CORS, errors that leak internals,
panics a request can cause, and allocations sized by the client.

### `tls`: TLS and certificates

Paths: `core/pkg/security`, `core/cmd/cert`, `core/pkg/server`, `core/cmd/listener`, and
the TLS options in `client/ts`, `client/py`, `client/cpp`, `freighter`, and `driver`.

Look at certificate generation, verification that is off or falls back to insecure,
minimum versions and ciphers, hostname checks, and private key file permissions.

### `files`: file and path handling

Paths: `core/pkg/service/imex`, `core/pkg/api` handlers that take files or paths,
`core/cmd` (data and config directories), `cesium`, and the Console import and export
code in `console/src`.

Look at path traversal, archive extraction, permissions on created files, symlinks, and
temporary files.

### `driver`: Driver hardware commands and protocols

Paths: `driver/opcua`, `driver/modbus`, `driver/ethercat`, `driver/http`,
`driver/labjack`, `driver/ni`, `driver/task`, `driver/rack`, `driver/control`,
`driver/arc`.

Look at parsing of bytes from devices and servers (bounds, integer overflow), task
configuration from the Core, commands that reach hardware without a control authority
check, request forgery and injection in HTTP tasks, and OPC UA security modes and
certificate trust.

### `console`: Console IPC and deep links

Paths: `console/src-tauri`, `console/src/platform`, `console/src/session/link`, and the
Console code that renders user content or opens URLs.

Look at Tauri capabilities and plugin scopes, the IPC commands in `src-tauri/src`, deep
link handling, the CSP, updater signature checks, user text rendered as HTML, and URLs
opened outside the app.

### `ci`: workflows, agents, and cloud infrastructure

Paths: `.github/workflows`, `.github/actions`, `.github/scripts`, `scripts/agent`,
`infra/agent-runner`, `deploy/preview`.

Look at `pull_request_target` and `workflow_run` jobs that check out untrusted code,
untrusted input (titles, branch names, bodies, labels, dispatch payloads) in `run:`
blocks, secrets and self-hosted runners that a fork pull request reaches, permissions
wider than needed, IAM policies and OIDC trust conditions wider than needed, open
security groups, preview environments with default credentials, and agent prompts that
take text from pull requests or issues.

## Report

Your last message is the report, in Markdown. For each finding, highest severity first:

```markdown
### <Severity>: <short title>

- Location: `path:line`
- Reach: <entry point> (`path:line`) → … → <fault> (`path:line`). Needs: <access>.
- Impact: <what the attacker gets>
- Fix: <one sentence>
```

With no findings, write only `No findings.` Then, in both cases, end with one line that
names the parts of the area you did not read.
