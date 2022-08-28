import numpy as np
from typing import Union, Type
import numpy.typing as npt

from . import *

NUMPY_TYPES: dict[str, np.ScalarType] = {
    FLOAT64.key: np.float64,
    FLOAT32.key: np.float32,
    INT64.key: np.int64,
    INT32.key: np.int32,
    INT16.key: np.int16,
    INT8.key: np.int8,
    UINT64.key: np.uint64,
    UINT32.key: np.uint32,
    UINT16.key: np.uint16,
    UINT8.key: np.uint8,
}


def to_numpy_type(data_type: DataType) -> np.ScalarType:
    return NUMPY_TYPES.get(data_type.key, None)
