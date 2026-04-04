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
from collections.abc import Callable, Mapping


def _resolve(target: str, module_globals: dict[str, object]) -> object:
    """Resolves a dotted name like 'Variant.SUCCESS' against module globals."""
    parts = target.split(".")
    val = module_globals[parts[0]]
    for part in parts[1:]:
        val = getattr(val, part)
    return val


def deprecated_getattr(
    module_name: str,
    deprecated: Mapping[str, str],
    module_globals: dict[str, object],
) -> Callable[[str], object]:
    """Creates a module-level __getattr__ that warns on deprecated name access.

    :param module_name: The module's __name__ (used in AttributeError messages).
    :param deprecated: Mapping of old_name to new_name. new_name supports dotted
        paths (e.g. "Variant.SUCCESS") resolved against module_globals.
    :param module_globals: The module's globals() dict to resolve new names from.
    :returns: A __getattr__ function suitable for assignment at module level.
    """

    def __getattr__(name: str) -> object:
        if name in deprecated:
            target = deprecated[name]
            val = _resolve(target, module_globals)
            warnings.warn(
                f"{name} is deprecated, use {target} instead",
                DeprecationWarning,
                stacklevel=2,
            )
            module_globals[name] = val
            return val
        raise AttributeError(f"module {module_name!r} has no attribute {name!r}")

    return __getattr__
