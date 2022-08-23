import asyncio

from dataclasses import dataclass

from . import Segment
from asyncio import Event, Task
import freighter
from ..util.notification import Notification


@dataclass
class WriterRequest:
    open_keys: list[str]
    segments: list[Segment]


@dataclass
class WriterResponse:
    ack: bool
    error: freighter.errors.ErrorPayload


class BaseWriter:
    keys: list[str]
    transport: freighter.StreamClient
    stream: freighter.Stream[WriterResponse, WriterRequest]
    responses: Task[Exception | None]
    error: Notification[Exception]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport
        self.error = Notification()

    async def open(self, keys: list[str]):
        self.stream = await self.transport.stream("/segment/write")
        await self.stream.send(WriterRequest(keys, []))
        res = WriterResponse(error=freighter.errors.ErrorPayload(None, None))
        # Wait for the server to acknowledge our open request. If we receive
        # an error, it means the request failed and writes should not proceed.
        err = await self.stream.receive(res)
        if err is not None:
            raise err
        self.responses = asyncio.create_task(self.receive_errors())

    async def write(self, segments: list[Segment]) -> bool:
        if self.error.received():
            return False

        err = await self.stream.send(WriterRequest([], segments))
        if err is not None:
            raise err

        return True

    async def close(self):
        await self.stream.close_send()
        err = await self.responses
        if err is not None:
            raise err

    async def receive_errors(self):
        res = WriterResponse(error=freighter.errors.ErrorPayload(None, None))
        err = await self.stream.receive(res)
        if err is None:
            err = freighter.errors.decode(res.error)
        await self.error.notify(err)
