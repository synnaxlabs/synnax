#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from pydantic import BaseModel


class ChannelMeta(BaseModel):
    """General channel metadata that can be read from a file."""

    name: str
    """The name of the channel."""
    meta_data: dict[str, Any]
    """Any additional metadata associated with the channel."""
