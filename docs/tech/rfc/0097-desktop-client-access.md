# 97 Client access to Synnax Desktop

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-02
- **Related**: [RFC 0063 - Synnax Desktop](0063-synnax-desktop.md),
  [RFC 0073 - Authentication](0073-authentication.md)

## 0 Summary

Scripts on the same computer connect to the Core of Synnax Desktop. Desktop saves the
address and an API key of the root user in `~/.synnax/desktop.json`, a file that only
the user can read. With no arguments, `sy.Synnax()` reads the file. A "Connect from
code" dialog gives the install command and code for each client, and a button that
replaces the key.

## 1 Motivation

- **Desktop users cannot automate**: RFC 0063 §7 left client access out. A bench
  engineer who records data in Desktop cannot read it into pandas or script a test
  sequence. The docs send this user to Enterprise.
- **Nothing in Desktop is stable for a script**: The port changes on each launch (RFC
  0063 §8 decision 5), and the root password changes on each launch and is never saved
  (RFC 0063 §4 Principle 5).

## 2 Principles

1. **Zero configuration for Python**: `sy.Synnax()` connects to Desktop with no
   arguments. A saved `sy login` comes first, so a person who logged in to another Core
   keeps that behavior.
2. **The file is the only secret on disk**: The file holds an API key, never the launch
   password. The same user can already read the Core database, so the file adds no
   access that the user does not have.
3. **Desktop makes the key through the public API**: The Console creates the key after
   sign-in, as `sy login` does. The Core gets no change.
4. **Always on**: There is no setting. "Reset key" is the only control.

## 3 Design

### 3.0 The discovery file

The shell writes `~/.synnax/desktop.json` on every operating system: `host`, `port`,
`secure`, and `api_key`, the field names of the Python `Options`. The write goes to a
partial file with mode `0600`, then renames it. "Erase all data" deletes the file.

### 3.1 A port that stays the same

The supervisor reads the port from the `core.json` of the last launch and binds it again
when it is free (`console/src-tauri/src/supervisor/launch.rs`). When another program
holds it, the supervisor picks a free port, as before. Python reads the new port from
the file; copied TypeScript code needs it again.

### 3.2 The key

After sign-in, the main window reads the saved key. When it is missing, malformed, or
not in `client.auth.retrieveKeys()`, the window creates a key named "Scripts on this
computer" and saves it. Otherwise it saves the same key again, with the current port.
Root's API keys survive the password reset of each launch (RFC 0073 §4.7). When a save
fails, the window deletes the new key. The shell saves a key only while the file holds
the key the window read, so a window that loses a race deletes its new key and uses the
saved one.

### 3.3 The dialog

A code button in the top bar, beside the docs button, and the command "Connect from
code" open the dialog. It has three tabs:

- **Python**: `pip install "synnax~=X.Y.0"`, pinned to the Core version, and
  `client = sy.Synnax()`.
- **TypeScript**: `npm install @synnaxlabs/client@~X.Y.0`, and `new Synnax(...)` with
  the host, the port, and the key filled in. The key shows masked; the copy holds it.
- **Other**: The host, the port, and the key, for the C++ client and LabVIEW.

The footer shows the number of sessions of the key, a link to the client docs, and
"Reset key". A reset asks first, deletes the old key, which ends its sessions, then
saves a new key. A reset that fails never leaves the old key working.

### 3.4 Python client errors

When the options are the ones in the file, from the file or from the caller, and the
Core does not answer, the client raises "Synnax Desktop is not running". A version
mismatch names Desktop and gives the `pip` command that matches the Core.

## 4 Implementation phases

One pull request changes the shell, the Console, the TypeScript and Python clients, and
the docs. The dialog has no use without the file, and the file has no use without the
Python loader.

| Phase | Scope                                        | Touches                                                                                    | Depends on |
| ----- | -------------------------------------------- | ------------------------------------------------------------------------------------------ | ---------- |
| 1     | Let scripts on the same computer use Desktop | `console/src-tauri`, `console/src/feature/embedded`, `client/py`, `client/ts`, `site/docs` | -          |

## 5 Resolved decisions

1. **The Core makes the key**: Rejected. "Reset key" must work with no restart of the
   Core, and a Core flag would add a Desktop path to the Core.
2. **The launch password in the file**: Rejected. It would last only one launch, and it
   logs in as root with no way to revoke it apart from the others.
3. **A fixed port**: Rejected again for RFC 0063 §8 decision 5. The last port is a
   preference, not a promise.
4. **A Node loader for TypeScript**: Rejected. Most TypeScript runs in a browser, which
   cannot read the file. The dialog gives code to copy. The trade is real: copied code
   breaks when the port changes or the key resets.
5. **A setting to turn access off**: Rejected. The file gives no access that the same
   user lacks. "Reset key" ends every session.

This RFC amends RFC 0063 §4 Principle 2 for the dialog, which names a host, a port, and
a credential. It replaces RFC 0063 §4 Principle 5 with Principle 2, removes the client
line of RFC 0063 §7, and changes the Synnax Desktop line of RFC 0073 §4.11 to "API key
of root, in an owner-only file".

## 6 What this RFC does not cover

- Access from other computers. A Desktop license refuses any listener that is not
  loopback.
- Access from a browser on the same computer.
- A Core that outlives the app. Scripts stop when Desktop closes.

## 7 Open questions

None.
