import asyncio
import contextlib
from threading import Thread
from typing import AsyncIterator, Generic, Optional, Type

from janus import Queue

from .stream import AsyncStreamClient, AsyncStreamReceiver, AsyncStreamSenderCloser
from .transport import RQ, RS, P
from .util.threading import Notification


@contextlib.asynccontextmanager
async def process(queue: Queue) -> AsyncIterator[P]:
    pld = await queue.async_q.get()
    try:
        yield pld
    finally:
        queue.async_q.task_done()


class _Receiver(Generic[RS]):
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


class _SenderCloser(Generic[RQ]):
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
        return None

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
        return None

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


class SyncStream(Thread, Generic[RQ, RS]):
    """An implementation of the Stream protocol that wraps an AsyncStreamClient
    and exposes a synchronous interface.
    """

    _client: AsyncStreamClient
    _target: str
    _open_exception: Optional[Notification[Optional[Exception]]]
    _receiver: _Receiver[RS]
    _sender: _SenderCloser[RQ]
    _response_factory: Type[RS]
    _request_type: Type[RQ]

    def __init__(
        self, client: AsyncStreamClient, target: str, req_t: Type[RQ], res_t: Type[RS]
    ) -> None:
        super().__init__()
        self._client = client
        self._target = target
        self._response_factory = res_t
        self._request_type = req_t
        self._open_exception = Notification()
        self.start()
        self._ack_open()

    def run(self) -> None:
        asyncio.run(self._run())

    def received(self) -> bool:
        """Implement the Stream protocol."""
        return self._receiver.received()

    def receive(self) -> tuple[RS | None, Exception | None]:
        """Implement the Stream protocol."""
        res, exc = self._receiver.receive()
        if exc is not None:
            self._sender.cancel()
        return res, exc

    def send(self, pld: RQ) -> Exception | None:
        """Implement the Stream protocol."""
        return self._sender.send(pld)

    def close_send(self) -> Exception | None:
        """Implement the Stream protocol."""
        return self._sender.close_send()

    async def _run(self):
        try:
            self._wrapped = await self._client.stream(
                self._target, self._request_type, self._response_factory
            )
        except Exception as e:
            self._open_exception.notify(e)
            return
        self._receiver = _Receiver(self._wrapped)
        self._sender = _SenderCloser(self._wrapped)
        self._open_exception.notify(None)
        await asyncio.gather(self._receiver.run(), self._sender.run())

    def _ack_open(self):
        exc = self._open_exception.read(block=True)
        if exc is not None:
            raise exc
        self._open_exception = None


class SyncStreamClient:
    """A synchronous wrapper around an AsyncStreamClient that allows a caller to
    use an AsyncStream synchronously.
    """

    wrapped: AsyncStreamClient

    def __init__(self, wrapped: AsyncStreamClient) -> None:
        self.wrapped = wrapped

    def stream(
        self, target: str, req_t: Type[RQ], res_t: Type[RS]
    ) -> SyncStream[RQ, RS]:
        """Implement the StreamClient protocol."""
        return SyncStream[RQ, RS](self.wrapped, target, req_t, res_t)
