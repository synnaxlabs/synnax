#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Deprecated: Use synnax.device, synnax.rack, synnax.task instead."""

import warnings

warnings.warn(
    "synnax.hardware is deprecated and will be removed in a future version. "
    "Import directly from synnax instead (e.g., 'from synnax import ni', "
    "'from synnax.device import Device').",
    FutureWarning,
    stacklevel=2,
)
