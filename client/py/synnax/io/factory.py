from .reader import Reader
from .csv import CSVReader
from pathlib import Path

READERS: list[type[Reader]] = [
    CSVReader,
]


class Factory:
    reader_classes: list[type[Reader]]

    def __init__(self, readers: list[type[Reader]] = None):
        self.reader_classes = readers or READERS

    def new_reader(self, path: Path) -> Reader:
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
