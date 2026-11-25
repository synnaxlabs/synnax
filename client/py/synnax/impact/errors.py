#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.impact.types_gen import Plan
from x.exceptions import Error, ExceptionPayload, Provider


class PlanChangedError(Error):
    """Raised by an apply whose plan changed after the caller planned it."""

    TYPE = "sy.impact.plan_changed"
    plan: Plan
    """Is the plan as it is now, as the caller may see it."""

    def __init__(self, plan: Plan) -> None:
        super().__init__("the records that the change touches changed")
        self.plan = plan


def _decode(encoded: ExceptionPayload) -> Exception | None:
    if encoded.type != PlanChangedError.TYPE:
        return None
    return PlanChangedError(Plan.model_validate_json(encoded.data or ""))


def _encode(_: Exception) -> ExceptionPayload | None:
    return None


EXCEPTION_PROVIDER = Provider(encode=_encode, decode=_decode)
"""Decodes a PlanChangedError with its plan. Register it before the Core's provider,
which decodes every other ``sy.`` type."""
