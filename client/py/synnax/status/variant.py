#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

SUCCESS_VARIANT = "success"
INFO_VARIANT = "info"
WARNING_VARIANT = "warning"
ERROR_VARIANT = "error"

Variant = Literal["success", "info", "warning", "error"]
"""Represents the variant of a status message."""
