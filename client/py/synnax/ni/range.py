#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import warnings
from typing import Any

from pydantic import BaseModel, Field, model_validator

from x import spatial


class MinMaxVal(BaseModel):
    """Bounds the expected signal range in scaled units.

    Attributes:
        range: Is the expected signal range, in scaled units.
    """

    range: spatial.ClosedBounds = Field(
        default_factory=lambda: spatial.ClosedBounds(lower=0, upper=1)
    )

    @model_validator(mode="before")
    @classmethod
    def _lift_min_max_val(cls, data: Any) -> Any:
        """Reads the min_val and max_val keywords that releases up to 0.62 took."""
        if not isinstance(data, dict) or (
            "min_val" not in data and "max_val" not in data
        ):
            return data
        if "range" in data:
            raise TypeError("pass range, or min_val and max_val, but not both")
        warnings.warn(
            "min_val and max_val are deprecated; pass "
            "range=spatial.ClosedBounds(lower=..., upper=...) instead",
            DeprecationWarning,
            stacklevel=3,
        )
        data = dict(data)
        default = cls.model_fields["range"].get_default(call_default_factory=True)
        data["range"] = spatial.ClosedBounds(
            lower=data.pop("min_val", default.lower),
            upper=data.pop("max_val", default.upper),
        )
        return data
