#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from opentelemetry.sdk.resources import Attributes as Attributes
from opentelemetry.sdk.resources import Resource

def configure_opentelemetry(
    dsn: str = "",
    service_name: str | None = "",
    service_version: str | None = "",
    deployment_environment: str | None = "",
    resource_attributes: Attributes | None = None,
    resource: Resource | None = None,
    logging_level: int = ...,
) -> None: ...
