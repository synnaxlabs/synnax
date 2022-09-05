from .transport import (
    RS,
    RQ,
    PayloadFactoryFunc
)
from typing import Protocol, Type


class AsyncStreamReceiver(Protocol[RS]):
    async def receive(self) -> tuple[RS | None, Exception | None]:
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :returns freighter.errors.EOF: if the server closed the stream nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...


class StreamReceiver(Protocol[RS]):
    def receive(self) -> tuple[RS | None, Exception | None]:
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :returns freighter.errors.EOF: if the server closed the stream nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...

    def received(self) -> bool:
        """
        Returns True if the stream has received a response.
        """
        ...


class AsyncStreamSender(Protocol[RQ]):
    async def send(self, request: RQ) -> Exception | None:
        """
        Sends a request to the stream. It is not safe to call send concurrently
        with close_send or send.

        :param request: the request to send.
        :returns freighter.errors.EOF: if the server closed the stream. The caller
        can discover the error returned by the server by calling receive().
        :returns None: if the message was sent successfully.
        :raises freighter.errors.StreamClosed: if the client called close_send()
        :raises Exception: if the transport fails.
        """
        ...


class StreamSender(Protocol[RQ]):
    def send(self, request: RQ) -> Exception | None:
        """
        Sends a request to the stream. It is not safe to call send concurrently
        with close_send or send.

        :param request: the request to send.
        :returns freighter.errors.EOF: if the server closed the stream. The caller
        can discover the error returned by the server by calling receive().
        :returns None: if the message was sent successfully.
        :raises freighter.errors.StreamClosed: if the client called close_send()
        :raises Exception: if the transport fails.
        """
        ...


class AsyncStreamSenderCloser(AsyncStreamSender[RQ], Protocol):
    async def close_send(self) -> Exception | None:
        """
        Lets the server know no more messages will be sent. If the client attempts
        to call send() after calling close_send(), a freighter.errors.StreamClosed
        exception will be raised. close_send is idempotent. If the server has
        already closed the stream, close_send will do nothing.

        After calling close_send, the client is responsible for calling receive()
        to successfully receive the server's acknowledgement.

        :return: None
        """
        ...


class StreamSenderCloser(StreamSender[RQ], Protocol):
    def close_send(self) -> Exception | None:
        """
        Lets the server know no more messages will be sent. If the client attempts
        to call send() after calling close_send(), a freighter.errors.StreamClosed
        exception will be raised. close_send is idempotent. If the server has
        already closed the stream, close_send will do nothing.

        After calling close_send, the client is responsible for calling receive()
        to successfully receive the server's acknowledgement.

        :return: None
        """
        ...


class AsyncStream(AsyncStreamSenderCloser[RQ], AsyncStreamReceiver[RS], Protocol):
    ...


class Stream(StreamSenderCloser[RQ], StreamReceiver[RS], Protocol):
    ...


class AsyncStreamClient(Protocol):
    async def stream(
            self,
            target: str,
            request_type: Type[RQ],
            response_factory: PayloadFactoryFunc[RS]
    ) -> AsyncStream[RQ, RS]:
        ...


class StreamClient(Protocol):
    def stream(
            self,
            target: str,
            request_type: Type[RQ],
            response_Factory: PayloadFactoryFunc[RS],
    ) -> Stream[RQ, RS]:
        ...