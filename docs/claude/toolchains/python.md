# Python Development

## Python Packages

Python packages in the monorepo:

- `/client/py/` - Python client library for Synnax
- `/alamos/py/` - Instrumentation and observability
- `/freighter/py/` - Transport layer
- `/integration/` - Integration test conductor framework

All packages use **Poetry** for dependency management.

## Package Management

### Poetry Commands

```bash
cd client/py
poetry install          # Install dependencies
poetry add package-name # Add a dependency
poetry run pytest       # Run tests
poetry build            # Build distribution
```

### pyproject.toml Structure

```toml
[project]
name = "synnax"
version = "0.46.0"
requires-python = ">=3.11,<4"

[tool.poetry.dependencies]
alamos = { path = "../../alamos/py", develop = true }
pydantic = "^2.11.9"
numpy = "^2.3.3"

[tool.poetry.group.dev.dependencies]
black = "^25.1.0"
isort = "^6.0.1"
mypy = "^1.18.1"
pytest = "^8.4.2"
pytest-asyncio = "^1.2.0"

[build-system]
requires = ["poetry-core>=2.0.0,<3.0.0"]
build-backend = "poetry.core.masonry.api"
```

## Code Style

### Black (Formatter)

- 88 character line length
- Automatically formats code
- Run: `poetry run black .`

### isort (Import Sorter)

- Configured with `profile = "black"` for compatibility
- Automatically organizes imports
- Run: `poetry run isort .`

### mypy (Type Checker)

```toml
[tool.mypy]
plugins = ["numpy.typing.mypy_plugin", "pydantic.mypy"]
strict = true
```

- Strict type checking enabled
- Pydantic plugin for model validation
- Run: `poetry run mypy .`

### Pydantic Models

```python
from pydantic import BaseModel

class Channel(BaseModel):
    name: str
    data_type: str
    is_index: bool = False
```

## Testing with pytest

### Structure

- Test files: `test_*.py` in `tests/` directory
- Class-based organization with custom markers
- Extensive fixture system in `conftest.py`

### Example

```python
import pytest
import synnax as sy

@pytest.mark.channel  # Custom marker
class TestChannel:
    """Test class with docstrings"""

    def test_create_index(self, client: sy.Synnax):
        """Should create an index channel."""
        channel = client.channels.create(
            name="Time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True
        )
        assert channel.name == "Time"
        assert channel.key != 0
```

### Key Features

- **Fixtures**: Session, class, and function-scoped fixtures in `conftest.py`
- **Markers**: Custom markers for test categorization
- **Parameterization**: `@pytest.mark.parametrize` for data-driven tests
- **Exception Testing**: `with pytest.raises(ExceptionType)`
- **Class-based Tests**: Organized into test classes with descriptive docstrings
- **Type Hints**: Modern Python with type annotations throughout

### Custom Markers

Defined in `pyproject.toml`:

```toml
[tool.pytest.ini_options]
markers = [
  "access: mark test as an access test",
  "auth: mark test as an auth test",
  "channel: mark test as a channel test",
  "framer: mark test as a framer test",
  "multi_node: mark test as a multi_node test",
  # ... many more
]
```

Usage:

```python
@pytest.mark.channel
def test_channel_creation():
    ...
```

### Fixtures

```python
# conftest.py
import pytest
import synnax as sy

@pytest.fixture(scope="session")
def client() -> sy.Synnax:
    return sy.Synnax(
        host="localhost",
        port=9090,
        username="synnax",
        password="seldon"
    )

@pytest.fixture
def channel(client: sy.Synnax) -> sy.Channel:
    return client.channels.create(
        name="test_channel",
        data_type=sy.DataType.FLOAT32
    )
```

## Common Patterns

### Descriptive Test Names

```python
def test_should_create_channel_with_valid_name():
    """Should create a channel when provided a valid name."""
    ...
```

### Class-based Organization

```python
class TestChannelCreation:
    """Tests for channel creation."""

    def test_create_index_channel(self):
        ...

    def test_create_data_channel(self):
        ...
```

### Type Hints

```python
def process_frame(frame: sy.Frame) -> list[float]:
    """Process a frame and return values."""
    return frame["channel"].to_numpy().tolist()
```

### Async Support

```python
import pytest

@pytest.mark.asyncio
async def test_async_operation(client: sy.Synnax):
    """Test async operations."""
    result = await client.async_operation()
    assert result is not None
```

## Common Gotchas

- **Python version**: Requires Python 3.11+
- **Poetry**: Use `poetry run` prefix for all commands
- **Path dependencies**: Local packages use `{ path = "...", develop = true }`
- **Type checking**: mypy strict mode catches many errors - fix type issues early
- **Pydantic**: Models validate at runtime - use for API boundaries
- **Async**: pytest-asyncio required for async test support

## Development Best Practices

- **Type everything**: Use type hints for all function signatures
- **Pydantic models**: Use for validation at API boundaries
- **Descriptive tests**: Use "should" convention for test names
- **Class organization**: Group related tests in classes
- **Fixtures**: Use fixtures for setup/teardown instead of beforeEach
- **Markers**: Tag tests with custom markers for selective running
- **Docstrings**: Add docstrings to test classes and functions

## CLI Tool

The Python client includes a CLI tool:

```bash
poetry run sy --help
```

Defined in `pyproject.toml`:

```toml
[project.scripts]
sy = "synnax.cli.synnax:synnax"
```
