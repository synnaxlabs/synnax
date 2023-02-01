#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from .protocol import Matcher


def new_extension_matcher(extensions: list[str]) -> type[Matcher]:
    """Return a matcher that matches file extensions.

    :returns: a matcher that matches file extensions.
    """
    _e = extensions

    class ExtensionMatcher:
        _extensions = _e

        @classmethod
        def match(cls, path: Path) -> bool:
            return path.suffix[1:] in cls.extensions()

        @classmethod
        def extensions(cls) -> list[str]:
            return cls._extensions

    return ExtensionMatcher
