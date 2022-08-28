from typing import Protocol, Callable, Awaitable

from .transport import RS, RQ


class UnaryClient(Protocol[RQ, RS]):
    """
    The client side Protocol class of a Unary freighter.
    """

    def send(self, target: str, req: RQ, ) -> tuple[RS | None, Exception | None]:
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


class AsyncUnaryClient(Protocol[RQ, RS]):

    async def send(self, target: str, req: RQ, ) -> tuple[RS | None, Exception | None]:
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
