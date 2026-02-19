#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.status.client import Client
from synnax.status.payload import (
    VARIANT_DISABLED,
    VARIANT_ERROR,
    VARIANT_INFO,
    VARIANT_LOADING,
    VARIANT_SUCCESS,
    VARIANT_WARNING,
    Status,
    Variant,
    ontology_id,
)

# Backwards compatibility
SUCCESS_VARIANT = VARIANT_SUCCESS
INFO_VARIANT = VARIANT_INFO
WARNING_VARIANT = VARIANT_WARNING
ERROR_VARIANT = VARIANT_ERROR
DISABLED_VARIANT = VARIANT_DISABLED
LOADING_VARIANT = VARIANT_LOADING

__all__ = [
    "Client",
    "Variant",
    "Status",
    "VARIANT_SUCCESS",
    "VARIANT_INFO",
    "VARIANT_WARNING",
    "VARIANT_ERROR",
    "VARIANT_DISABLED",
    "VARIANT_LOADING",
    "SUCCESS_VARIANT",
    "INFO_VARIANT",
    "WARNING_VARIANT",
    "ERROR_VARIANT",
    "DISABLED_VARIANT",
    "LOADING_VARIANT",
    "ontology_id",
]
