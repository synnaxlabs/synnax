#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence
from os import PathLike
from types import TracebackType
from typing import Self

import numpy as np
from numpy.typing import ArrayLike, NDArray

_Path = str | PathLike[str]

class TdmsChannel:
    @property
    def name(self) -> str: ...
    def __len__(self) -> int: ...
    def __getitem__(self, index: slice) -> NDArray[np.generic]: ...

class TdmsGroup:
    @property
    def name(self) -> str: ...
    def channels(self) -> list[TdmsChannel]: ...

class TdmsFile:
    @staticmethod
    def open(file: _Path) -> TdmsFile: ...
    def groups(self) -> list[TdmsGroup]: ...
    def __enter__(self) -> Self: ...
    def __exit__(
        self,
        exc_type: type[BaseException] | None,
        exc_value: BaseException | None,
        traceback: TracebackType | None,
    ) -> None: ...

class TdmsObject: ...

class RootObject(TdmsObject):
    def __init__(self) -> None: ...

class GroupObject(TdmsObject):
    def __init__(self, group: str) -> None: ...

class ChannelObject(TdmsObject):
    def __init__(self, group: str, channel: str, data: ArrayLike) -> None: ...

class TdmsWriter:
    def __init__(self, file: _Path) -> None: ...
    def write_segment(self, objects: Sequence[TdmsObject]) -> None: ...
    def __enter__(self) -> Self: ...
    def __exit__(
        self,
        exc_type: type[BaseException] | None,
        exc_value: BaseException | None,
        traceback: TracebackType | None,
    ) -> None: ...
