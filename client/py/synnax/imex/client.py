#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from freighter import FileClient, FilePath
from synnax import ontology

_IMPORT_PATH = "/imex/import"
_EXPORT_PATH = "/imex/export"


class Client:
    """Imports and exports metadata resources to and from the Core.

    Each call moves exactly one envelope, streamed to or from disk: ``import_`` streams
    a file from disk, and ``export`` streams the response straight into a destination
    file.
    """

    _file_client: FileClient

    def __init__(self, file_client: FileClient) -> None:
        self._file_client = file_client

    def import_(self, source: FilePath) -> ontology.ID:
        """Imports the resource at source and returns its new ontology id.

        :param source: a file path streamed from disk.
        :returns: the new resource's ontology id as stamped by the Core.
        """
        return self._file_client.upload(_IMPORT_PATH, source, ontology.ID)

    def export(self, id: ontology.ID, dest: FilePath) -> None:
        """Exports the resource identified by id, streaming it into dest.

        The response body is streamed straight into dest — the on-disk format is driven
        by the destination's extension.

        :param id: the ontology id of the resource to export.
        :param dest: the file path to stream into.
        """
        self._file_client.download(_EXPORT_PATH, id, dest)
