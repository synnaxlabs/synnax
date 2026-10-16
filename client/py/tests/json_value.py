#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import JsonValue


def _walk(value: JsonValue, path: tuple[str, ...]) -> JsonValue:
    for key in path:
        value = obj(value)[key]
    return value


def obj(value: JsonValue, *path: str) -> dict[str, JsonValue]:
    """Returns the JSON object at path within value.

    :param value: The JSON value to read from.
    :param path: Object keys to follow from value.
    :returns: The JSON object at path.
    :raises TypeError: If the value at path, or one on the way to it, is not an object.
    :raises KeyError: If a key in path does not exist.
    """
    value = _walk(value, path)
    if not isinstance(value, dict):
        raise TypeError(f"expected a JSON object, got {value!r}")
    return value


def arr(value: JsonValue, *path: str) -> list[JsonValue]:
    """Returns the JSON array at path within value.

    :param value: The JSON value to read from.
    :param path: Object keys to follow from value.
    :returns: The JSON array at path.
    :raises TypeError: If the value at path is not an array, or one on the way to it is
        not an object.
    :raises KeyError: If a key in path does not exist.
    """
    value = _walk(value, path)
    if not isinstance(value, list):
        raise TypeError(f"expected a JSON array, got {value!r}")
    return value
