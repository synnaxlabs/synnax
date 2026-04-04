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
    Status,
    Variant,
)
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "VARIANT_SUCCESS": "Variant.SUCCESS",
    "VARIANT_INFO": "Variant.INFO",
    "VARIANT_WARNING": "Variant.WARNING",
    "VARIANT_ERROR": "Variant.ERROR",
    "VARIANT_LOADING": "Variant.LOADING",
    "VARIANT_DISABLED": "Variant.DISABLED",
    "SUCCESS_VARIANT": "Variant.SUCCESS",
    "INFO_VARIANT": "Variant.INFO",
    "WARNING_VARIANT": "Variant.WARNING",
    "ERROR_VARIANT": "Variant.ERROR",
    "DISABLED_VARIANT": "Variant.DISABLED",
    "LOADING_VARIANT": "Variant.LOADING",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "Client",
    "DELETE_CHANNEL",
    "ONTOLOGY_TYPE",
    "SET_CHANNEL",
    "Status",
    "Variant",
    "ontology_id",
]
