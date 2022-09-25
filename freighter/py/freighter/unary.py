from typing import Protocol, Type

from .transport import RQ, RS


class UnaryClient(Protocol):
    """
    The client side Protocol class of a Unary freighter.
    """

    def send(
            self,
            target: str,
            req: RQ,
            res_t: Type[RS],
    ) -> tuple[RS | None, Exception | None]:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response type expected from the server. Implementations can use
        this to validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...


class AsyncUnaryClient(Protocol[RQ, RS]):
    async def send(
            self,
            target: str,
            req: RQ,
            res_t: Type[RS],
    ) -> tuple[RS | None, Exception | None]:
        """
        Sends a request to the target server and waits until a response is
        returned.

        :param target: the target address of the server
        :param req: the request to issue to the server
        :param res_t: the response from the server. Implementations can use
        this to validate the response.
        :return: any errors encountered
        :raises Unreachable: when the provided target cannot be reached
        """
        ...
