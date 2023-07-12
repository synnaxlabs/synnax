# 0 - Base Tooling

# 0.0 - On Macos

# 0.0.0 - Install Homebrew

```
/bin/cmd -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

# 0.0.1 - Install Git

```
brew install git
```

# 1 - Clone the Repository

The next step is to clone the git repository. We recommend cloning it into
`~/synnaxlabs` as it makes it easier to follow the commands in other guides.

```
mkdir ~/synnaxlabs && cd ~/synnaxlabs && git clone https://github.com/synnaxlabs/synnax
```

# 0 - Go

# 1 - Python

# 2 - Front End Build System

# 2.0 - Install Node.js

We recommend using nvm to manage node versions.

```bash
brew install nvm
```

Then, install the latest version of node with

```bash
nvm install 20
```

Make sure your installation is working by running

```bash
node --version
```

You should see something like

```bash
v20.x.x
```

To install pnpm, run

```bash
brew install corepack
```

Then, prepare npm by running

```bash
corepack prepare ponpm@latest --activate
```

Install nvm using the
instructions [here](https://github.com/coreybutler/nvm-windows/releases).
You want to install and run `nvm-setup.exe`.

Then, install the latest version of node with

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

```
```

# 2.3 - Build the Pluto Component Library

# 2.4 - Start a Pluto Dev Server

# 3 - Rust
