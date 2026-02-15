#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.status.client import (
    DELETE_CHANNEL,
    ONTOLOGY_TYPE,
    SET_CHANNEL,
    Client,
    ontology_id,
)
from synnax.status.types_gen import (
    VARIANT_DISABLED,
    VARIANT_ERROR,
    VARIANT_INFO,
    VARIANT_LOADING,
    VARIANT_SUCCESS,
    VARIANT_WARNING,
    Status,
    Variant,
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
    "DELETE_CHANNEL",
    "DISABLED_VARIANT",
    "ERROR_VARIANT",
    "INFO_VARIANT",
    "LOADING_VARIANT",
    "ONTOLOGY_TYPE",
    "SET_CHANNEL",
    "Status",
    "SUCCESS_VARIANT",
    "VARIANT_DISABLED",
    "VARIANT_ERROR",
    "VARIANT_INFO",
    "VARIANT_LOADING",
    "VARIANT_SUCCESS",
    "VARIANT_WARNING",
    "Variant",
    "ontology_id",
]
