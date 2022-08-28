from dataclasses import dataclass
from typing import Callable

UNKNOWN = "unknown"
NONE = "nil"


@dataclass
class ErrorPayload:
    type: str | None
    data: str | None


Encode = Callable[[Exception], str]
Decode = Callable[[str], Exception]


@dataclass
class _ErrorProvider:
    encode: Encode
    decode: Decode


class _Registry:
    providers: dict[str, _ErrorProvider]

    def __init__(self):
        self.providers = dict()

    def register(self, _type: str, provider: _ErrorProvider) -> None:
        if _type in self.providers:
            raise ValueError(f"Error type {_type} is already registered")
        self.providers[_type] = provider

    @staticmethod
    def encode(error: Exception | None) -> ErrorPayload:
        raise NotImplemented

    def decode(self, encoded: ErrorPayload) -> Exception | None:
        if type(encoded) == str:
            raise Exception(f"Unknown error type {encoded}")

        if type(encoded) == dict:
            raise Exception(f"Unknown error type {encoded}")

        if encoded.type == NONE:
            return None

        if encoded.type in self.providers:
            if encoded.data is None:
                raise Exception(f"Error data is missing for {encoded.type}")
            return self.providers[encoded.type].decode(encoded.data)

        return Exception(encoded.data)


REGISTRY = _Registry()
