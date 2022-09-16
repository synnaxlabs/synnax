from __future__ import annotations

from dataclasses import dataclass

from synnax import telem
from synnax.errors import UnexpectedError
from synnax.telem.numpy import to_numpy_type

import numpy as np


@dataclass
class Channel:
    key: str
    name: str
    node_id: int
    rate: telem.Rate
    data_type: telem.DataType
    density: telem.Density

    def __init__(
            self,
            name: str = "",
            node_id: int = 0,
            rate: telem.UnparsedRate = telem.Rate(0),
            data_type: telem.UnparsedDataType = telem.DATA_TYPE_UNKNOWN,
            density: telem.UnparsedDensity = telem.Density(0),
            key: str = "",
    ):
        self.name = name
        self.node_id = node_id
        self.rate = telem.Rate(rate)
        self.data_type = telem.DataType(data_type)
        self.density = telem.Density(density)
        self.key = key

    @property
    def numpy_type(self) -> np.ScalarType:
        npt = to_numpy_type(self.data_type)
        if npt is None:
            raise UnexpectedError(
                f"Cannot convert data type {self.data_type} to numpy type"
            )
        return npt
