#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Deprecated: Use synnax.labjack instead."""

import warnings

warnings.warn(
    "synnax.hardware.labjack is deprecated and will be removed in a future version. "
    "Use synnax.labjack instead.",
    FutureWarning,
    stacklevel=2,
)

from synnax.labjack import *
from synnax.labjack import __all__
