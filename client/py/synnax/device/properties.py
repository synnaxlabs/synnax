#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import JsonValue


def ensure_object(props: dict[str, JsonValue], *path: str) -> dict[str, JsonValue]:
    """Returns the JSON object at path in props, and adds an empty object for each
    missing key on the path.

    :param props: The device properties to walk.
    :param path: The keys to follow from props.
    :returns: The object at the end of the path, which the caller can change in place.
    :raises TypeError: If a value on the path is not an object.
    """
    obj = props
    for key in path:
        child = obj.setdefault(key, {})
        if not isinstance(child, dict):
            raise TypeError(
                f"device property {key!r} is a {type(child).__name__}, not an object"
            )
        obj = child
    return obj
