#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any

import numpy as np
import numpy.typing as npt

# We dynamically define all of these operators to allow for interoperability with numpy
# arrays.
_NP_COMPARISON_OPERATORS = [
    "__eq__",
    "__ne__",
    "__lt__",
    "__gt__",
    "__le__",
    "__ge__",
    "__add__",
    "__sub__",
    "__mul__",
    "__floordiv__",
    "__mod__",
    "__pow__",
    "__lshift__",
    "__rshift__",
    "__and__",
    "__xor__",
    "__or__",
    "__neg__",
    "__pos__",
    "__abs__",
    "__invert__",
    "__round__",
    "__floor__",
    "__ceil__",
    "__trunc__",
    "__int__",
    "__float__",
    "__index__",
    "__truediv__",
    "__divmod__",
    "__radd__",
    "__rsub__",
    "__rmul__",
    "__rfloordiv__",
    "__rmod__",
    "__rpow__",
    "__rlshift__",
    "__rrshift__",
    "__rand__",
    "__rxor__",
    "__ror__",
    "__iadd__",
    "__isub__",
    "__imul__",
    "__ifloordiv__",
    "__imod__",
    "__ipow__",
    "__ilshift__",
    "__irshift__",
    "__iand__",
    "__ixor__",
    "__ior__",
    "__rtruediv__",
    "__rdivmod__",
    "__itruediv__",
    "__ifloordiv__",
    "__matmul__",
    "__rmatmul__",
    "__imatmul__",
]


if TYPE_CHECKING:

    class ArrayOperators:
        """Declares the operators that overload_comparison_operators forwards to the
        numpy array. In-place, rounding, and scalar conversion operators fail at
        runtime on a one-dimensional array, so they are left out.
        """

        # Any, as in numpy's own stubs: object types these as returning bool.
        def __eq__(self, other: object) -> Any: ...
        def __ne__(self, other: object) -> Any: ...
        def __lt__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]: ...
        def __gt__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]: ...
        def __le__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]: ...
        def __ge__(self, other: npt.ArrayLike) -> npt.NDArray[np.bool_]: ...
        def __add__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __sub__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __mul__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __truediv__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __floordiv__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __mod__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __pow__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __lshift__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rshift__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __and__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __xor__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __or__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __divmod__(self, other: npt.ArrayLike) -> tuple[np.ndarray, np.ndarray]: ...
        def __matmul__(self, other: npt.ArrayLike) -> Any: ...
        def __radd__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rsub__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rmul__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rtruediv__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rfloordiv__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rmod__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rpow__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rlshift__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rrshift__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rand__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rxor__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __ror__(self, other: npt.ArrayLike) -> np.ndarray: ...
        def __rdivmod__(
            self, other: npt.ArrayLike
        ) -> tuple[np.ndarray, np.ndarray]: ...
        def __rmatmul__(self, other: npt.ArrayLike) -> Any: ...
        def __neg__(self) -> np.ndarray: ...
        def __pos__(self) -> np.ndarray: ...
        def __abs__(self) -> np.ndarray: ...
        def __invert__(self) -> np.ndarray: ...

else:

    class ArrayOperators:
        pass


def overload_comparison_operators(cls: type[Any], method: str) -> type[Any]:
    return forward_methods(cls, method, _NP_COMPARISON_OPERATORS)


def forward_methods(cls: type[Any], to: str, methods: list[str]) -> type[Any]:
    for method in methods:
        setattr(
            cls,
            method,
            lambda self, *args, method=method: getattr(getattr(self, to)(), method)(
                *args
            ),
        )
    return cls
