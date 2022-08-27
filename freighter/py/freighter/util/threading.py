from functools import wraps
from threading import (
    Lock,
    Event,
)
from typing import (
    Generic,
    TypeVar, Optional,
)


class Gate:
    _lock: Lock
    _event: Event

    def __init__(self):
        self._lock = Lock()
        self._event = Event()

    def try_close(self) -> bool:
        self._lock.acquire()
        _set = self._event.is_set()
        if not _set:
            self._event.set()
        self._lock.release()
        return not _set

    def is_closed(self) -> bool:
        return self._event.is_set()

    def open(self) -> None:
        self._lock.acquire()
        self._event.clear()
        self._lock.release()

    def wait_open(self) -> None:
        self._event.wait()


T = TypeVar("T")


class Notification(Generic[T]):
    gate: Gate
    _value: Optional[T]

    def __init__(self):
        self._value = None
        self.gate = Gate()

    def notify(self, value: T):
        if self.gate.try_close():
            self._value = value

    def received(self) -> bool:
        return self.gate.is_closed()

    def read(self) -> Optional[T]:
        self.gate.wait_open()
        return self._value

    def clear(self) -> None:
        if not self.gate.is_closed():
            raise Exception(
                "attempted to clear a notification that has not been received"
            )
        self._value = None
        self.gate.open()