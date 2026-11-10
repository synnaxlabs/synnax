#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Strips the comments from Go and Python source."""

import io
import re
import tokenize

SUFFIXES = (".go", ".py")
# Go has no regex literals, so one regex lexes it exactly.
_GO_TOKEN = re.compile(
    r"//[^\n]*|/\*.*?\*/|\"(?:\\.|[^\"\\\n])*\"|'(?:\\.|[^'\\\n])*'|`[^`]*`", re.S
)
# Comments the toolchain reads.
_GO_DIRECTIVE = re.compile(r"//(?:go:|line |export |\s*\+build)|/\*line ")
# Left between tokens, these mean the source did not lex.
_GO_LEFTOVER = re.compile(r"[\"'`]|/\*")
_GO_NEWLINE = re.compile(r"[ \t\r\f]*\n\s*")
_GO_SPACE = re.compile(r"[ \t\r\f]+")


def _squeeze(code: str) -> str:
    """Collapses whitespace but keeps one newline, where Go inserts a semicolon."""
    return _GO_SPACE.sub(" ", _GO_NEWLINE.sub("\n", code))


def go(text: str) -> str | None:
    """Returns Go source without comments, or None for cgo or a lex error."""
    if 'import "C"' in text:
        return None
    out: list[str] = []
    code: list[str] = []
    pos = 0
    for m in _GO_TOKEN.finditer(text):
        gap = text[pos : m.start()]
        if _GO_LEFTOVER.search(gap):
            return None
        code.append(gap)
        token = m.group()
        if token[0] == "/" and not _GO_DIRECTIVE.match(token):
            code.append("\n" if "\n" in token else " ")
        else:
            out.append(_squeeze("".join(code)))
            out.append(token)
            code = []
        pos = m.end()
    tail = text[pos:]
    if _GO_LEFTOVER.search(tail):
        return None
    code.append(tail)
    out.append(_squeeze("".join(code)))
    return "".join(out).strip()


def python(data: bytes) -> str | None:
    """Returns Python tokens without comments, or None for a tokenize error."""
    skip = {tokenize.COMMENT, tokenize.NL}
    try:
        tokens = tokenize.tokenize(io.BytesIO(data).readline)
        return "\n".join(f"{t.type} {t.string}" for t in tokens if t.type not in skip)
    except (tokenize.TokenError, SyntaxError, UnicodeDecodeError):
        return None


def strip(path: str, data: bytes) -> str | None:
    """Returns the Go or Python source at path without comments, else None."""
    if path.endswith(".go"):
        try:
            return go(data.decode())
        except UnicodeDecodeError:
            return None
    if path.endswith(".py"):
        return python(data)
    return None
