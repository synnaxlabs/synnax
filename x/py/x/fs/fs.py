#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import os
import pathlib
import tempfile
from collections.abc import Iterable
from typing import TypeAlias

FilePath: TypeAlias = str | os.PathLike[str]
"""A filesystem path.

Equivalent to ``str | os.PathLike[str]`` — the same shape ``open()`` accepts for paths.
"""


def stream_to_file(
    chunks: Iterable[bytes], dest: FilePath, *, allow_empty: bool = True
) -> None:
    """Writes a stream of byte chunks into dest atomically.

    The chunks are streamed into a temporary file alongside dest and the temp file is
    renamed into place only once the full stream is consumed, so dest is observed to
    hold either its previous contents or the complete new contents — never a partial
    write. The temp file is flushed and fsynced before the rename so its contents are
    durable on disk before they become visible at dest. A failure partway through
    removes the temp file and leaves any existing dest untouched.

    :param chunks: an iterable of byte chunks to write in order.
    :param dest: the destination file path.
    :param allow_empty: when False, a stream that yields no bytes raises ValueError and
        leaves any existing dest untouched, instead of replacing it with a zero-byte
        file. Defaults to True.
    :raises ValueError: if the stream is empty and allow_empty is False.
    """
    dest_path = pathlib.Path(os.fspath(dest))
    fd, tmp_name = tempfile.mkstemp(dir=dest_path.parent, suffix=".part")
    try:
        written = 0
        with os.fdopen(fd, "wb") as out:
            for chunk in chunks:
                out.write(chunk)
                written += len(chunk)
            out.flush()
            os.fsync(out.fileno())
        if not allow_empty and written == 0:
            raise ValueError(
                f"refusing to write an empty stream to {os.fspath(dest)!r}"
            )
        os.replace(tmp_name, dest_path)
    except BaseException:
        os.unlink(tmp_name)
        raise
