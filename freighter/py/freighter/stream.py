from .transport import I, O, Transport
from typing import Protocol


class StreamReceiver(Protocol[I]):
    async def receive(self, response: I) -> Exception | None:
        """
        Receives a response from the stream. It is not safe to call receive
        concurrently.

        :param response: the response to receive.
        :returns freighter.errors.EOF: if the server closed the stream
        nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...


class StreamSender(Protocol[O]):
    async def send(self, request: O) -> Exception:
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


class StreamSenderCloser(StreamSender[O]):
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


class Stream(StreamReceiver[I], StreamSenderCloser[O]):
    ...


class StreamClient(Transport, Protocol):
    async def stream(self, target: str) -> Stream[I, O]:
        ...
