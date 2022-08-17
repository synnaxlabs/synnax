import asyncio
import dataclasses

from dataclasses import dataclass

from delta import errors
from . import Segment
from asyncio import Event, Task
import freighter

from .. import errors


@dataclass
class WriterRequest:
    open_keys: list[str]
    segments: list[Segment]


@dataclass
class WriterResponse:
    error: freighter.errors.ErrorPayload


class Writer:
    keys: list[str]
    transport: freighter.StreamClient
    stream: freighter.Stream[WriterResponse, WriterRequest]
    error_loop: Task[None]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport
        self.error = Event()

    async def open(self, keys: list[str]):
        self.stream = await self.transport.stream("/segment/write")
        await self.stream.send(WriterRequest(keys, []))
        self.error_loop = asyncio.create_task(self.listen_for_errors())

    async def write(self, segments: list[Segment]) -> Exception | None:
        if self.error.is_set():
            return freighter.EOF()
        return await self.stream.send(WriterRequest([], segments))

    async def close(self):
        await self.stream.close_send()
        err = await self.error_loop
        if err is not None:
            raise err

    async def check_for_error(self) -> Exception | None:
        if self.error.is_set():
            self.error.clear()
            return await self.error_loop

    async def listen_for_errors(self) -> Exception | None:
        while True:
            res = WriterResponse(error=freighter.errors.ErrorPayload(None, None))
            err = await self.stream.receive(res)
            self.error.set()

            if isinstance(err, freighter.errors.EOF):
                return None

            if err is not None:
                raise err

            return freighter.errors.decode(res.error)
