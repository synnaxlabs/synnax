#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from threading import Event
from typing import Generic, TypeVar

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
