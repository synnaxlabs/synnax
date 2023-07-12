# 0 - Before Getting Started

This guide assumes you're using powershell as your terminal of choice. If you're using
cmd, you may need to vary the commands slightly.

# 1 - Install Git

The first step is to install git. The way to do so is using
the [installer](https://git-scm.com/download/win).

# 2 - Clone the Repository

The next step is to clone the git repository. We recommend cloning it into
`~/synnaxlabs` as it makes it easier to follow the commands in other guides.

```
mkdir ~/synnaxlabs && cd ~/synnaxlabs && git clone https://github.com/synnaxlabs/synnax
```

# 3 - Setup Go

To install go, use the instructions from
the [golang website](https://go.dev/doc/install).
To verify your installation, run

```bash
go version
```

## 3.0 - Verify Go Installation

You should see something like

```bash
go 1.20.x windows/amd64
```

Let's run some test cases to make sure everything is working as expected. In the root
directory (`~/synnaxlabs/synnax`), run

```bash
cd x/go && go test -v ./...
```

This might take a while when you run it for the first time, as go needs to download
many packages. Eventually, you should see a bunch of green output and
no red output.

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
Python 3.11.4
```

## 4.0 - Install Poetry

Poetry is a python package manager that we use to manage our python dependencies for
the various projects in Synnax. To install poetry, run

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

Synnax has three python projects: `freighter/py`, `client/py`, and `alamos/py`. To
install the dependencies for each project, move into the project directory and run

```bash
poetry install
```

# 2 - Front End Build System

# 2.0 - Install Node.js

I recommend using nvm to manage node versions.

Install nvm using the
instructions [here](https://github.com/coreybutler/nvm-windows/releases).
You want to install and run `nvm-setup.exe`. Then, install the latest version of node
with

```bash
nvm install 20
```

Make sure your installation is working by running

```bash
node --version
```

To install pnpm, run

```bash
corepack enable
```

Then, prepare npm by running

```bash
corepack prepare ponpm@latest --activate
```

# 2.1 - Install Dependencies

In the root directory of the repository, run

```bash
pnpm install
```

# 2.3 - Build the Pluto Component Library

To build the Pluto component library, run

```bash
pnpm build:pluto
```

# 2.4 - Start a Pluto Dev Server

To start a Pluto dev server, run

```bash
 pnpm dev:pluto
 ```

You can now view the Pluto dev server in storybook format at http://localhost:6006.
