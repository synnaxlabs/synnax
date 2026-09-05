# Windows tooling setup guide

## 0 Summary

This guide walks you through the setup process for developing Synnax on Windows. The
setup guide for macOS is available [here](setup-macos.md). This guide is complete,
meaning that it gives installation and configuration instructions for all the tooling
you need, but it does not tell you how to use that tooling on a specific project. For
that information, see the project's `README.md`. Links to all project `README.md` files
are in the [project index](../../README.md).

Sections 1 to 5 set up the tools that every contributor needs. Sections 6 and 7 apply
only if you work on the Console, and sections 6 and 8 only if you work on the Driver.

This guide assumes [PowerShell 7](https://github.com/PowerShell/PowerShell) as your
terminal, and it installs tools with
[winget](https://learn.microsoft.com/windows/package-manager/), which ships with
Windows. Some tools ask you to run commands as an administrator.

This guide also does not need to be followed verbatim. As long as the correct tools are
installed and configured, use whatever methods you prefer.

## 1 Install Git

Install Git with

```powershell
winget install Git.Git
```

Open a new shell, then verify the installation by running

```powershell
git --version
```

The output should look something like

```text
git version 2.55.0.windows.1
```

## 2 Clone the repository

We recommend cloning into `~\Desktop\synnaxlabs`, as it makes the commands in other
guides easier to follow. The Driver keeps its C++ dependencies in submodules, so clone
them at the same time.

```powershell
mkdir ~\Desktop\synnaxlabs
cd ~\Desktop\synnaxlabs
git clone --recurse-submodules https://github.com/synnaxlabs/synnax
```

If you already cloned the repository without submodules, run
`git submodule update --init --recursive` in the repository root.

## 3 Set up Go

We use [Go](https://go.dev/) for the Core, Aspen, Cesium, Arc, Freighter, and Oracle.
Install it with the [installer](https://go.dev/doc/install), or with

```powershell
winget install GoLang.Go
```

Verify the installation by running

```powershell
go version
```

The output should look something like

```text
go version go1.26.5 windows/amd64
```

Each Go module pins its toolchain in `go.mod`. The Core currently does not compile with
Go 1.27, because a Pebble dependency uses runtime internals that 1.27 removed. If your
local Go is newer than the pinned version, run `$env:GOTOOLCHAIN = "go1.26.5"` before
you run commands in `core\`, and Go downloads the correct toolchain for you.

As an additional check, run some test cases. In the repository root
(`~\Desktop\synnaxlabs\synnax`), run

```powershell
cd x/go
go test -v ./...
```

This runs the tests for the common utilities that all Synnax Go projects use. The first
run can take a while, because Go downloads many packages. Later runs are much faster.
You **should see a lot of green output and no red output.**

## 4 Set up Python

### 4.0 Install uv

We use [uv](https://docs.astral.sh/uv/) to manage Python versions and dependencies.

```powershell
winget install astral-sh.uv
```

Open a new shell, then verify the installation by running

```powershell
uv --version
```

The output should look something like

```text
uv 0.12.0
```

If you have trouble, see the [uv docs](https://docs.astral.sh/uv/).

### 4.1 Install Python

uv installs and manages the interpreter, so you do not need a separate Python installer.
The workspace needs Python 3.12.

```powershell
uv python install 3.12
```

### 4.2 Install dependencies

Synnax uses a uv workspace with five Python projects: `alamos/py`, `client/py`,
`freighter/py`, `integration`, and `x/py`. To install the dependencies for all five, run
this from the repository root:

```powershell
uv sync
```

Run Python commands with `uv run`, which selects the workspace interpreter and
environment for you.

## 5 Set up TypeScript

### 5.0 Install pnpm and Node.js

We use [pnpm](https://pnpm.io/) as our package manager. pnpm installs itself and then
installs Node.js for you, so you do not need nvm or Corepack.

```powershell
$env:PNPM_VERSION="12"
Invoke-WebRequest https://get.pnpm.io/install.ps1 -UseBasicParsing | Invoke-Expression
```

Open a new shell, then install Node.js with

```powershell
pnpm runtime set node 24 -g
```

Verify both installations by running

```powershell
pnpm --version
node --version
```

The output should look something like

```text
12.1.0
v24.20.0
```

pnpm reads the versions this repository needs from the `devEngines` field in the root
`package.json`, and downloads them when they are missing. To change the pnpm or Node.js
version for everyone, edit that field.

### 5.1 Install dependencies

In the repository root, run

```powershell
pnpm install
```

### 5.2 Build the Pluto component library

We use [Turborepo](https://turbo.build/repo) to build our TypeScript projects. It has
good monorepo support, and caches builds to speed up development. As a test that the
build system works, build the Synnax component library, [Pluto](../../pluto):

```powershell
pnpm build:pluto
```

### 5.3 Start a Pluto dev server

As another test, start a development server for Pluto. We use this server to develop
components in isolation before we integrate them into the [Console](../../console).

```powershell
pnpm dev:pluto
```

Vite serves the component sandbox at [localhost:5173](http://localhost:5173).

## 6 Install the Visual C++ build tools

Rust and the Driver build native code with the Microsoft C++ toolchain. Install it with

```powershell
winget install Microsoft.VisualStudio.2022.BuildTools --override "--wait --passive --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"
```

## 7 Set up Rust

The [Console](../../console) uses [Tauri](https://tauri.app/), which builds its back end
with [Rust](https://www.rust-lang.org/). Install Rust with rustup, the toolchain manager
that Tauri needs.

```powershell
winget install Rustlang.Rustup
```

Open a new shell, then verify the installation by running

```powershell
rustc --version
```

The output should look something like

```text
rustc 1.98.0 (88d9e12ae 2026-08-18)
```

Tauri also needs WebView2, which Windows 11 and current Windows 10 installations
include. If the Console fails to start with a WebView2 error, install the
[Evergreen Runtime](https://developer.microsoft.com/microsoft-edge/webview2/).

## 8 Set up C++

You only need this section if you work on the [Driver](../../driver).

### 8.0 Install Bazel

We build the Driver with [Bazel](https://bazel.build/). Install Bazelisk, which reads
the Bazel version from `.bazeliskrc` and downloads it for you.

```powershell
winget install Bazel.Bazelisk
```

Verify the installation from the repository root by running

```powershell
bazel --version
```

### 8.1 Install clang-format

We format all C++ with clang-format, which the LLVM toolchain provides.

```powershell
winget install LLVM.LLVM
```

CI pins clang-format 22, and other versions format differently. If your diffs show
formatting changes you did not make, install LLVM 22 from the
[LLVM releases](https://github.com/llvm/llvm-project/releases) page.

### 8.2 Build the Driver

Make sure the submodules from section 2 and the build tools from section 6 are present,
then run

```powershell
bazel build //driver
```

The first build takes a long time, because Bazel compiles all the vendored dependencies.
Later builds use the cache.
