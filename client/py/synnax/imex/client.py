#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any

from pydantic import BaseModel

from freighter import DownloadClient, FilePath, UnaryClient, UploadClient
from synnax.imex.types import Envelope
from synnax.ontology.payload import ID

_IMPORT_PATH = "/imex/import"
_EXPORT_PATH = "/imex/export"


class _ImportResponse(BaseModel):
    key: str


class Client:
    """Imports and exports metadata resources to and from the cluster.

    Each call moves exactly one envelope. Large payloads are streamed: passing a path to
    ``import_`` streams the file from disk, and passing ``dest`` to ``export`` streams
    the response into it as it arrives.
    """

    _unary: UnaryClient
    _upload: UploadClient
    _download: DownloadClient

    def __init__(
        self, unary: UnaryClient, upload: UploadClient, download: DownloadClient
    ) -> None:
        self._unary = unary
        self._upload = upload
        self._download = download

    def import_(self, source: FilePath | Envelope | dict[str, Any]) -> str:
        """Imports the resource described by source and returns its new key.

        :param source: an ``Envelope`` or ``dict`` sent as a typed payload, or a file
            path streamed from disk.
        :returns: the new resource's key as stamped by the server.
        """
        if isinstance(source, dict):
            source = Envelope.model_validate(source)
        if isinstance(source, Envelope):
            res, err = self._unary.send(_IMPORT_PATH, source, _ImportResponse)
        else:
            res, err = self._upload.upload(_IMPORT_PATH, source, _ImportResponse)
        if err is not None:
            raise err
        assert res is not None
        return res.key

    def export(self, id: ID, *, dest: FilePath | None = None) -> Envelope | None:
        """Exports the resource identified by id.

        When ``dest`` is provided, the response body is streamed straight into that file
        path and the call returns None — the on-disk format is driven by the
        destination's extension. When ``dest`` is None, the response is decoded into an
        in-memory ``Envelope``.

        :param id: the ontology id of the resource to export.
        :param dest: optional file path to stream into.
        :returns: the parsed Envelope when ``dest`` is None; otherwise None.
        """
        if dest is None:
            res, err = self._unary.send(_EXPORT_PATH, id, Envelope)
            if err is not None:
                raise err
            assert res is not None
            return res
        err = self._download.download(_EXPORT_PATH, id, dest)
        if err is not None:
            raise err
        return None
