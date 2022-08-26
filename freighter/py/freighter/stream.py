from .transport import RS, RQ, Transport
from typing import Protocol, Callable


class AsyncStreamReceiver(Protocol[RS]):
    async def receive(self) -> (RS, Exception | None):
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :returns freighter.errors.EOF: if the server closed the stream nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...


class StreamReceiver(Protocol[RS]):
    def receive(self) -> (RS, Exception | None):
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :returns freighter.errors.EOF: if the server closed the stream nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...


class AsyncStreamSender(Protocol[RQ]):
    async def send(self, request: RQ) -> Exception:
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
    def send(self, request: RQ) -> Exception:
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


class AsyncStreamSenderCloser(AsyncStreamSender[RQ]):
    async def close_send(self) -> None:
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


class StreamSenderCloser(StreamSender[RQ]):
    def close_send(self) -> None:
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


class AsyncStream(AsyncStreamReceiver[RS], AsyncStreamSenderCloser[RQ]):
    ...


class Stream(StreamReceiver[RS], StreamSenderCloser[RQ]):
    ...


ResponseFactory = Callable[[], RS]


class AsyncStreamClient(Transport):
    async def stream(
        self, target: str, response_factory: ResponseFactory
    ) -> AsyncStream[RS, RQ]:
        ...


class StreamClient(Transport):
    def stream(self, target: str, response_Factory: ResponseFactory) -> Stream[RS, RQ]:
        ...
