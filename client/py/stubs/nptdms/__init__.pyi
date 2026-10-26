#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os

import numpy as np
from nptdms.types import TdmsType

class TdmsChannel:
    data_type: type[TdmsType]
    @property
    def name(self) -> str: ...
    def __len__(self) -> int: ...
    def __getitem__(self, index: slice) -> np.ndarray: ...

class TdmsGroup:
    @property
    def name(self) -> str: ...
    def channels(self) -> list[TdmsChannel]: ...

class TdmsFile:
    @staticmethod
    def read(
        file: str | os.PathLike[str], raw_timestamps: bool = False
    ) -> TdmsFile: ...
    @property
    def tdms_version(self) -> int: ...
    def groups(self) -> list[TdmsGroup]: ...

__all__ = ["TdmsChannel", "TdmsFile", "TdmsGroup"]
