# macOS tooling setup guide

## 0 Summary

This guide walks you through the setup process for developing Synnax on macOS. The setup
guide for Windows is available [here](setup-windows.md). This guide is complete, meaning
that it gives installation and configuration instructions for all the tooling you need,
but it does not tell you how to use that tooling on a specific project. For that
information, see the project's `README.md`. Links to all project `README.md` files are
in the [project index](../../README.md).

Sections 1 to 7 set up the tools that every contributor needs. Section 8 applies only if
you work on the Console, and section 9 only if you work on the Driver.

Some tools ask you to run commands with `sudo` privileges. This guide also does not need
to be followed verbatim. As long as the correct tools are installed and configured, use
whatever methods you prefer.

## 1 Install Homebrew

We recommend [Homebrew](https://brew.sh/) to install and manage tooling for Synnax
development.

```zsh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

## 2 Install the Xcode command line tools

Go, Rust, and the Driver all build native code, which needs Apple's compiler toolchain.

```zsh
xcode-select --install
```

## 3 Install Git

We use Git for all our version control.

```zsh
brew install git
```

## 4 Clone the repository

We recommend cloning into `~/Desktop/synnaxlabs`, as it makes the commands in other
guides easier to follow. The Driver keeps its C++ dependencies in submodules, so clone
them at the same time.

```zsh
mkdir ~/Desktop/synnaxlabs && cd ~/Desktop/synnaxlabs && git clone --recurse-submodules https://github.com/synnaxlabs/synnax
```

If you already cloned the repository without submodules, run
`git submodule update --init --recursive` in the repository root.

## 5 Set up Go

We use [Go](https://go.dev/) for the Core, Aspen, Cesium, Arc, Freighter, and Oracle.

```zsh
brew install go
```

Verify the installation by running

```zsh
go version
```

The output should look something like

```text
go version go1.26.5 darwin/arm64
```

Each Go module pins its toolchain in `go.mod`. The Core currently does not compile with
Go 1.27, because a Pebble dependency uses runtime internals that 1.27 removed. If your
local Go is newer than the pinned version, prefix commands in `core/` with
`GOTOOLCHAIN=go1.26.5`, and Go downloads the correct toolchain for you.

As an additional check, run some test cases. In the repository root, run

```zsh
cd x/go && go test -v ./...
```

This runs the tests for the common utilities that all Synnax Go projects use. The first
run can take a while, because Go downloads many packages. Later runs are much faster.
You **should see a lot of green output and no red output.**

## 6 Set up Python

### 6.0 Install uv

We use [uv](https://docs.astral.sh/uv/) to manage Python versions and dependencies.

```zsh
brew install uv
```

Verify the installation by running

```zsh
uv --version
```

The output should look something like

```text
uv 0.12.0
```

### 6.1 Install Python

uv installs and manages the interpreter, so you do not need Homebrew Python or a shell
alias. The workspace needs Python 3.12.

```zsh
uv python install 3.12
```

### 6.2 Install dependencies

Synnax uses a uv workspace with five Python projects: `alamos/py`, `client/py`,
`freighter/py`, `integration`, and `x/py`. To install the dependencies for all five, run
this from the repository root:

```zsh
uv sync
```

Run Python commands with `uv run`, which selects the workspace interpreter and
environment for you.

## 7 Set up TypeScript

### 7.0 Install pnpm and Node.js

We use [pnpm](https://pnpm.io/) as our package manager. pnpm installs itself and then
installs Node.js for you, so you do not need nvm or Corepack.

```zsh
curl -fsSL https://get.pnpm.io/install.sh | env PNPM_VERSION=12 sh -
```

Open a new shell, then install Node.js with

```zsh
pnpm runtime set node 24 -g
```

Verify both installations by running

```zsh
pnpm --version && node --version
```

The output should look something like

```text
12.1.0
v24.20.0
```

pnpm reads the versions this repository needs from the `devEngines` field in the root
`package.json`, and downloads them when they are missing. To change the pnpm or Node.js
version for everyone, edit that field.

### 7.1 Install dependencies

In the repository root, run

```zsh
pnpm install
```

### 7.2 Build the Pluto component library

We use [Turborepo](https://turbo.build/repo) to build our TypeScript projects. It has
good monorepo support, and caches builds to speed up development. As a test that the
build system works, build the Synnax component library, [Pluto](../../pluto):

```zsh
pnpm build:pluto
```

### 7.3 Start a Pluto dev server

As another test, start a development server for Pluto. We use this server to develop
components in isolation before we integrate them into the [Console](../../console).

```zsh
pnpm dev:pluto
```

Vite serves the component sandbox at [localhost:5173](http://localhost:5173).

## 8 Set up Rust

The [Console](../../console) uses [Tauri](https://tauri.app/), which builds its back end
with [Rust](https://www.rust-lang.org/). Install Rust with rustup, the toolchain manager
that Tauri needs. Do not install the Homebrew `rust` formula, which does not manage
toolchains.

```zsh
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

Open a new shell, then verify the installation by running

```zsh
rustc --version
```

The output should look something like

```text
rustc 1.98.0 (88d9e12ae 2026-08-18)
```

## 9 Set up C++

You only need this section if you work on the [Driver](../../driver).

### 9.0 Install Bazel

We build the Driver with [Bazel](https://bazel.build/). Install Bazelisk, which reads
the Bazel version from `.bazeliskrc` and downloads it for you.

```zsh
brew install bazelisk
```

Verify the installation from the repository root by running

```zsh
bazel --version
```

### 9.1 Install clang-format

We format all C++ with clang-format. CI pins version 22, and other versions format
differently, so install that release. Homebrew keeps it keg-only, which means you must
add it to your `PATH` yourself.

```zsh
brew install llvm@22
echo 'export PATH="$(brew --prefix llvm@22)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Format the Driver with

```zsh
bash scripts/clang_format.sh driver
```

### 9.2 Build the Driver

Make sure the submodules from section 4 are present, then run

```zsh
bazel build //driver
```

The first build takes a long time, because Bazel compiles all the vendored dependencies.
Later builds use the cache.
