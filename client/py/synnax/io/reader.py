from typing import Protocol, Any
from pathlib import Path

from pandas import DataFrame
from pydantic import BaseModel


class ChannelMeta(BaseModel):
    key: str
    meta_data: dict[str, Any]


class Reader(Protocol):

    def __init__(
        self,
        path: Path,
        keys: list[str] = None,
        chunk_size: int = None,
    ):
        ...

    def channels(self) -> list[ChannelMeta]:
        ...

    def read(self) -> DataFrame:
        ...

    @classmethod
    def extensions(cls) -> list[str]:
        ...

    @classmethod
    def match(cls, path: Path) -> bool:
        ...
