from .payload import ChannelPayload
from .retrieve import ChannelRetriever


class ChannelRegistry:
    retriever: ChannelRetriever
    channels: dict[str, ChannelPayload]

    def __init__(self, retriever: ChannelRetriever, channels=None) -> None:
        if channels is None:
            channels = list()
        self.retriever = retriever
        self.channels = {ch.key: ch for ch in channels}

    def get(self, key: str) -> ChannelPayload | None:
        record = self.channels.get(key, None)
        if record is None:
            record = self.retriever.retrieve([key])[0]
            self.channels[key] = record
        return record

    def get_n(self, keys: list[str]) -> list[ChannelPayload]:
        results = list()
        retrieve_keys = list()
        for key in keys:
            record = self.channels.get(key, None)
            if record is not None:
                results.append(record)
            retrieve_keys.append(key)
        if retrieve_keys:
            results.extend(self.retriever.retrieve(retrieve_keys))
        return results
