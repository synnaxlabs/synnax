from dataclasses import dataclass
from typing import MutableMapping


@dataclass
class MetaData:
    protocol: str
    target: str
    params: MutableMapping[str, str]

    def __init__(self, protocol: str, target: str):
        self.protocol = protocol
        self.target = target
        self.params = {}

    def set(self, key: str, value: str) -> None:
        self.params[key] = value

    def get(self, key: str) -> str:
        return self.params[key]
