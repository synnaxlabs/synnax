#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pydantic import BaseModel


class InstrumentationMeta(BaseModel):
    """"
    """
    key: str
    path: str
    service_name: str | None = None

    def sub(self, key: str) -> InstrumentationMeta:
        return InstrumentationMeta(
            key=key,
            path=f"{self.path}.{key}",
            service_name=self.service_name
        )
