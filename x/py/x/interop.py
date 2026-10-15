#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from typing import Any

import numpy as np
import numpy.typing as npt


class ArrayOperators:
    """A mixin that gives a class the operators of the NumPy array returned by its
    __array__ method. Operators return NumPy values, not instances of the class. A
    Pydantic model must list this mixin before BaseModel so that __eq__ and __ne__
    compare element-wise.
    """

    @abstractmethod
    def __array__(
        self, dtype: np.dtype | None = None, copy: bool | None = None
    ) -> np.ndarray: ...

    # Any matches NumPy's stubs, as object.__eq__ requires a bool return.
    def __eq__(self, other: object) -> Any:
        return self.__array__().__eq__(other)

    def __ne__(self, other: object) -> Any:
        return self.__array__().__ne__(other)

    def __lt__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]:
        return self.__array__().__lt__(other)

    def __gt__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]:
        return self.__array__().__gt__(other)

    def __le__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]:
        return self.__array__().__le__(other)

    def __ge__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]:
        return self.__array__().__ge__(other)

    def __add__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__add__(other)

    def __radd__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__radd__(other)

    def __sub__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__sub__(other)

    def __rsub__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rsub__(other)

    def __mul__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__mul__(other)

    def __rmul__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rmul__(other)

    def __truediv__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__truediv__(other)

    def __rtruediv__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rtruediv__(other)

    def __floordiv__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__floordiv__(other)

    def __rfloordiv__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rfloordiv__(other)

    def __mod__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__mod__(other)

    def __rmod__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rmod__(other)

    def __pow__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__pow__(other)

    def __rpow__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rpow__(other)

    def __lshift__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__lshift__(other)

    def __rlshift__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rlshift__(other)

    def __rshift__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rshift__(other)

    def __rrshift__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rrshift__(other)

    def __and__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__and__(other)

    def __rand__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rand__(other)

    def __xor__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__xor__(other)

    def __rxor__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__rxor__(other)

    def __or__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__or__(other)

    def __ror__(self, other: npt.ArrayLike) -> np.ndarray:
        return self.__array__().__ror__(other)

    def __divmod__(self, other: npt.ArrayLike) -> tuple[np.ndarray, np.ndarray]:
        return np.divmod(self.__array__(), other)

    def __rdivmod__(self, other: npt.ArrayLike) -> tuple[np.ndarray, np.ndarray]:
        return np.divmod(other, self.__array__())

    # 1-D operands give a NumPy scalar, so the result is not always an array.
    def __matmul__(self, other: npt.ArrayLike) -> Any:
        return self.__array__().__matmul__(other)

    def __rmatmul__(self, other: npt.ArrayLike) -> Any:
        return self.__array__().__rmatmul__(other)

    def __neg__(self) -> np.ndarray:
        return self.__array__().__neg__()

    def __pos__(self) -> np.ndarray:
        return self.__array__().__pos__()

    def __abs__(self) -> np.ndarray:
        return self.__array__().__abs__()

    def __invert__(self) -> np.ndarray:
        return self.__array__().__invert__()
