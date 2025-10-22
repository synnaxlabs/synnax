# Windows Tooling Setup Guide

# 0 - Summary

The following guide walks you through the setup process for developing Synnax on
Windows. The setup guide for macOS is available [here](setup-macos.md). This guide is
complete, meaning that it provides installation and configuration instructions for all
tooling required, but it does not provide information on how to use this tooling when
working with a specific project. For that information, see the project's `README.md`.
Links to all project `README.md` files can be found in the
[project index](../../README.md).

This guide assumes you're using PowerShell as your terminal of choice. If you're using
cmd, you may need to vary the commands slightly. Certain tools may require running
commands as an administrator.

As a final note, this guide does not need to be followed verbatim. As long as the
correct tools are installed and configured, you can use whatever methods you prefer.

# 1 - Install Git

The first step is to install Git. The best way of doing so is using the
[installer](https://git-scm.com/download/win). Validate your installation by running

```bash
git --version
```

You should see something like

```bash
git version 2.x.x.windows.x
```

# 2 - Clone the Repository

The next step is to clone the Git repository. We recommend cloning it into
`~/Desktop/synnaxlabs` as it makes it easier to follow the commands in other guides.

```
mkdir ~/Desktop/synnaxlabs && cd ~/Desktop/synnaxlabs && git clone https://github.com/synnaxlabs/synnax
```

# 3 - Setup Go

To install Go, use the instructions from the [Go website](https://go.dev/doc/install).
To verify your installation, run:

```bash
go version
```

You should see something like

```bash
go 1.20.x windows/amd64
```

As additional verification, let's run some test cases to make sure everything is working
as expected. In the root directory (`~/synnaxlabs/Desktop/synnax`), run

```bash
cd x/go && go test -v ./...
```

This will run the tests for the common utilities used across Synnax's go projects. This
might take a while when you run it for the first time, as Go needs to download many
packages. Future runs will be much faster. Eventually, you **should see a bunch of green
output and no red output.**

# 4 - Python

Getting Python setup correctly can be tricky, but luckily you'll only need to do it
once. To get started, use the installer available
[here](https://www.python.org/downloads/release/python-3114/). Make sure to check the
box that says "Add Python 3.11 to PATH". After installing, run

```bash
python --version
```

You should see something like

```bash
Python 3.11.x
```

## 4.0 - Install Poetry

Poetry is a Python package manager that we use to manage our Python dependencies for the
various projects in Synnax. To install poetry, run

```bash
(Invoke-WebRequest -Uri https://install.python-poetry.org -UseBasicParsing).Content | py -
```

Then, run

```bash
poetry --version
```

You should see something like

```bash
Poetry version 1.x.x
```

If you run into trouble, check out the [poetry docs](https://python-poetry.org/docs/)
for more information.

## 4.1 - Install Dependencies

Synnax has three Python projects: `freighter/py`, `client/py`, and `alamos/py`. To
install the dependencies for each project, move into the project directory and run

```bash
poetry install
```

# 5 - Front End Build System

# 5.0 - Install Node.js

I recommend using nvm to manage Node.js versions. Install nvm using the instructions
[here](https://github.com/coreybutler/nvm-windows/releases). You want to install and run
the file titled `nvm-setup.exe`. Then, install the latest version of node with

```bash
nvm install 20
```

If your command line prompts you with instructions to use the version, execute it.

```bash
If you want to use this version, type
nvm use 20.x.x
```

Make sure your installation is working by running

```bash
node --version
```

# 5.1 - Install pnpm

We use pnpm as our package manager of choice. It's a drop-in replacement for npm that
has a few nice features. To install pnpm, run

```bash
corepack enable
```

Then, prepare npm by running

```bash
corepack prepare pnpm@latest --activate
```

# 5.1 - Install Dependencies

In the root directory of the repository, run

```bash
pnpm install
```

# 5.2 - Build the Pluto Component Library

We use [Turborepo](https://turbo.build/repo) to build our various typescript projects.
It has great monorepo support, and intelligently caches builds to speed up development.
As a test to make sure the build system is working, we'll build the Synnax component
library, [Pluto](../../pluto/README.md) by running

```bash
pnpm build:pluto
```

# 5.3 - Start a Pluto Dev Server

As another test, we'll start a development server for Pluto. We use this server to
develop components in isolation before integrating them into the main Synnax
application, [Console](../console/README.md). To start, run

```bash
pnpm dev:pluto
```

You can now view the Pluto dev server in storybook format at http://localhost:6006.
