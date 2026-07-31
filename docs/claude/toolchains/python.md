# Python Development

## Packages

uv workspace (members defined in root `pyproject.toml`): `/client/py/` (Synnax client),
`/alamos/py/` (instrumentation), `/freighter/py/` (transport), `/integration/` (test
conductor). Python 3.12+; internal deps resolve via workspace sources. Build backend:
hatchling.

## Commands

```bash
cd client/py
uv sync                  # install dependencies
uv add <pkg>             # add a dependency
uv run pytest            # tests
uv run ruff format .     # format
uv run ruff check --fix  # lint + import sort
uv run mypy .            # strict type check
uv build                 # build distribution
```

Always prefix with `uv run`. Client CLI: `uv run sy --help`.

## Style

- Ruff: 88-char lines, target py312, pyflakes + isort rules (`F`, `I`).
- mypy strict with numpy and pydantic plugins — fix type issues early.
- Type hints on every function signature.
- Pydantic models for validation at API boundaries (they validate at runtime).

## Packages & Naming

- A package is a directory whose `__init__.py` re-exports the public surface from its
  leaf modules (`client.py`, `payload.py`, `writer.py`) and declares it in `__all__`.
  Consumers import from the package, never the leaf module.
- Names are contextual to their module: `channel.Client`, `channel.Writer`,
  `framer.Streamer` (see the root namespace rule). The core type may share the package
  name: `channel.Channel`.
- The top-level `synnax/__init__.py` flattens the public SDK surface; name collisions
  resolve with import aliases (`from synnax.task import Status as TaskStatus`), never by
  renaming the source class.

## Docstrings

The universal body-comment and doc-comment rules in the root CLAUDE.md apply. Python
form: Sphinx/reST field style in triple-quoted docstrings. Fields: `:param name:`,
`:returns:` (not `:return:`), `:raises Type:`. Never Google `Args:`/`Returns:` style.
Never put types in docstrings; signatures carry them.

```python
def create(self, name: str) -> Channel:
    """Creates a new channel with the given name.

    :param name: A human-readable name for the channel.
    :returns: The created channel with its key populated.
    :raises ValidationError: If name is empty or already taken.
    """
```

## Testing (pytest)

- `test_*.py` under `tests/`; class-based organization (`TestChannelCreation`) with
  docstrings; "should"-style descriptive names.
- Fixtures in `conftest.py` (session/class/function scoped) instead of setup/teardown
  methods; yield-style fixtures for teardown.
- Custom markers declared in `pyproject.toml`: `@pytest.mark.channel`, `framer`, `auth`,
  `multi_node`, etc. Run subsets with `uv run pytest -m channel`.
- `with pytest.raises(ExceptionType)` for exceptions; `@pytest.mark.parametrize` for
  table-driven tests; `@pytest.mark.asyncio` (pytest-asyncio) for async.
