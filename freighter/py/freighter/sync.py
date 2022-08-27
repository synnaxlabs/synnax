import asyncio
from asyncio import Event
from threading import Thread

from .transport import RS, RQ, PayloadFactoryFunc, PayloadFactory
from typing import Generic, Optional, ClassVar
from janus import Queue
from freighter.stream import (
    AsyncStreamReceiver,
    AsyncStreamClient,
    AsyncStreamSenderCloser,
)
from freighter.util.threading import Notification


class Receiver(Generic[RS]):
    _wrapped: AsyncStreamReceiver[RS]
    _responses: Queue[Optional[RS]]
    _exception: Notification[Optional[Exception]]

    def __init__(self, wrapped: AsyncStreamReceiver[RS]):
        self._responses = Queue(maxsize=1)
        self._exception = Notification()
        self._wrapped = wrapped

    def received(self) -> bool:
        return self._responses.sync_q.qsize() > 0

    def receive(self) -> tuple[RS | None, Exception | None]:
        if self._exception.received():
            return None, self._exception.read()
        pld = self._responses.sync_q.get()
        if pld is None:
            return None, self._exception.read()
        return pld, None

    async def run(self):
        try:
            while True:
                pld, exc = await self._wrapped.receive()
                if exc is not None:
                    self._exception.notify(exc)
                    await self._responses.async_q.put(None)
                    return
                await self._responses.async_q.put(pld)
        except Exception as e:
            self._exception.notify(e)


class SenderCloser(Generic[RQ]):
    _wrapped: AsyncStreamSenderCloser[RQ]
    _requests: Queue[Optional[RQ]]
    _exception: Notification[Optional[Exception]]
    _close_send: Event
    _cancel: Event

    def __init__(self, wrapped: AsyncStreamSenderCloser[RQ]):
        self._wrapped = wrapped
        self._requests = Queue()
        self._exception = Notification()
        self._close_send = Event()
        self._cancel = Event()

    def send(self, pld: RQ) -> Exception | None:
        if self._exception.received():
            return self._exception.read()
        self._requests.sync_q.put(pld)
        return None

    def close_send(self):
        self._requests.sync_q.put(None, block=False)
        self._close_send.set()
        return self._exception.read()

    def cancel(self):
        self._cancel.set()

    async def run(self):
        try:
            while True:
                payload = await self._requests.async_q.get()
                if self._close_send.is_set() and payload is None:
                    break
                if self._cancel.is_set() and payload is None:
                    return
                exc = await self._wrapped.send(payload)
                if exc is not None:
                    self._exception.notify(exc)
                    return
            self._exception.notify(await self._wrapped.close_send())
        except Exception as e:
            self._exception.notify(e)


class Stream(Thread, Generic[RQ, RS]):
    _client: AsyncStreamClient
    _target: str
    _open_exception: Optional[Notification[Optional[Exception]]]
    _receiver: Receiver[RS]
    _sender: SenderCloser[RQ]
    _response_factory: PayloadFactory[RS]

    def __init__(
            self,
            client: AsyncStreamClient,
            target: str,
            response_factory: PayloadFactoryFunc[RS]
    ) -> None:
        super().__init__()
        self._client = client
        self._target = target
        self._response_factory = PayloadFactory[RS](response_factory)
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
            self._sender.send(None)
        return res, exc

    def send(self, pld: RQ) -> Exception | None:
        return self._sender.send(pld)

    def close_send(self) -> Exception | None:
        return self._sender.close_send()

    async def _run(self):
        try:
            self._wrapped = await self._client.stream(self._target,
                                                      self._response_factory)
        except Exception as e:
            self._open_exception.notify(e)
            return
        self._receiver = Receiver(self._wrapped)
        self._sender = SenderCloser(self._wrapped)
        self._open_exception.notify(None)
        try:
            await asyncio.gather(self._receiver.run(), self._sender.run())
        except Exception as e:
            raise e

    def _ack_open(self):
        exc = self._open_exception.read()
        if exc is not None:
            raise exc
        self._open_exception = None


class StreamClient(Generic[RQ, RS]):
    wrapped: AsyncStreamClient

    def __init__(self, wrapped: AsyncStreamClient) -> None:
        self.wrapped = wrapped

    def stream(
            self,
            target: str,
            response_factory: PayloadFactoryFunc[RS]
    ) -> Stream[RQ, RS]:
        return Stream[RQ, RS](self.wrapped, target, response_factory)
