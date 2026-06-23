#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import overload

from freighter import FileClient, FilePath, UnaryClient
from synnax import ontology
from synnax.imex.types import Envelope

_IMPORT_PATH = "/imex/import"
_EXPORT_PATH = "/imex/export"


class Client:
    """Imports and exports metadata resources to and from the Core.

    Each call moves exactly one envelope. Large payloads are streamed: passing a path to
    ``import_`` streams the file from disk, and passing ``dest`` to ``export`` streams
    the response into it as it arrives. In-memory ``Envelope`` payloads are sent and
    received inline via the unary client.
    """

    _file_client: FileClient
    _unary_client: UnaryClient

    def __init__(self, file_client: FileClient, unary_client: UnaryClient) -> None:
        self._file_client = file_client
        self._unary_client = unary_client

    def import_(self, source: FilePath | Envelope) -> ontology.ID:
        """Imports the resource described by source and returns its new ontology id.

        :param source: an ``Envelope`` sent inline as a typed payload, or a file path
            streamed from disk.
        :returns: the new resource's ontology id as stamped by the Core.
        """
        if isinstance(source, Envelope):
            return self._unary_client.send(_IMPORT_PATH, source, ontology.ID)
        return self._file_client.upload(_IMPORT_PATH, source, ontology.ID)

    @overload
    def export(self, id: ontology.ID) -> Envelope: ...
    @overload
    def export(self, id: ontology.ID, *, dest: FilePath) -> None: ...
    def export(
        self, id: ontology.ID, *, dest: FilePath | None = None
    ) -> Envelope | None:
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
            return self._unary_client.send(_EXPORT_PATH, id, Envelope)
        self._file_client.download(_EXPORT_PATH, id, dest)
        return None
