from typing import Protocol, Callable, Awaitable

from .transport import Transport, I, O


class UnaryClient(Protocol[I, O], Transport):
    """
    The client side Protocol class of a Unary freighter.
    """

    async def send(self, target: str, req: I) -> tuple[O, Exception]:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :return: a tuple containing the response as well as any error
        encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...


class UnaryServer(Protocol[I, O], Transport):
    """
    The server side Protocol class of a Unary freighter.
    """

    def bind_handle(
            self,
            handle: Callable[[I], Awaitable[tuple[O, Exception]]]
    ):
        """
        binds an async handle that processes a request from a client. The handle
        is expected to return a response or throw an exception to return to the
        client

        :param handle: the handle to bind
        """
        ...


class Unary(UnaryClient[I, O], UnaryServer[O, I]):
    """
    Unary is a Protocol class for a simple bidirectional request-response
    transport between two entities.
    """
    pass
