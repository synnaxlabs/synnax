# Debugging the Driver

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

   | From CI Artifact | Rename To |
   |------------------|-----------|
   | `synnax-driver-v{version}-windows.exe` | `driver.exe` |
   | `driver.pdb` | `driver.pdb` (no change) |

3. Both files **must** be named `driver.exe` and `driver.pdb` in the **same folder**
4. Open the folder in Visual Studio: **File > Open > Folder**
5. Select `driver.exe` from the debug dropdown
6. Press **F5**

## Troubleshooting

**Symbols show "Unknown"**: The PDB must match the exact build of the exe. Re-download both from the same CI run. You may also need to manually load symbols:
- **Debug > Windows > Modules**
- Right-click `driver.exe`
- Click **Load Symbols**
- Browse to the PDB file

**"Symbol loading disabled"**: Go to **Tools > Options > Debugging > Symbols** and select **Load all modules, unless excluded**.
