#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Generic, Literal, TypeVar
from uuid import uuid4

from freighter import Payload
from pydantic import Field

from synnax.telem import TimeStamp
from synnax.ontology import ID

SUCCESS_VARIANT = "success"
INFO_VARIANT = "info"
WARNING_VARIANT = "warning"
ERROR_VARIANT = "error"
DISABLED_VARIANT = "disabled"
LOADING_VARIANT = "loading"

Variant = Literal[
    "success",
    "info",
    "warning",
    "error",
    "disabled",
    "loading",
]
"""Represents the variant of a status message."""

D = TypeVar("D", bound=Payload)

STATUS_ONTOLOGY_TYPE = ID(type="status")


def ontology_id(key: str) -> ID:
    """Create an ontology ID for a status.

    Args:
        key: The status key.

    Returns:
        An ontology ID dictionary with type "status" and the given key.
    """
    return ID(type=STATUS_ONTOLOGY_TYPE, key=key)


class Status(Payload, Generic[D]):
    """A standardized payload used across Synnax."""

    key: str = Field(default_factory=lambda: str(uuid4()))
    """A unique key for the status."""
    name: str = ""
    """A human-readable name for the status."""
    variant: Variant
    """The variant of the status."""
    message: str
    """The message of the status."""
    description: str = ""
    """The description of the status."""
    time: TimeStamp = Field(default_factory=TimeStamp.now)
    """The time the status was created."""
    labels: list[Any] | None = None
    """Optional labels attached to the status (only present in responses)."""
    details: D | None = None
    """The details are customizable details for component specific statuses."""

    @property
    def ontology_id(self) -> ID:
        """Get the ontology ID for the status.

        Returns:
            An ontology ID dictionary with type "status" and the status key.
        """
        return ontology_id(self.key)
