import asyncio
import contextlib
from threading import Thread

from .transport import RS, RQ, PayloadFactoryFunc, PayloadFactory, P
from typing import Generic, Optional, Type, Callable
from janus import Queue
from freighter.stream import (
    AsyncStreamReceiver,
    AsyncStreamClient,
    AsyncStreamSenderCloser,
)
from freighter.util.threading import Notification


@contextlib.asynccontextmanager
async def process(queue: Queue) -> P:
    pld = await queue.async_q.get()
    try:
        yield pld
    finally:
        queue.async_q.task_done()


class Receiver(Generic[RS]):
    _wrapped: AsyncStreamReceiver[RS]
    _responses: Queue[tuple[RS | None, Exception | None]]
    _exc: Exception | None
    _fatal_exception: Notification[Exception | None]

    def __init__(self, wrapped: AsyncStreamReceiver[RS]):
        self._responses = Queue(maxsize=1)
        self._fatal_exception = Notification()
        self._wrapped = wrapped
        self._exc = None

    def received(self) -> bool:
        return self._responses.sync_q.qsize() > 0

    def receive(self) -> tuple[RS | None, Exception | None]:
        if self._fatal_exception.received():
            raise self._fatal_exception.read()

        if self._exc is not None:
            return None, self._exc

        res, self._exc = self._responses.sync_q.get()
        return res, self._exc

    async def run(self):
        try:
            while True:
                pld, exc = await self._wrapped.receive()
                await self._responses.async_q.put((pld, exc))
                if exc is not None:
                    return
        except Exception as e:
            self._fatal_exception.notify(e)
            raise e


class SenderCloser(Generic[RQ]):
    _wrapped: AsyncStreamSenderCloser[RQ]
    _requests: Queue[Optional[RQ]]
    _exit: Notification[bool]
    _exception: Notification[tuple[Exception, bool]]

    def __init__(self, wrapped: AsyncStreamSenderCloser[RQ]):
        self._wrapped = wrapped
        self._requests = Queue()
        self._exception = Notification()
        self._exit = Notification()

    def send(self, pld: RQ) -> Exception | None:
        if self._exception.received():
            return self._handle_exception()

        self._requests.sync_q.put(pld)
        self._requests.sync_q.join()

    def cancel(self):
        if self._exception.received():
            return self._handle_exception()

        self._requests.sync_q.put(None)
        self._exit.notify(False)
        exc, fatal = self._exception.read(block=True)
        assert not fatal and exc is None

    def close_send(self) -> Exception | None:
        if self._exception.received():
            return self._handle_exception()
        self._requests.sync_q.put(None)
        self._exit.notify(True)
        exc, fatal = self._exception.read(block=True)
        if fatal:
            raise exc
        return exc

    def _handle_exception(self) -> Exception | None:
        if self._exception.received():
            exc, fatal = self._exception.read()
            if fatal:
                raise exc
            return exc

    async def run(self):
        try:
            while True:
                async with process(self._requests) as pld:
                    if await self._maybe_exit(pld):
                        return
                    exc = await self._wrapped.send(pld)
                    if exc is not None:
                        self._exception.notify((exc, False))
                        return

        except Exception as e:
            self._exception.notify((e, True))
            raise e

    async def _maybe_exit(self, pld: RQ | None) -> bool:
        if not self._exit.received() or pld is not None:
            return False
        exc: Exception | None = None
        graceful = self._exit.read()
        if graceful:
            exc = await self._wrapped.close_send()
        self._exception.notify((exc, False))
        return True


class Stream(Thread, Generic[RQ, RS]):
    _client: AsyncStreamClient
    _target: str
    _open_exception: Optional[Notification[Optional[Exception]]]
    _receiver: Receiver[RS]
    _sender: SenderCloser[RQ]
    _response_factory: PayloadFactory[RS]
    _request_type: Type[RQ]

    def __init__(
            self,
            client: AsyncStreamClient,
            target: str,
            request_type: Type[RQ],
            response_factory: PayloadFactoryFunc[RS]
    ) -> None:
        super().__init__()
        self._client = client
        self._target = target
        self._response_factory = PayloadFactory[RS](response_factory)
        self._request_type = request_type
        self._open_exception = Notification()
        self.start()
        self._ack_open()

    def run(self) -> None:
        asyncio.run(self._run())

    def received(self) -> bool:
        return self._receiver.received()

    def receive(self) -> tuple[RS | None, Exception | None]:
        res, exc = self._receiver.receive()
        if exc is not None:
            self._sender.cancel()
        return res, exc

    def send(self, pld: RQ) -> Exception | None:
        return self._sender.send(pld)

    def close_send(self) -> Exception | None:
        return self._sender.close_send()

    async def _run(self):
        try:
            self._wrapped = await self._client.stream(self._target,
                                                      self._request_type,
                                                      self._response_factory)
        except Exception as e:
            self._open_exception.notify(e)
            return
        self._receiver = Receiver(self._wrapped)
        self._sender = SenderCloser(self._wrapped)
        self._open_exception.notify(None)
        await asyncio.gather(self._receiver.run(), self._sender.run())

    def _ack_open(self):
        exc = self._open_exception.read(block=True)
        if exc is not None:
            raise exc
        self._open_exception = None


class StreamClient:
    wrapped: AsyncStreamClient

    def __init__(self, wrapped: AsyncStreamClient) -> None:
        self.wrapped = wrapped

    def stream(
            self,
            target: str,
            request_type: Type[RQ],
            response_factory: PayloadFactoryFunc[RS]
    ) -> Stream[RQ, RS]:
        return Stream[RQ, RS](self.wrapped, target, request_type, response_factory)
