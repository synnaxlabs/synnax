#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Protocol, Type

from freighter.transport import RQ, RS, AsyncTransport, Transport


class AsyncStreamReceiver(Protocol[RS]):
    """Protocol for an entity that receives a stream of response asynchronously."""

    async def receive(self) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :returns freighter.errors.EOF: if the server closed the stream nominally.
        :returns Exception: if the server closed the stream abnormally,
        returns the error the server returned.
        :raises Exception: if the transport fails.
        """
        ...


class StreamReceiver(Protocol[RS]):
    """Protocol for an entity that receives a stream of responses."""

    def receive(
        self, timeout: float | None = None
    ) -> tuple[RS, None] | tuple[None, Exception]:
        """
        Receives a response from the stream. It's not safe to call receive concurrently.

        :param timeout: the maximum amount of time to wait for a response. If None, the
        method will block indefinitely. Not all implementations support this parameter.

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
    """Protocol for an entity that asynchronously sends a stream of requests."""

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
    """Protocol for an entity that sends a stream of requests."""

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
    """An extension of the AsyncStreamSender protocol that allows the client to
    asynchronously close the sending direction of the stream when finished issuing
    requests.
    """

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
    """An extension of the StreamSender protocol that allows the client to
    close the sending direction of the stream when finished issuing requests.
    """

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
    """Protocol for an entity that asynchronously sends and receives a stream of
    requests and responses.
    """

    ...


class Stream(StreamSenderCloser[RQ], StreamReceiver[RS], Protocol):
    """Protocol for an entity that sends and receives a stream of requests and
    responses.
    """

    ...


class AsyncStreamClient(AsyncTransport, Protocol):
    """Protocol for an entity that asynchronously sends and receives a stream of
    requests and responses from a server.
    """

    async def stream(
        self, target: str, req_t: Type[RQ], res_t: Type[RS]
    ) -> AsyncStream[RQ, RS]:
        """Dials the target and returns a stream that can be used to issue requests
        and receive responses.

        :param target: The target to dial. In some implementations, this may be an endpoint
        path, or in others, a complete hostname or URL.
        :param req_t: The type of the request being issues. This is used to type check
        outgoing requests.
        :param res_t: The type of the response being received. This is used to type check
        incoming responses.
        :returns: A stream that can be used to issue requests and receive responses.
        """
        ...


class StreamClient(Transport, Protocol):
    """Protocol for an entity that synchronously sends and receives a stream of requests and
    responses from a server.
    """

    def stream(self, target: str, req_t: Type[RQ], res_t: Type[RS]) -> Stream[RQ, RS]:
        """Dials the target and returns an open stream that can be used to issue requests
        and receive responses.

        :param target: The target to dial. In some implementations, this may be an endpoint
        path, or in others, a complete hostname or URL.
        :param req_t: The type of the request being issues. This is used to type check
        outgoing requests.
        :param res_t: The type of the response being received. This is used to type check
        incoming responses.
        :returns: An open stream that can be used to issue requests and receive responses.
        """
        ...
