import asyncio

from dataclasses import dataclass
from typing import Generic, TypeVar

from . import Segment
from asyncio import Event, Task
import freighter


@dataclass
class WriterRequest:
    open_keys: list[str]
    segments: list[Segment]


@dataclass
class WriterResponse:
    error: freighter.errors.ErrorPayload


T = TypeVar('T')


class Notifier(Generic[T]):
    event: asyncio.Event
    lock: asyncio.Lock
    value: T

    async def maybe_notify(self, value: T):
        if not self.is_set() and not self.lock.locked():
            await self.notify(value)

    async def notify(self, value: T):
        async with self.lock.acquire():
            self.value = value
            self.event.set()
            self.lock.release()

    def is_set(self) -> bool:
        return self.event.is_set()


class BaseWriter:
    keys: list[str]
    transport: freighter.StreamClient
    stream: freighter.Stream[WriterResponse, WriterRequest]
    responses: Task[Exception | None]
    error_notifier: Notifier[Exception]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport
        self.error = Event()

    async def open(self, keys: list[str]):
        self.stream = await self.transport.stream("/segment/write")
        await self.stream.send(WriterRequest(keys, []))
        self.responses = asyncio.create_task(self.listen_for_errors())

    async def write(self, segments: list[Segment]) -> bool:
        if self.error_notifier.is_set():
            return False
        err = await self.stream.send(WriterRequest([], segments))
        if err is not None:
            await self.error_notifier.maybe_notify(err)
            return False
        return True

    async def close(self):
        await self.stream.close_send()
        err = await self.responses
        if err is not None:
            raise err

    async def listen_for_errors(self) -> Exception | None:
        res = WriterResponse(error=freighter.errors.ErrorPayload(None, None))
        err = await self.stream.receive(res)
        self.error.set()

        if isinstance(err, freighter.errors.EOF):
            return None

        if err is not None:
            raise err

        return freighter.errors.decode(res.error)
