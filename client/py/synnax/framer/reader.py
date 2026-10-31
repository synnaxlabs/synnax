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

from pydantic import BaseModel

import synnax.channel.payload as channel
from freighter import FileTransport
from synnax.channel.retrieve import Retriever
from synnax.exceptions import UnexpectedError
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.frame import Frame
from synnax.telem import Reduction, StrideReduction, TimeRange
from x.exceptions import ExceptionPayload, Registry
from x.fs import FilePath

_ENDPOINT = "/frame/read"
_FRAME_CONTENT_TYPE = "application/vnd.synnax.frame"
_HEADER_SIZE = 5
_TERMINATOR_KIND = 1
_TRUNCATED = "The Core cut the read short before it finished."


class _Request(BaseModel):
    keys: list[channel.Key]
    bounds: TimeRange
    reduction: Reduction | None = None
    timestamps_included: bool = False


class Reader:
    """Reader pulls historical telemetry from the Core in a single request per read."""

    _retriever: Retriever
    _file: FileTransport
    _registry: Registry

    def __init__(
        self, *, retriever: Retriever, file_transport: FileTransport, registry: Registry
    ) -> None:
        self._retriever = retriever
        self._file = file_transport
        self._registry = registry

    def read(
        self,
        tr: TimeRange,
        channels: channel.Params,
        *,
        reduction: Reduction | None = None,
    ) -> Frame:
        """Reads every sample the given channels hold within tr.

        :param tr: the time range to read.
        :param channels: the channels to read, by key or by name. A frame read by name
            is keyed by name.
        :param reduction: how the Core reduces the samples of each channel. Defaults to
            keeping every sample.
        :returns: the samples as a frame.
        """
        adapter = ReadFrameAdapter(self._retriever)
        adapter.update(channels)
        req = _Request(keys=adapter.keys, bounds=tr, reduction=reduction)
        frame = Frame()
        with self._file.stream(_ENDPOINT, req, accept=_FRAME_CONTENT_TYPE) as body:
            for record in _records(body, self._registry):
                pld = adapter.codec.decode(record)
                frame.append(Frame(channels=pld.keys, series=pld.series))
        return adapter.adapt(frame)

    def read_csv(
        self,
        tr: TimeRange,
        channels: channel.Params,
        *,
        dest: FilePath,
        reduction: StrideReduction | None = None,
    ) -> None:
        """Reads every sample the given channels hold within tr into a CSV file.

        The Core writes the CSV. Channels are grouped by the index channel that
        timestamps them, and each group contributes its index column followed by one
        column per channel. Channels with no index are excluded.

        :param tr: the time range to read.
        :param channels: the channels to read, by key or by name.
        :param dest: the path of the CSV file to write. It must end in .csv.
        :param reduction: keeps every factor-th sample of each channel. Defaults to
            keeping every sample.
        """
        self._read_file(tr, channels, dest, reduction)

    def read_tdms(
        self,
        tr: TimeRange,
        channels: channel.Params,
        *,
        dest: FilePath,
        reduction: StrideReduction | None = None,
    ) -> None:
        """Reads every sample the given channels hold within tr into an NI TDMS file.

        The Core writes the file. Channels are grouped by the index channel that
        timestamps them, and each group holds its index channel followed by its other
        channels. Channels with no index are excluded.

        :param tr: the time range to read.
        :param channels: the channels to read, by key or by name.
        :param dest: the path of the TDMS file to write. It must end in .tdms.
        :param reduction: keeps every factor-th sample of each channel. Defaults to
            keeping every sample.
        """
        self._read_file(tr, channels, dest, reduction)

    def _read_file(
        self,
        tr: TimeRange,
        channels: channel.Params,
        dest: FilePath,
        reduction: StrideReduction | None,
    ) -> None:
        adapter = ReadFrameAdapter(self._retriever)
        adapter.update(channels)
        req = _Request(
            keys=adapter.keys,
            bounds=tr,
            reduction=reduction,
            timestamps_included=True,
        )
        self._file.download(_ENDPOINT, req, dest=dest)


def _records(body: Iterator[bytes], registry: Registry) -> Iterator[bytes]:
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
        with memoryview(buf) as view:
            payload = bytes(view[_HEADER_SIZE : _HEADER_SIZE + size])
        del buf[: _HEADER_SIZE + size]
        if kind != _TERMINATOR_KIND:
            yield payload
            continue
        if size == 0:
            return
        raise registry.decode(
            ExceptionPayload.model_validate_json(payload)
        ) or UnexpectedError(_TRUNCATED)
