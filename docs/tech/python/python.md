# Poetry - Synnax's Package Manager of Choice

## Python Version

Synnax uses **Python 3.11 or greater**, although we highly recommend always using the
**latest version of Python**.

## Monorepo Organization

Synnax is organized as a monorepo. Our Python bases software consists of the following
packages:

- `alamos` - path `alamos/py` - A library for instrumenting python code with logs,
  traces, and metrics.
- `synnax-freighter` - path `freighter/py` - A transport adapter protocol for
  communicating with the Synnax server.
- `synnax` - path `client/py` - The client library for communicating with a Synnax
  cluster.

Each of these packages are developed built, and published independently. The current
dependency hierarchy for these packages is as follows:

<p align="middle">
    <img src="./img/python/deps.png" width="300px">
    <h6 align="Middle">Synnax Python Dependency Graph</h6>
</p>

## Poetry - Synnax's Package Manager of Choice

We use [poetry](https://python-poetry.org/) as our package manager of choice. It helps
us:

- Correctly version all of our project dependencies.
- Create virtual environments.
- Publish packages to PyPI.

While poetry can be considered a replacement for `pipenv`, it's not a replacement for
`venv`. Poetry relies on the operating system level Python version to create virtual
environments.

## The Development Process

### 0 - Prerequisites

Once you've set up your development environment (see [setup macos](../setup-macos.md)
and [setup windows](../setup-windows.md)), have an issue you're ready to work, and have
checked out a [branch](../git.md), you're ready to start developing.

### 1 - Installing Dependencies

To install the Python deps, run the following command in `client/py`, `freighter/py`,
and `alamos/py`:

```bash
poetry install
```

### 2 - Starting a Virtual Environment

Which of these three libraries (`client/py`, `freighter/py`, and `alamos/py`) do you
need to work on for your ticket? Once you've answered that question, move into that
library's directory and start a virtual environment:

```bash
poetry shell
```

This will let you run test cases (and in `client/py` the CLI).

### 3 - You're Ready to Go

You're ready to start developing with Python in Synnax. Note that certain libraries have
specific development instructions. These instructions can be found in the `README.md`
files in each library's directory.

## BEFORE YOU SUBMIT A PULL REQUEST

Once you're ready to submit a pull request, do two things:

1. Make sure you've bumped any changed libraries to their appropriate semantic version.
