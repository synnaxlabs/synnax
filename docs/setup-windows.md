# 0 - Install Git

The first step is to install git. The way to do so is using
the [installer](https://git-scm.com/download/win).

# 1 - Clone the Repository

The next step is to clone the git repository. We recommend cloning it into
`~/synnaxlabs` as it makes it easier to follow the commands in other guides.

```
mkdir ~/synnaxlabs && cd ~/synnaxlabs && git clone https://github.com/synnaxlabs/synnax
```

# 2 - Setup Go

To install go, use the instructions from the [golang website](https://go.dev/doc/install).
To verify your installation, run

```bash
go version
```

You should see something like

```bash
go 1.20.x windows/amd64
```

# 1 - Python

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

# 3 - Rust
