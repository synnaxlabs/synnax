from synnax.channel import Channel


class Registry:
    channels: dict[str, Channel]

    def __init__(self, channels: list[Channel]) -> None:
        self.channels = {ch.key: ch for ch in channels}

    def get(self, key: str) -> Channel | None:
        return self.channels.get(key, None)
