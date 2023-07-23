#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel


class SynnaxOptions(BaseModel):
    """Options class for the Synnax py."""

    host: str
    port: int
    username: str = ""
    password: str = ""
    secure: bool = False
