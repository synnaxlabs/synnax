#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings
from collections.abc import Mapping
from pathlib import Path
from typing import Any

from pydantic import BaseModel

from synnax.exceptions import ValidationError
from x import params


class Options(BaseModel):
    """Options class for the Synnax py."""

    host: str
    port: int
    username: str = ""
    password: str = ""
    api_key: str = ""
    insecure: bool = False
    ca_cert: str | None = None


def from_stored(raw: Mapping[str, Any]) -> dict[str, Any]:
    """Returns the options that a config file stores, with insecure in place of the
    secure field that Synnax 0.62 stores. A record with neither field connects without
    TLS.
    """
    stored = {k: v for k, v in raw.items() if k != "secure"}
    stored.setdefault("insecure", not raw.get("secure", False))
    return stored


BOTH_TLS_FLAGS_MSG = (
    "Give insecure or SYNNAX_INSECURE, not the deprecated secure or SYNNAX_SECURE too."
)

SECURE_DEPRECATED_MSG = (
    "secure and SYNNAX_SECURE are deprecated and stop working in Synnax 0.64. Use "
    "insecure and SYNNAX_INSECURE, which mean the inverse."
)

# The warning names the first frame of the caller's code.
_LIBRARY_DIRS = (str(Path(__file__).parent), str(Path(params.__file__).parent))


def without_secure(source: Mapping[str, Any]) -> Mapping[str, Any]:
    """Returns the options that one source sets, with the deprecated secure replaced by
    insecure. Warns with DeprecationWarning when the source sets secure.

    :raises ValidationError: If the source sets both insecure and secure.
    """
    if "secure" not in source:
        return source
    if "insecure" in source:
        raise ValidationError(BOTH_TLS_FLAGS_MSG)
    warnings.warn(
        SECURE_DEPRECATED_MSG, DeprecationWarning, skip_file_prefixes=_LIBRARY_DIRS
    )
    rest = {k: v for k, v in source.items() if k != "secure"}
    return {**rest, "insecure": not source["secure"]}
