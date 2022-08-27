from typing import Generic
from asyncio import Event, Lock
from typing import TypeVar

T = TypeVar('T')


class Notification(Generic[T]):
    event: Event
    lock: Lock
    value: T | None

    def __init__(self):
        self.value = None
        self.event = Event()
        self.lock = Lock()

    async def notify(self, value: T):
        if self.received() or self.lock.locked():
            return
        async with self.lock:
            self.value = value
            self.event.set()

    def received(self) -> bool:
        return self.event.is_set()

    async def read(self) -> T:
        await self.event.wait()
        async with self.lock.acquire():
            return self.value
