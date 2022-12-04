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
