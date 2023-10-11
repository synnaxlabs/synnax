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
    """The metadata for instrumentation."""

    key: str
    """The key of the instrumentation. ex. "api"""
    path: str
    """The path of the instrumentation. "api.v1.users"""
    service_name: str | None = None
    """An optional service name eg. "synnax"""

    def child_(self, key: str) -> InstrumentationMeta:
        return InstrumentationMeta(
            key=key, path=self.extend_path_(key), service_name=self.service_name
        )

    def extend_path_(self, key: str) -> str:
        return f"{self.path}.{key}"
