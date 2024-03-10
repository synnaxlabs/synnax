#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


# We dynamically define all of these operators to allow for interoperability with
# numpy arrays.
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
    # "__getitem__",
    # "__len__",
]


def overload_comparison_operators(cls, method: str):
    return forward_methods(cls, method, _NP_COMPARISON_OPERATORS)


def forward_methods(cls, to: str, methods: list[str]):
    for method in methods:
        setattr(
            cls,
            method,
            lambda self, *args, method=method: getattr(getattr(self, to)(), method)(
                *args
            ),
        )
    return cls
