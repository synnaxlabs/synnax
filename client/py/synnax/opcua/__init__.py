#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Deprecated: use synnax.opc instead."""

import warnings
from typing import Any

warnings.warn(
    "synnax.opcua is deprecated and will be removed in a future version. "
    "Use synnax.opc instead.",
    FutureWarning,
    stacklevel=2,
)

from synnax import opc as _opc
from synnax.opc import *  # noqa: E402, F403
from synnax.opc import __all__ as __all__  # noqa: F401


def __getattr__(name: str) -> Any:
    """Resolves names synnax.opc only serves through its deprecation shim."""
    return getattr(_opc, name)
