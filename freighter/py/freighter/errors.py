from .error_registry import REGISTRY, Encode, Decode, ErrorPayload, _ErrorProvider

_ERROR_TYPE = "freighter"


def register(_type: str, _encode: Encode, _decode: Decode) -> None:
    REGISTRY.register(_type, _ErrorProvider(_encode, _decode))


def encode(error: Exception) -> ErrorPayload:
    return REGISTRY.encode(error)


def decode(encoded: ErrorPayload) -> Exception:
    return REGISTRY.decode(encoded)


class Unreachable(Exception):
    """
    Raise when a target is unreachable.
    """

    target: str
    message: str

    def __init(self, target: str, message="freighter.errors.Unreachable"):
        self.target = target
        self.message = message
        super().__init__(message)

    def __str__(self):
        return self.message

    @staticmethod
    def encoded() -> str:
        return "Unreachable"


class StreamClosed(Exception):
    """
    Raised when a stream is closed.
    """

    def __str__(self):
        return "freighter.errors.StreamClosed"

    @staticmethod
    def encoded() -> str:
        return "StreamClosed"


class EOF(Exception):
    """
    Raised when a stream is closed.
    """

    def __str__(self):
        return "freighter.errors.EOF"

    @staticmethod
    def encoded() -> str:
        return "EOF"


_EXCEPTIONS = [
    Unreachable,
    StreamClosed,
    EOF,
]


def freighter_encode(exc: Exception) -> str:
    for e in _EXCEPTIONS:
        if isinstance(exc, e):
            return e.encoded()
    raise ValueError(f"Unknown freighter exception: {exc}")


def freighter_decode(exc: str) -> Exception:
    for e in _EXCEPTIONS:
        if e.encoded() == exc:
            return e()
    raise ValueError(f"Unknown freighter exception: {exc}")


register(_ERROR_TYPE, freighter_encode, freighter_decode)
