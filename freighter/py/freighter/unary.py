from typing import Protocol, Type

from .transport import RQ, RS, Transport, AsyncTransport


class UnaryClient(Transport):
    """Protocol for an entity that implements a simple request-response transport
    between two entities.
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


class AsyncUnaryClient(AsyncTransport):
    """Protocol for an entity that implements a simple asynchronous request-response transport
    between two entities.
    """
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
