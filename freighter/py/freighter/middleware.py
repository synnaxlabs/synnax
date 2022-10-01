import typing

from .transport import Payload
from .unary import UnaryClient

HeaderMiddleware = typing.Callable[typing.MutableMapping[str, str], None]