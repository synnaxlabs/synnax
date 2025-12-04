#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Deprecated: Use synnax.device instead."""

import warnings

warnings.warn(
    "synnax.hardware.device is deprecated and will be removed in a future version. "
    "Use synnax.device instead.",
    FutureWarning,
    stacklevel=2,
)

from synnax.device import *
from synnax.device import __all__
