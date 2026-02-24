#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import warnings
from collections.abc import Mapping
from typing import Any


def deprecated_getattr(
    module_name: str,
    deprecated: Mapping[str, str | tuple[str, str]],
    module_globals: dict[str, Any],
) -> Any:
    """Creates a module-level __getattr__ that warns on deprecated name access.

    :param module_name: The module's __name__ (used in AttributeError messages).
    :param deprecated: Mapping of old_name to either new_name (str) or a tuple of
        (display_name, globals_key) for cases where the display name in the warning
        message differs from the key used to look up the value in module globals.
    :param module_globals: The module's globals() dict to resolve new names from.
    :returns: A __getattr__ function suitable for assignment at module level.
    """

    def __getattr__(name: str) -> Any:
        if name in deprecated:
            entry = deprecated[name]
            if isinstance(entry, tuple):
                display_name, globals_key = entry
            else:
                display_name = globals_key = entry
            warnings.warn(
                f"{name} is deprecated, use {display_name} instead",
                DeprecationWarning,
                stacklevel=2,
            )
            val = module_globals[globals_key]
            module_globals[name] = val
            return val
        raise AttributeError(f"module {module_name!r} has no attribute {name!r}")

    return __getattr__
