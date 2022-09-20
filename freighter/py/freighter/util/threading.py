from threading import Event
from typing import Generic, Optional, TypeVar

T = TypeVar("T")


class Notification(Generic[T]):
    _event: Event
    value: T | None

    def __init__(self):
        self.value = None
        self._event = Event()

    def notify(self, value: T):
        self._event.set()
        self.value = value

    def received(self) -> bool:
        return self._event.is_set()

    def read(self, block: bool = False) -> T | None:
        if block:
            self._event.wait()
        return self.value

    def clear(self) -> None:
        self._event.clear()
        self.value = None
