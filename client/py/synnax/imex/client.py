#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from freighter import FileTransport
from synnax import ontology
from x.fs import FilePath


class Client:
    """Imports and exports resources from and to the Core.

    Each call moves exactly one envelope, streamed to or from disk: ``import_`` streams
    a file from disk, and ``export`` streams the response straight into a destination
    file.
    """

    _file_transport: FileTransport

    def __init__(self, file_transport: FileTransport) -> None:
        self._file_transport = file_transport

    def import_(self, source: FilePath) -> ontology.ID:
        """Imports the resource at source and returns its new ontology ID.

        :param source: a file path streamed from disk.
        :returns: the new resource's ontology ID.
        """
        return self._file_transport.upload("/imex/import", source, ontology.ID)

    def export(self, id: ontology.ID, dest: FilePath) -> None:
        """Exports the resource identified by id, streaming it into dest.

        The response body is streamed straight into dest — the on-disk format is driven
        by the destination's extension.

        :param id: the ontology ID of the resource to export.
        :param dest: a file path to stream the response body into.
        """
        self._file_transport.download("/imex/export", id, dest)
