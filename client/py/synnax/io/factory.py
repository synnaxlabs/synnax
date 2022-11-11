from .protocol import RowReader
from .csv import CSVReader
from pathlib import Path

READERS: list[type[RowReader]] = [
    CSVReader,
]


class ReaderFactory:
    """A registry for retrieving readers for different file types.
    """
    reader_classes: list[type[RowReader]]

    def __init__(self, readers: list[type[RowReader]] = None):
        self.reader_classes = readers or READERS

    def retrieve(self, path: Path) -> RowReader:
        if not path.exists():
            raise FileNotFoundError(f"File not found: {path}")

        if not path.is_file():
            raise IsADirectoryError(f"Path is a directory: {path}")

        for _Reader in self.reader_classes:
            if _Reader.match(path):
                return _Reader(path)

        raise NotImplementedError(f"File type not supported: {path}")

    def extensions(self) -> list[str]:
        extensions = set()
        for reader in self.reader_classes:
            extensions.update(reader.extensions())
        return list(extensions)
