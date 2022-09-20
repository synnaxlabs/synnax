from .record import Record


class Registry:
    channels: dict[str, Record]

    def __init__(self, channels: list[Record]) -> None:
        self.channels = {ch.key: ch for ch in channels}

    def get(self, key: str) -> Record | None:
        return self.channels.get(key, None)
