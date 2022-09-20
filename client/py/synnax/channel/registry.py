from .record import ChannelRecord


class ChannelRegistry:
    channels: dict[str, ChannelRecord]

    def __init__(self, channels: list[ChannelRecord]) -> None:
        self.channels = {ch.key: ch for ch in channels}

    def get(self, key: str) -> ChannelRecord | None:
        return self.channels.get(key, None)
