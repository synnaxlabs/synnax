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
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "SUCCESS_VARIANT": "VARIANT_SUCCESS",
    "INFO_VARIANT": "VARIANT_INFO",
    "WARNING_VARIANT": "VARIANT_WARNING",
    "ERROR_VARIANT": "VARIANT_ERROR",
    "DISABLED_VARIANT": "VARIANT_DISABLED",
    "LOADING_VARIANT": "VARIANT_LOADING",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "Client",
    "DELETE_CHANNEL",
    "ONTOLOGY_TYPE",
    "SET_CHANNEL",
    "Status",
    "VARIANT_DISABLED",
    "VARIANT_ERROR",
    "VARIANT_INFO",
    "VARIANT_LOADING",
    "VARIANT_SUCCESS",
    "ontology_id",
    "Variant"
]
