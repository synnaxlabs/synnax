#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import operator
import warnings
from collections.abc import Callable, Mapping
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


def _warn(msg: str) -> None:
    warnings.warn(msg, DeprecationWarning, skip_file_prefixes=_LIBRARY_DIRS)


def _warn_secure() -> None:
    _warn(_SECURE_DEPRECATED_MSG)


def replace_alias(
    source: Mapping[str, Any],
    name: str,
    alias: str,
    *,
    deprecated_msg: str,
    disagree_msg: str,
    convert: Callable[[Any], Any] = lambda v: v,
) -> Mapping[str, Any]:
    """Returns source with its deprecated alias key replaced by name, and the value of
    alias passed through convert. Warns with deprecated_msg as a DeprecationWarning when
    source sets alias without name. A source may set both when they agree, with no
    warning, so that clients older and newer than Synnax 0.63 read the same choice.

    :raises ValidationError: With disagree_msg, if source sets both, and the converted
        value of alias is not the value of name.
    """
    if alias not in source:
        return source
    value = convert(source[alias])
    if name not in source:
        _warn(deprecated_msg)
    elif source[name] != value:
        raise ValidationError(disagree_msg)
    rest = {k: v for k, v in source.items() if k != alias}
    return {**rest, name: value}


def without_secure(source: Mapping[str, Any]) -> Mapping[str, Any]:
    """Returns the options that one source sets, with the deprecated secure replaced by
    insecure. See replace_alias.

    :raises ValidationError: If the source sets both, and secure is not the inverse of
        insecure.
    """
    return replace_alias(
        source,
        "insecure",
        "secure",
        deprecated_msg=_SECURE_DEPRECATED_MSG,
        disagree_msg=_TLS_FLAGS_DISAGREE_MSG,
        convert=operator.not_,
    )
