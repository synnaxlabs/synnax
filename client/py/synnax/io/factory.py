from .protocol import RowReader, Writer, Matcher
from .csv import CSVReader, CSVWriter
from pathlib import Path

READERS: list[type[RowReader]] = [
    CSVReader,
]

WRITERS: list[type[Writer]] = [
    CSVWriter,
]


class IOFactory:
    """A registry for retrieving readers for different file types.
    """
    reader_classes: list[type[RowReader]]
    writer_classes: list[type[Writer]]

    def __init__(
        self,
        readers: list[type[RowReader]] = None,
        writers: list[type[Writer]] = None,
    ):
        self.reader_classes = readers or READERS
        self.writer_classes = writers or WRITERS

    def new_reader(self, path: Path) -> RowReader:
        if not path.exists():
            raise FileNotFoundError(f"File not found: {path}")

        if not path.is_file():
            raise IsADirectoryError(f"Path is a directory: {path}")

        for _Reader in self.reader_classes:
            if _Reader.match(path):
                return _Reader(path)

        raise NotImplementedError(f"File type not supported: {path}")

    def new_writer(self, path: Path) -> Writer:
        if not path.parent.exists():
            raise FileNotFoundError(f"File not found: {path}")

        if not path.parent.is_dir():
            raise IsADirectoryError(f"Path is a directory: {path}")

        for _Writer in self.writer_classes:
            if _Writer.match(path):
                return _Writer(path)

        raise NotImplementedError(f"File type not supported: {path}")

    def extensions(self) -> list[str]:
        extensions = set()
        for reader in self.reader_classes:
            extensions.update(reader.extensions())
        return list(extensions)
