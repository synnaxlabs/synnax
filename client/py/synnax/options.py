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

import pydantic
from pydantic import BaseModel, model_validator

from synnax.exceptions import ValidationError
from x import params


class Options(BaseModel):
    """Options class for the Synnax py. Takes the deprecated secure as the inverse of
    insecure, and raises ValidationError when the two disagree.
    """

    host: str
    port: int
    username: str = ""
    password: str = ""
    api_key: str = ""
    insecure: bool = False
    ca_cert: str | None = None

    @model_validator(mode="before")
    @classmethod
    def _secure_alias(cls, data: Any) -> Any:
        return without_secure(data) if isinstance(data, Mapping) else data

    @property
    def secure(self) -> bool:
        """Deprecated: the inverse of insecure. It stops working in Synnax 0.64."""
        _warn_secure()
        return not self.insecure


def from_stored(raw: Mapping[str, Any]) -> dict[str, Any]:
    """Returns the options that a config file stores, with insecure in place of the
    secure field that Synnax 0.62 stores. A record with neither field connects without
    TLS.
    """
    stored = {k: v for k, v in raw.items() if k != "secure"}
    stored.setdefault("insecure", not raw.get("secure", False))
    return stored


_TLS_FLAGS_DISAGREE_MSG = (
    "insecure and the deprecated secure disagree. secure must be the inverse of "
    "insecure. Give only insecure."
)

_SECURE_DEPRECATED_MSG = (
    "secure and SYNNAX_SECURE are deprecated and stop working in Synnax 0.64. Use "
    "insecure and SYNNAX_CLIENT_INSECURE, which mean the inverse."
)

# The warning names the first frame of the caller's code.
_LIBRARY_DIRS = tuple(
    str(Path(f).parent) for f in (__file__, pydantic.__file__, params.__file__) if f
)


def warn_deprecated(msg: str) -> None:
    """Warns with msg as a DeprecationWarning that names the first frame outside the
    client.
    """
    warnings.warn(msg, DeprecationWarning, skip_file_prefixes=_LIBRARY_DIRS)


def _warn_secure() -> None:
    warn_deprecated(_SECURE_DEPRECATED_MSG)


def without_secure(source: Mapping[str, Any]) -> Mapping[str, Any]:
    """Returns the options that one source sets, with the deprecated secure replaced by
    insecure. Warns with a DeprecationWarning when source sets secure without insecure.
    When source sets both, they must agree, so that clients older and newer than Synnax
    0.63 read the same choice.

    :raises ValidationError: If the source sets both, and secure is not the inverse of
        insecure.
    """
    if "secure" not in source:
        return source
    rest = {k: v for k, v in source.items() if k != "secure"}
    insecure = not source["secure"]
    if "insecure" not in source:
        warn_deprecated(_SECURE_DEPRECATED_MSG)
    elif source["insecure"] != insecure:
        raise ValidationError(_TLS_FLAGS_DISAGREE_MSG)
    return {**rest, "insecure": insecure}
