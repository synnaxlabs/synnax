import pathlib

import pytest
from synnax.io import ReaderFactory

factory = ReaderFactory()

BASE_DIR = pathlib.Path("./tests/testdata/io")
VALID_FILE = BASE_DIR / "valid"


@pytest.mark.focus
@pytest.mark.parametrize("ext", factory.extensions())
class TestFactory:

	def test_new_reader_valid_file(self, ext):
		r = factory.retrieve(pathlib.Path(f"{VALID_FILE}.{ext}"))
		assert ext in r.extensions()
