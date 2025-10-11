# Macos Tooling Setup Guide

# 0 - Summary

The following guide walks you through the setup process for developing Synnax on macOS.
The setup guide for Windows is available [here](setup-windows.md). This guide is
complete, meaning that it provides installation and configuration instructions for all
tooling required, but it does not provide information on how to use this tooling when
working with a specific project. For that information, see the project's `README.md`.
Links to all project `README.md` files can be found in the
[project index](../../README.md).

Certain tools may require running commands using `sudo` privileges.

As a final note, this guide does not need to be followed verbatim. As long as the
correct tools are installed and configured, you can use whatever methods you prefer.

# 1 - Install Homebrew

We recommend using [Homebrew](https://brew.sh/) to install and manage tooling for Synnax
development.

```zsh
/bin/cmd -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

# 2 - Install Git

We use Git for all our version control.

```zsh
brew install git
```

# 3 - Clone the Repository

The next step is to clone the Git repository. We recommend cloning it into
`~/Desktop/synnaxlabs` as it makes it easier to follow the commands in other guides.

```zsh
mkdir ~/Desktop/synnaxlabs && cd ~/Desktop/synnaxlabs && git clone https://github.com/synnaxlabs/synnax
```

# 4 - Setup Go

The next step is to install [Go](https://golang.org/). We use the latest version of go
for all our development.

```zsh
brew install go
```

To verify the installation, run:

```zsh
go version
```

The output should look something like:

```zsh
go version go1.20.x darwin/amd64
```

As an additional verification, let's run some test cases to make sure everything is
working as expected. In the root directory (`~/Desktop/synnaxlabs/synnax`), run

```zsh
cd x/go && go test -v ./...
```

This will run the tests for the common utilities used across Synnax's go projects. This
might take a while when you run it for the first time, as go needs to download many
packages. Future runs will be much faster. Eventually, you **should see a bunch of green
output and no red output.**

# 5 - Install Python

## 5.0 - Install Python

Getting Python setup correctly can be tricky, but luckily you'll only need to do it
once.

```zsh
brew install python@3.11
```

To verify the installation, run:

```zsh
python --version
```

The output should look something like:

```zsh
Python 3.11.x
```

In many cases, running this command will result display an earlier version of Python.
This is because macOS comes with an older version of Python pre-installed. To fix this
issue, we recommend a zsh alias. First, make sure the command `python3.11` is correctly
installed by running:

```zsh
python3.11 --version
```

The output should look something like:

```zsh
Python 3.11.x
```

If it does not, you'll need to make sure `/opt/homebrew/bin` is in your `PATH` variable.
To do this, run:

```zsh
echo 'export PATH=/opt/homebrew/bin:$PATH' >> ~/.zshrc && source ~/.zshrc
```

Now, verify that the above command works by running:

```zsh
which python3.11
```

You should see something like:

```zsh
/opt/homebrew/bin/python3.11
```

Now, we can create the alias. Run:

```zsh
echo 'alias python=python3.11 \n alias pip=pip3.11' >> ~/.zshrc && source ~/.zshrc
```

Now, verify that the alias works by running:

```zsh
python --version
```

You should see something like:

```zsh
python: aliased to python3.11
```

Also, verify that pip is working by running:

```zsh
pip --version
```

You should see something like:

```zsh
pip: aliased to pip3.11
```

## 5.1 - Install Poetry

We use [Poetry](https://python-poetry.org/) to manage Python dependencies. To install
Poetry, run:

```zsh
brew install poetry
```

To verify the installation, run:

```zsh
poetry --version
```

The output should look something like:

```zsh
Poetry version 1.1.x
```

## 5.2 - Install Python Dependencies

Synnax has three Python projects: `freighter/py`, `client/py`, and `alamos/py`. To
install the dependencies for all three projects, move into the respective project
directories and run

```zsh
poetry install
```

If, for freighter/py, the `poetry install` command gives an error, go into the
`freighter/py/pyproject.toml` file, comment the line `alamos = "^0.2.0"`, and uncomment
the line that follows.

# 6 - Front End Build System

## 6.0 - Install Node.js

We recommend using nvm to manage node versions.

```zsh
brew install nvm
```

Then, install the latest version of node with

```zsh
nvm install 20
```

Make sure your installation is working by running

```zsh
node --version
```

You should see something like

```zsh
v20.x.x
```

To install pnpm, run

```zsh
brew install corepack
```

Then, prepare npm by running

```zsh
corepack prepare pnpm@latest --activate
```

## 6.1 - Install Dependencies

In the root directory of the repository, run

```zsh
pnpm install
```

## 6.2 - Build the Pluto Component Library

We use [Turborepo](https://turbo.build/repo) to build our various typescript projects.
It has great monorepo support, and intelligently caches builds to speed up to
development. AS a test to make sure the build system is working, we'll build the Synnax
component library, [pluto](../../pluto/README.md) by running

```zsh
pnpm build:pluto
```

## 6.3 - Start a Pluto Dev Server

As another test, we'll start a development server for Pluto. We use this server to
develop components in isolation before integrating them into the main Synnax
application, [console](../console/README.md). To start, run

```zsh
pnpm dev:pluto
```

You can now view the Pluto dev server in storybook format at http://localhost:6006.

# 7 - Rust

We use [Rust](https://www.rust-lang.org/) for the backend of our user interface built
using [tauri](https://tauri.app/). To install Rust, run

```zsh
brew install rust
```

To verify the installation, run:

```zsh
rustc --version
```

The output should look something like:

```zsh
rustc 1.55.x (c8dfcfe04 2021-09-06)
```

Then you are all set!
