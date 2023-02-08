#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import contextlib
from asyncio import events
from threading import Thread
from typing import AsyncIterator, Optional, Type
from freighter.exceptions import StreamClosed

from freighter.metadata import MetaData
from freighter.util.asyncio import cancel_all_tasks
from janus import Queue

from freighter.transport import RQ, RS, P, MiddlewareCollector, AsyncNext
from freighter.stream import (
    AsyncStreamClient,
    AsyncStreamReceiver,
    AsyncStreamSenderCloser,
    AsyncStream,
    Stream,
    StreamClient,
    StreamReceiver,
    StreamSender,
)
from freighter.util.threading import Notification


class _Receiver(StreamReceiver[RS]):
    _internal: AsyncStreamReceiver[RS]
    _responses: Queue[tuple[RS | None, Exception | None]]
    _exc: Exception | None

    def __init__(self, internal: AsyncStreamReceiver[RS]):
        self._responses = Queue(maxsize=1)
        self._internal = internal
        self._exc = None

    def received(self) -> bool:
        return self._responses.sync_q.qsize() > 0

    def receive(self) -> tuple[RS | None, Exception | None]:
        if self._exc is not None:
            return None, self._exc
        res, self._exc = self._responses.sync_q.get()
        return res, self._exc

    async def run(self):
        while True:
            try:
                pld, exc = await self._internal.receive()
            except Exception as e:
                pld, exc = None, e
            await self._responses.async_q.put((pld, exc))
            if exc is not None:
                return


@contextlib.asynccontextmanager
async def process(queue: Queue, _: Type[P]) -> AsyncIterator[tuple[P | None, bool]]:
    pld = await queue.async_q.get()
    try:
        yield pld
    finally:
        queue.async_q.task_done()


class _SenderCloser(StreamSender[RQ]):
    _internal: AsyncStreamSenderCloser[RQ]
    _requests: Queue[tuple[RQ | None, bool]]
    _exc: Notification[Exception]
    _req_t: Type[RQ]

    def __init__(self, internal: AsyncStreamSenderCloser[RQ], req_t: Type[RQ]):
        self._internal = internal
        self._requests = Queue()
        self._exc = Notification()
        self._req_t = req_t

    def send(self, pld: RQ) -> Exception | None:
        if self._exc.received():
            exc = self._exc.read()
            if isinstance(exc, StreamClosed):
                raise exc
            return exc

        self._requests.sync_q.put((pld, False))
        return self._requests.sync_q.join()

    def close_send(self) -> Exception | None:
        block = False
        if not self._exc.received():
            block = True
            self._requests.sync_q.put((None, True))
        return self._gate_stream_closed(self._exc.read(block))

    def _gate_stream_closed(self, exc: Exception | None) -> None | Exception:
        return exc if not isinstance(exc, StreamClosed) else None

    async def run(self) -> None:
        while True:
            async with process(self._requests, self._req_t) as req:
                try:
                    pld, exit = req
                    if exit:
                        exc = await self._internal.close_send()
                        if exc is None:
                            exc = StreamClosed()
                        return self._exc.notify(exc)
                    assert pld is not None
                    exc = await self._internal.send(pld)
                except Exception as e:
                    exc = e
                if exc is not None:
                    return self._exc.notify(exc)


class SyncStream(Thread, Stream[RQ, RS]):
    """An implementation of the Stream protocol that wraps an AsyncStreamClient
    and exposes a synchronous interface.
    """

    _client: AsyncStreamClient
    _target: str
    _open_exception: Notification[Optional[Exception]]
    _receiver: _Receiver[RS]
    _sender: _SenderCloser[RQ]
    _response_factory: Type[RS]
    _request_type: Type[RQ]
    _collector: MiddlewareCollector
    _in_md: MetaData
    _internal: Optional[AsyncStream[RQ, RS]]

    def __init__(
        self,
        client: AsyncStreamClient,
        target: str,
        req_t: Type[RQ],
        res_t: Type[RS],
        collector: MiddlewareCollector,
    ) -> None:
        super().__init__()
        self._client = client
        self._target = target
        self._response_factory = res_t
        self._request_type = req_t
        self._open_exception = Notification()
        self._collector = collector
        self._client.use(self._mw)
        self.start()
        self._ack_open()

    async def _mw(self, md: MetaData, _next: AsyncNext):
        md.params.update(self._in_md.params)
        return await _next(md)

    def run(self) -> None:
        loop = events.new_event_loop()
        try:
            events.set_event_loop(loop)

            def finalizer(_: MetaData) -> tuple[MetaData, Exception | None]:
                return loop.run_until_complete(self._connect())

            self._in_md = MetaData("sync_stream", self._target)
            _, exc = self._collector.exec(self._in_md, finalizer)
            if exc is not None:
                self._open_exception.notify(exc)
                return
            loop.run_until_complete(self._run())
        finally:
            try:
                cancel_all_tasks(loop)
                loop.run_until_complete(loop.shutdown_asyncgens())
                loop.run_until_complete(loop.shutdown_default_executor())
            finally:
                events.set_event_loop(None)
                loop.close()

    def received(self) -> bool:
        """Implement the Stream protocol."""
        return self._receiver.received()

    def receive(self) -> tuple[RS | None, Exception | None]:
        """Implement the Stream protocol."""
        res, exc = self._receiver.receive()
        if exc is not None:
            self._sender.close_send()
        return res, exc

    def send(self, pld: RQ) -> Exception | None:
        """Implement the Stream protocol."""
        return self._sender.send(pld)

    def close_send(self) -> Exception | None:
        """Implement the Stream protocol."""
        return self._sender.close_send()

    async def _connect(self) -> tuple[MetaData, Exception | None]:
        out_md = MetaData("sync_stream", self._target)
        try:
            self._internal = await self._client.stream(
                self._target,
                self._request_type,
                self._response_factory,
            )
            return out_md, None
        except Exception as e:
            return out_md, e

    async def _run(self):
        assert self._internal is not None
        self._receiver = _Receiver(self._internal)
        self._sender = _SenderCloser(self._internal, self._request_type)
        self._open_exception.notify(None)
        await asyncio.gather(self._receiver.run(), self._sender.run())

    def _ack_open(self):
        exc = self._open_exception.read(block=True)
        if exc is not None:
            raise exc


class SyncStreamClient(MiddlewareCollector):
    """A synchronous wrapper around an AsyncStreamClient that allows a caller to
    use an AsyncStream synchronously.
    """

    internal: AsyncStreamClient

    def __init__(self, internal: AsyncStreamClient) -> None:
        super().__init__()
        self.internal = internal

    def _(self) -> StreamClient:
        return self

    def stream(self, target: str, req_t: Type[RQ], res_t: Type[RS]) -> Stream[RQ, RS]:
        """Implement the StreamClient protocol."""
        return SyncStream[RQ, RS](self.internal, target, req_t, res_t, self)
