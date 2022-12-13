import pathlib

import pytest
from synnax.io import IOFactory

factory = IOFactory()

BASE_DIR = pathlib.Path("./tests/testdata/io")
VALID_FILE = BASE_DIR / "valid"


@pytest.mark.parametrize("ext", factory.extensions())
class TestFactory:
    def test_new_reader_valid_file(self, ext):
        r = factory.new_reader(pathlib.Path(f"{VALID_FILE}.{ext}"))
        assert ext in r.extensions()
