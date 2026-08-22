#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Iterator
from contextlib import closing

from pydantic import BaseModel

import synnax.channel.payload as channel
from freighter import FileTransport
from synnax.channel.retrieve import Retriever
from synnax.exceptions import UnexpectedError
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.frame import Frame
from synnax.telem import TimeRange
from x.exceptions import ExceptionPayload, decode_exception

_ENDPOINT = "/frame/read"
_FRAME_CONTENT_TYPE = "application/vnd.synnax.frame"
_HEADER_SIZE = 5
_TERMINATOR_KIND = 1
_TRUNCATED = "The Core cut the read short before it finished."


class _Request(BaseModel):
    keys: list[channel.Key]
    bounds: TimeRange
    downsample_factor: int


class Reader:
    """Reader pulls historical telemetry from the Core in a single request per read."""

    _retriever: Retriever
    _file: FileTransport

    def __init__(self, retriever: Retriever, file_transport: FileTransport) -> None:
        self._retriever = retriever
        self._file = file_transport

    def read(
        self,
        tr: TimeRange,
        channels: channel.Params,
        downsample_factor: int = 1,
    ) -> Frame:
        """Reads every sample the given channels hold within tr.

        :param tr: the time range to read.
        :param channels: the channels to read, by key or by name. A frame read by name
            is keyed by name.
        :param downsample_factor: keeps one sample in every downsample_factor. A factor
            of 1, the default, keeps every sample.
        :returns: the samples as a frame.
        """
        adapter = ReadFrameAdapter(self._retriever)
        adapter.update(channels)
        req = _Request(
            keys=adapter.keys, bounds=tr, downsample_factor=downsample_factor
        )
        frame = Frame()
        with closing(self._file.stream(_ENDPOINT, req, _FRAME_CONTENT_TYPE)) as body:
            for record in _records(body):
                pld = adapter.codec.decode(record)
                frame.append(Frame(channels=pld.keys, series=pld.series))
        return adapter.adapt(frame)


def _records(body: Iterator[bytes]) -> Iterator[bytes]:
    """Yields the payload of every data record in a frame-encoded read body.

    The terminator ends the iteration, raising the error it carries when the read failed
    partway. A body that ends without one was cut short in transit.
    """
    buf = bytearray()

    def fill(size: int) -> bool:
        while len(buf) < size:
            chunk = next(body, None)
            if chunk is None:
                return False
            buf.extend(chunk)
        return True

    while True:
        if not fill(_HEADER_SIZE):
            raise UnexpectedError(_TRUNCATED)
        kind = buf[0]
        size = int.from_bytes(buf[1:_HEADER_SIZE], "little")
        if not fill(_HEADER_SIZE + size):
            raise UnexpectedError(_TRUNCATED)
        payload = bytes(buf[_HEADER_SIZE : _HEADER_SIZE + size])
        del buf[: _HEADER_SIZE + size]
        if kind != _TERMINATOR_KIND:
            yield payload
            continue
        if size == 0:
            return
        raise decode_exception(
            ExceptionPayload.model_validate_json(payload)
        ) or UnexpectedError(_TRUNCATED)
