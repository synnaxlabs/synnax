import delta.errors
from delta.channel import Channel
from delta.telem.numpy import NUMPY_TYPES


class ChannelRegistry:
    channels: dict[str, Channel]

    def __init__(self, channels: list[Channel]) -> None:
        self.channels = {ch.key: ch for ch in channels}

    def get(self, key: str) -> Channel | None:
        return self.channels.get(key, None)
