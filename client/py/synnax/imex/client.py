#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import overload

from freighter import FileClient, FilePath
from synnax import ontology
from synnax.imex.types import Envelope

_EXPORT_PATH = "/imex/export"


class Client:
    """Imports and exports metadata resources to and from the Core.

    Each call moves exactly one envelope. Large payloads are streamed: passing a path to
    ``import_`` streams the file from disk, and passing ``dest`` to ``export`` streams
    the response into it as it arrives.
    """

    _client: FileClient

    def __init__(self, client: FileClient) -> None:
        self._client = client

    def import_(self, source: FilePath | Envelope) -> ontology.ID:
        """Imports the resource described by source and returns its new ontology id.

        :param source: an ``Envelope`` sent as a typed payload, or a file path streamed
            from disk.
        :returns: the new resource's ontology id as stamped by the Core.
        """
        return self._client.upload("/imex/import", source, ontology.ID)

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
            return self._client.download(_EXPORT_PATH, id, Envelope)
        self._client.download(_EXPORT_PATH, id, dest=dest)
        return None
