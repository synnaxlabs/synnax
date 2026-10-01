#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.exceptions import ValidationError

# MAX_DOWNSAMPLE_FACTOR is the largest factor the Core accepts. The wire field is an
# unsigned 32-bit integer, so an out-of-range value would be reinterpreted rather than
# rejected by the server.
MAX_DOWNSAMPLE_FACTOR = 2**32 - 1


def validate_downsample_factor(factor: int) -> None:
    """Raises ValidationError if factor is outside the range the Core accepts."""
    if factor < 0 or factor > MAX_DOWNSAMPLE_FACTOR:
        raise ValidationError(
            f"downsample_factor must be between 0 and {MAX_DOWNSAMPLE_FACTOR}, "
            f"got {factor}"
        )
