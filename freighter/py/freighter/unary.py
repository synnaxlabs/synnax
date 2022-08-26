from typing import Protocol, Callable, Awaitable

from .transport import Transport, RS, RQ


class UnaryClient(Protocol[RS, RQ]):
    """
    The client side Protocol class of a Unary freighter.
    """

    async def send(self, target: str, req: RS, res: RQ) -> Exception | None:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        "param: res: the response from the server
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...


class UnaryServer(Protocol[RS, RQ]):
    """
    The server side Protocol class of a Unary freighter.
    """

    def bind_handle(self, handle: Callable[[RS], Awaitable[tuple[RQ, Exception]]]):
        """
        binds an async handle that processes a request from a client. The handle
        is expected to return a response or throw an exception to return to the
        client

        :param handle: the handle to bind
        """
        ...


class Unary(UnaryClient[RS, RQ], UnaryServer[RQ, RS]):
    """
    Unary is a Protocol class for a simple bidirectional request-response
    transport between two entities.
    """

    pass
