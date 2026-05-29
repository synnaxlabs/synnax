# Debugging the Driver

## Crash Stack Traces

The driver installs a crash handler at startup (`x::crash::install`) that writes a stack
trace to stderr before the process dies. It covers fatal signals (SIGSEGV, SIGABRT,
SIGILL, SIGFPE, SIGBUS on POSIX; unhandled structured exceptions on Windows) and
unhandled C++ exceptions, printing the exception's `what()` message when available.
Because stderr is where glog writes, the trace lands in the same place as the rest of
the driver logs (including the service log file).

The handler does not touch SIGINT or SIGTERM, which remain graceful-shutdown signals.
Stack-overflow crashes are also covered: the handler runs on an alternate signal stack
(POSIX) / with a reserved stack guarantee (Windows).

### Resolving addresses

Optimized release builds are stripped, so a field trace shows raw addresses with few or
no symbol names. To turn those into file and line numbers, resolve them offline against
a build that retains symbols.

On Linux/macOS, build an unstripped binary and use `addr2line`:

```bash
bazel build -c dbg //driver
addr2line -fCe bazel-bin/driver/driver 0x<address>
```

On Windows, load the matching `driver.pdb` (see below) into the debugger, or use the
`#: name [0x<address>]` lines the handler already symbolizes when a `.pdb` is present
alongside the executable.

The `driver` binary is linked with `-rdynamic` on POSIX, so non-stripped builds resolve
exported function names directly in the trace.

## Building with Debug Symbols

### Via GitHub Actions

1. Go to **Actions** > **Build - Synnax**
2. Click **Run workflow** and enable:
   - `BDEBUG: Build Driver with Debug Symbols`
   - `Platform: Windows` (or your platform)
   - `Build: Driver`
   - `Use simple artifact names` (recommended)
3. Run the setup script (downloads artifacts and opens Visual Studio):

   ```powershell
   .\driver\scripts\setup-debug.ps1 <run-id>
   ```

   The run ID is in the URL: `github.com/synnaxlabs/synnax/actions/runs/<run-id>`

4. Press **F5** to debug

**Script options:**

- `-Args "start -s"` - Set command line arguments (default: `start -s`)
- `-OutputDir ./mydir` - Custom output directory (default: `./debugdriver`)
- `-NoLaunch` - Skip opening Visual Studio

### Local Build

```bash
bazel build -c dbg --output_groups=+pdb_file //driver
```

Output:

- `bazel-bin/driver/driver.exe`
- `bazel-bin/driver/driver.pdb`

## Manual Setup (Windows)

If not using the script, set up manually:

1. Create a folder (e.g., `debugdriver/`)
2. Download and rename files:

   | From CI Artifact                       | Rename To                |
   | -------------------------------------- | ------------------------ |
   | `synnax-driver-v{version}-windows.exe` | `driver.exe`             |
   | `driver.pdb`                           | `driver.pdb` (no change) |

3. Both files **must** be named `driver.exe` and `driver.pdb` in the **same folder**
4. Open the folder in Visual Studio: **File > Open > Folder**
5. Select `driver.exe` from the debug dropdown
6. Press **F5**

## Troubleshooting

**Symbols show "Unknown"**: The PDB must match the exact build of the exe. Re-download
both from the same CI run. You may also need to manually load symbols:

- **Debug > Windows > Modules**
- Right-click `driver.exe`
- Click **Load Symbols**
- Browse to the PDB file

**"Symbol loading disabled"**: Go to **Tools > Options > Debugging > Symbols** and
select **Load all modules, unless excluded**.
